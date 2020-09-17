package prometheus

import (
	"fmt"
	"github.com/pkg/errors"
	chubaoapi "github.com/rook/rook/pkg/apis/chubao.rook.io/v1alpha1"
	"github.com/rook/rook/pkg/clusterd"
	"github.com/rook/rook/pkg/operator/chubao/commons"
	"github.com/rook/rook/pkg/operator/chubao/constants"
	"github.com/rook/rook/pkg/operator/k8sutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilintstr "k8s.io/apimachinery/pkg/util/intstr"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/tools/record"
	"reflect"
)

//set below
var PrometheusServiceUrl string

const (
	// message
	MessagePrometheusCreated        = "Prometheus[%s] Deployment created"
	MessagePrometheusServiceCreated = "Prometheus[%s] Service created"

	// error message
	MessageCreatePrometheusServiceFailed = "Failed to create Prometheus[%s] Service"
	MessageCreatePrometheusFailed        = "Failed to create Prometheus[%s] Deployment"
	MessageUpdatePrometheusFailed        = "Failed to update Prometheus[%s] Deployment"

	instanceName = "prometheus"
	serviceName  = "prometheus-service"

	defaultPort  = 9090
	defaultImage = "prom/prometheus:v2.13.1"
)

type Prometheus struct {
	monitorObj          *chubaoapi.ChubaoMonitor
	prometheusObj       chubaoapi.PrometheusSpec
	context             *clusterd.Context
	kubeInformerFactory kubeinformers.SharedInformerFactory
	ownerRef            metav1.OwnerReference
	recorder            record.EventRecorder
	namespace           string
	port                int32
	image               string
	imagePullPolicy     corev1.PullPolicy
	hostPath            *corev1.HostPathVolumeSource
}

func New(
	context *clusterd.Context,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	recorder record.EventRecorder,
	monitorObj *chubaoapi.ChubaoMonitor,
	ownerRef metav1.OwnerReference) *Prometheus {
	prometheusObj := monitorObj.Spec.Prometheus
	return &Prometheus{
		context:             context,
		kubeInformerFactory: kubeInformerFactory,
		recorder:            recorder,
		monitorObj:          monitorObj,
		prometheusObj:       prometheusObj,
		ownerRef:            ownerRef,
		namespace:           monitorObj.Namespace,
		port:                commons.GetIntValue(prometheusObj.Port, defaultPort),
		image:               commons.GetStringValue(prometheusObj.Image, defaultImage),
		imagePullPolicy:     commons.GetImagePullPolicy(prometheusObj.ImagePullPolicy),
		hostPath:            commons.GetHostPath(prometheusObj.HostPath),
	}
}

func (prometheus *Prometheus) Deploy() error {
	clientset := prometheus.context.Clientset
	service := prometheus.newPrometheusService()
	serviceKey := fmt.Sprintf("%s/%s", service.Namespace, service.Name)

	if _, err := k8sutil.CreateOrUpdateService(clientset, prometheus.namespace, prometheus.newPrometheusService()); err != nil {

		prometheus.recorder.Eventf(prometheus.monitorObj, corev1.EventTypeWarning, constants.ErrCreateFailed, MessageCreatePrometheusServiceFailed, serviceKey)
		return errors.Wrapf(err, MessageCreatePrometheusServiceFailed, serviceKey)
	}
	prometheus.recorder.Eventf(prometheus.monitorObj, corev1.EventTypeNormal, constants.SuccessCreated, MessagePrometheusServiceCreated, serviceKey)

	deployment := prometheus.newPrometheusDeployment()
	deploymentKey := fmt.Sprintf("%s/%s", deployment.Namespace, deployment.Name)
	if _, err := clientset.AppsV1().Deployments(prometheus.namespace).Create(deployment); err != nil {
		if !k8serrors.IsAlreadyExists(err) {

			prometheus.recorder.Eventf(prometheus.monitorObj, corev1.EventTypeWarning, constants.ErrCreateFailed, MessageCreatePrometheusFailed, deploymentKey)
			return errors.Wrapf(err, MessageCreatePrometheusFailed, deploymentKey)
		}

		_, err := clientset.AppsV1().Deployments(prometheus.namespace).Update(deployment)
		if err != nil {
			prometheus.recorder.Eventf(prometheus.monitorObj, corev1.EventTypeWarning, constants.ErrUpdateFailed, MessageUpdatePrometheusFailed, deploymentKey)
			return errors.Wrapf(err, MessageUpdatePrometheusFailed, deploymentKey)
		}
	}
	prometheus.recorder.Eventf(prometheus.monitorObj, corev1.EventTypeNormal, constants.SuccessCreated, MessagePrometheusCreated, serviceKey)
	prometheus.getPrometheusUrl()
	return nil
}

func (prometheus *Prometheus) newPrometheusService() *corev1.Service {
	labels := prometheusLabel(prometheus.monitorObj.Name)
	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       reflect.TypeOf(corev1.Service{}).Name(),
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            serviceName,
			Namespace:       prometheus.namespace,
			OwnerReferences: []metav1.OwnerReference{prometheus.ownerRef},
			Labels:          labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "port",
					Port:       prometheus.port,
					TargetPort: utilintstr.IntOrString{IntVal: prometheus.port},
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: labels,
		},
	}
	return service
}

func (prometheus *Prometheus) newPrometheusDeployment() *appsv1.Deployment {
	labels := prometheusLabel(prometheus.monitorObj.Name)
	replicas := int32(1)
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       reflect.TypeOf(appsv1.Deployment{}).Name(),
			APIVersion: appsv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            instanceName,
			Namespace:       prometheus.namespace,
			OwnerReferences: []metav1.OwnerReference{prometheus.ownerRef},
			Labels:          labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: createPodSpec(prometheus),
			},
		},
	}

	return deployment
}

func createPodSpec(prometheus *Prometheus) corev1.PodSpec {
	privileged := true
	pod := corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:            "prometheus-pod",
				Image:           prometheus.image,
				ImagePullPolicy: prometheus.imagePullPolicy,
				SecurityContext: &corev1.SecurityContext{
					Privileged: &privileged,
				},
				Ports: []corev1.ContainerPort{
					{
						Name: "port", ContainerPort: prometheus.port, Protocol: corev1.ProtocolTCP,
					},
				},
				Resources: prometheus.prometheusObj.Resources,
				Env:       createEnv(prometheus),
				// If grafana pod show the err "back-off restarting failed container", run this command to keep the container running ang then run ./run.sh in the container to check real error.
				//          Command:        []string{"/bin/bash", "-ce", "tail -f /dev/null"},
				VolumeMounts: createVolumeMounts(prometheus),
			},
		},
		Volumes: createVolumes(prometheus),
	}

	return pod
}

func createVolumes(prometheus *Prometheus) []corev1.Volume {
	var defaultmode int32 = 0555

	return []corev1.Volume{
		{
			Name: "monitor-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "monitor-config",
					},
					DefaultMode: &defaultmode,
				},
			},
		},
		{
			Name: "prometheus-data",
			VolumeSource: corev1.VolumeSource{
				HostPath: prometheus.hostPath,
			},
		},
	}
}

func createVolumeMounts(prometheus *Prometheus) []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      "monitor-config",
			MountPath: "/etc/prometheus/prometheus.yml",
			SubPath:   "prometheus.yml",
		},
		{
			Name:      "prometheus-data",
			MountPath: "/prometheus-data",
		},
	}
}

func createEnv(prometheus *Prometheus) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "CONSUL_ADDRESS",
			Value: prometheus.prometheusObj.ConsulUrl,
		},
		{
			Name:  "TZ",
			Value: " Asia/Shanghai",
		},
	}
}

func prometheusLabel(monitorname string) map[string]string {
	return commons.LabelsForMonitor(constants.ComponentPrometheus, monitorname)
}

func (prometheus *Prometheus) getPrometheusUrl() {
	PrometheusServiceUrl = fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", serviceName, prometheus.namespace, prometheus.port)
}

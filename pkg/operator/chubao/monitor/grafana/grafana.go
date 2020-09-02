package grafana

import (
	"fmt"
	"github.com/pkg/errors"
	chubaoapi "github.com/rook/rook/pkg/apis/chubao.rook.io/v1alpha1"
	"github.com/rook/rook/pkg/clusterd"
	"github.com/rook/rook/pkg/operator/chubao/commons"
	"github.com/rook/rook/pkg/operator/chubao/monitor/prometheus"
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

const (
	InstanceName = "grafana"
	ServiceName  = "grafana-service"

	DefaultPassword = "!!string 123456"
	DefaultPort     = 3000
	DefaultImage    = "grafana/grafana:6.4.4"
)

var matchLabels = map[string]string{
	"application": "rook-chubao-operator",
	"component":   "grafana",
}

type Grafana struct {
	monitorObj          *chubaoapi.ChubaoMonitor
	grafanaObj          chubaoapi.GrafanaSpec
	context             *clusterd.Context
	kubeInformerFactory kubeinformers.SharedInformerFactory
	ownerRef            metav1.OwnerReference
	recorder            record.EventRecorder
	namespace           string
	port                int32
	image               string
	imagePullPolicy     corev1.PullPolicy
}

func New(
	context *clusterd.Context,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	recorder record.EventRecorder,
	monitorObj *chubaoapi.ChubaoMonitor,
	ownerRef metav1.OwnerReference) *Grafana {
	grafanaObj := monitorObj.Spec.Grafana
	return &Grafana{
		context:             context,
		kubeInformerFactory: kubeInformerFactory,
		recorder:            recorder,
		monitorObj:          monitorObj,
		grafanaObj:          grafanaObj,
		ownerRef:            ownerRef,
		namespace:           monitorObj.Namespace,
		port:                commons.GetIntValue(grafanaObj.PortGrafana, DefaultPort),
		image:               commons.GetStringValue(grafanaObj.ImageGrafana, DefaultImage),
		imagePullPolicy:     commons.GetImagePullPolicy(grafanaObj.ImagePullPolicyGrafana),
	}
}

func (grafana *Grafana) Deploy() error {
	clientset := grafana.context.Clientset
	if _, err := k8sutil.CreateOrUpdateService(clientset, grafana.namespace, grafana.newGrafanaService()); err != nil {
		return errors.Wrap(err, "failed to create Service for grafana")
	}

	deployment := grafana.newGrafanaDeployment()
	msg := fmt.Sprintf("%s/%s", deployment.Namespace, deployment.Name)
	if _, err := clientset.AppsV1().Deployments(grafana.namespace).Create(deployment); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return errors.Wrap(err, fmt.Sprintf("failed to create Deployment for grafana[%s]", msg))
		}

		_, err := clientset.AppsV1().Deployments(grafana.namespace).Update(deployment)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to update Deployment for grafana[%s]", msg))
		}
	}

	return nil
}

func (grafana *Grafana) newGrafanaService() *corev1.Service {
	labels := commons.GrafanaLabels(ServiceName, grafana.monitorObj.Name)
	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       reflect.TypeOf(corev1.Service{}).Name(),
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            ServiceName,
			Namespace:       grafana.namespace,
			OwnerReferences: []metav1.OwnerReference{grafana.ownerRef},
			Labels:          labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "port",
					Port:       grafana.port,
					TargetPort: utilintstr.IntOrString{IntVal: grafana.port, Type: utilintstr.Int},
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: matchLabels,
		},
	}
	return service
}

func (grafana *Grafana) newGrafanaDeployment() *appsv1.Deployment {
	labels := commons.GrafanaLabels(InstanceName, grafana.monitorObj.Name)
	replicas := int32(1)
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       reflect.TypeOf(appsv1.Deployment{}).Name(),
			APIVersion: appsv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            InstanceName,
			Namespace:       grafana.namespace,
			OwnerReferences: []metav1.OwnerReference{grafana.ownerRef},
			Labels:          matchLabels,
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
				Spec: createPodSpec(grafana),
			},
		},
	}

	return deployment
}

func createPodSpec(grafana *Grafana) corev1.PodSpec {
	privileged := true
	pod := corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:            "grafana-pod",
				Image:           grafana.image,
				ImagePullPolicy: grafana.imagePullPolicy,
				SecurityContext: &corev1.SecurityContext{
					Privileged: &privileged,
				},
				Ports: []corev1.ContainerPort{
					{
						Name: "port", ContainerPort: grafana.port, Protocol: corev1.ProtocolTCP,
					},
				},
				Resources: grafana.grafanaObj.ResourcesGrafana,
				Env:       createEnv(grafana),
				// If grafana pod show the err "back-off restarting failed container", run this command to keep the container running ang then run ./run.sh in the container to check the really error.
				//          Command:        []string{"/bin/bash", "-ce", "tail -f /dev/null"},
				ReadinessProbe: createReadinessProbe(grafana),
				VolumeMounts:   createVolumeMounts(grafana),
			},
		},
		Volumes: createVolumes(grafana),
	}

	return pod
}

func createVolumes(grafana *Grafana) []corev1.Volume {
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
		{Name: "grafana-persistent-storage",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}
}

func createVolumeMounts(grafana *Grafana) []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      "grafana-persistent-storage",
			MountPath: "/var/lib/grafana",
		},
		{
			Name:      "monitor-config",
			MountPath: "/grafana/init.sh",
			SubPath:   "init.sh",
		},
		{
			Name:      "monitor-config",
			MountPath: "/etc/grafana/grafana.ini",
			SubPath:   "grafana.ini",
		},
		{
			Name:      "monitor-config",
			MountPath: "/etc/grafana/provisioning/dashboards/chubaofs.json",
			SubPath:   "chubaofs.json",
		},
		{
			Name:      "monitor-config",
			MountPath: "/etc/grafana/provisioning/dashboards/dashboard.yml",
			SubPath:   "dashboard.yml",
		},
		{
			Name:      "monitor-config",
			MountPath: "/etc/grafana/provisioning/datasources/datasource.yml",
			SubPath:   "datasource.yml",
		},
	}
}

func (grafana *Grafana) getPrometheusUrl() string {
	return fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", prometheus.ServiceName, grafana.namespace, grafana.grafanaObj.PortGrafana)
}

func createEnv(grafana *Grafana) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "GF_AUTH_BASIC_ENABLED",
			Value: "true",
		},
		{
			Name:  "GF_AUTH_ANONYMOUS_ENABLED",
			Value: "false",
		},
		{
			Name:  "GF_SECURITY_ADMIN_PASSWORD",
			Value: "123456",
		},
		{
			Name:  "GRAFANA_PASSWORD",
			Value: commons.GetPassword(grafana.grafanaObj.Password, DefaultPassword),
		},
		{
			Name:  "PROMETHEUS_URL",
			Value: grafana.getPrometheusUrl(),
		},
	}
}

func createReadinessProbe(grafana *Grafana) *corev1.Probe {
	return &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/login",
				Port: utilintstr.IntOrString{
					IntVal: 3000,
				},
			},
		},
	}
}
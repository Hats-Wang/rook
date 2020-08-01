package master

import (
	"fmt"
	"github.com/coreos/pkg/capnslog"
	"github.com/pkg/errors"
	chubaoapi "github.com/rook/rook/pkg/apis/chubao.rook.io/v1alpha1"
	"github.com/rook/rook/pkg/clusterd"
	"github.com/rook/rook/pkg/operator/chubao/cluster/consul"
	"github.com/rook/rook/pkg/operator/chubao/commons"
	"github.com/rook/rook/pkg/operator/k8sutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/tools/record"
	"reflect"
	"strings"
)

var logger = capnslog.NewPackageLogger("github.com/rook/rook", "chubao-master")

var matchLabels = map[string]string{
	"application": "rook-chubao-operator",
	"component":   "chubao-master",
}

const (
	InstanceName               = "master"
	ServiceAccountName         = "rook-chubao-master"
	ServiceName                = "master-service"
	DefaultServerImage         = "chubaofs/cfs-server:0.0.1"
	DefaultDataDirHostPath     = "/var/lib/chubaofs"
	DefaultLogDirHostPath      = "/var/log/chubaofs"
	DefaultReplicas            = 3
	DefaultClusterName         = "rook-chubao-cluster"
	DefaultLogLevel            = "info"
	DefaultRetainLogs          = 2000
	DefaultPort                = 17010
	DefaultProf                = 17020
	DefaultExporterPort        = 9500
	DefaultMetanodeReservedMem = 67108864

	volumeNameForLogPath       = "pod-log-path"
	volumeNameForDataPath      = "pod-data-path"
	defaultDataPathInContainer = "/cfs/data"
	defaultLogPathInContainer  = "/cfs/logs"
	masterNodeLabelValue       = "enabled"
)

const (
	startMasterScript = `
set -ex
echo "start master"
/cfs/bin/cfs-server -f -c /cfs/conf/master.json
`
)

type Master struct {
	clusterObj          *chubaoapi.ChubaoCluster
	masterObj           chubaoapi.MasterSpec
	context             *clusterd.Context
	kubeInformerFactory kubeinformers.SharedInformerFactory
	ownerRef            metav1.OwnerReference
	recorder            record.EventRecorder
	namespace           string
	serverImage         string
	imagePullPolicy     corev1.PullPolicy
	dataDirHostPath     string
	logDirHostPath      string
	replicas            int32
	clusterName         string
	logLevel            string
	retainLogs          int32
	port                int32
	prof                int32
	exporterPort        int32
	metanodeReservedMem int64
}

func New(
	context *clusterd.Context,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	recorder record.EventRecorder,
	clusterObj *chubaoapi.ChubaoCluster,
	ownerRef metav1.OwnerReference) *Master {
	spec := clusterObj.Spec
	masterObj := spec.Master
	return &Master{
		context:             context,
		kubeInformerFactory: kubeInformerFactory,
		recorder:            recorder,
		clusterObj:          clusterObj,
		ownerRef:            ownerRef,
		namespace:           clusterObj.Namespace,
		masterObj:           masterObj,
		serverImage:         commons.GetStringValue(spec.CFSVersion.ServerImage, DefaultServerImage),
		imagePullPolicy:     commons.GetImagePullPolicy(spec.CFSVersion.ImagePullPolicy),
		dataDirHostPath:     commons.GetStringValue(spec.DataDirHostPath, DefaultDataDirHostPath),
		logDirHostPath:      commons.GetStringValue(spec.LogDirHostPath, DefaultLogDirHostPath),
		replicas:            commons.GetIntValue(masterObj.Replicas, DefaultReplicas),
		clusterName:         commons.GetStringValue(masterObj.Cluster, DefaultClusterName),
		logLevel:            commons.GetStringValue(masterObj.LogLevel, DefaultLogLevel),
		retainLogs:          commons.GetIntValue(masterObj.RetainLogs, DefaultRetainLogs),
		port:                commons.GetIntValue(masterObj.Port, DefaultPort),
		prof:                commons.GetIntValue(masterObj.Prof, DefaultProf),
		exporterPort:        commons.GetIntValue(masterObj.ExporterPort, DefaultExporterPort),
		metanodeReservedMem: commons.GetInt64Value(masterObj.MetaNodeReservedMem, DefaultMetanodeReservedMem),
	}
}

func (m *Master) Deploy() error {
	clientset := m.context.Clientset
	if _, err := k8sutil.CreateOrUpdateService(clientset, m.namespace, m.newMasterService()); err != nil {
		return errors.Wrap(err, "failed to create Service for master")
	}

	statefulSet := m.newMasterStatefulSet()
	msg := fmt.Sprintf("%s/%s", statefulSet.Namespace, statefulSet.Name)
	if _, err := clientset.AppsV1().StatefulSets(m.namespace).Create(statefulSet); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return errors.Wrap(err, fmt.Sprintf("failed to create StatefulSet for master[%s]", msg))
		}

		_, err := clientset.AppsV1().StatefulSets(m.namespace).Update(statefulSet)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to update StatefulSet for master[%s]", msg))
		}
	}

	return nil
}

func (m *Master) newMasterStatefulSet() *appsv1.StatefulSet {
	labels := commons.MasterLabels(InstanceName, m.clusterObj.Name)
	statefulSet := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       reflect.TypeOf(appsv1.StatefulSet{}).Name(),
			APIVersion: appsv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            InstanceName,
			Namespace:       m.namespace,
			OwnerReferences: []metav1.OwnerReference{m.ownerRef},
			Labels:          matchLabels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:            &m.replicas,
			ServiceName:         ServiceName,
			PodManagementPolicy: appsv1.OrderedReadyPodManagement,
			UpdateStrategy:      m.masterObj.UpdateStrategy,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: createPodSpec(m),
			},
		},
	}

	return statefulSet
}

func createPodSpec(m *Master) corev1.PodSpec {
	privileged := true
	nodeSelector := make(map[string]string)
	nodeSelector[fmt.Sprintf("%s-chubao-master", m.clusterObj.Namespace)] = "enabled"

	pathType := corev1.HostPathDirectoryOrCreate
	return corev1.PodSpec{
		NodeSelector: nodeSelector,
		HostNetwork:  true,
		HostPID:      true,
		DNSPolicy:    corev1.DNSClusterFirstWithHostNet,
		Containers: []corev1.Container{
			{
				Name:            "master-pod",
				Image:           m.serverImage,
				ImagePullPolicy: m.imagePullPolicy,
				SecurityContext: &corev1.SecurityContext{
					Privileged: &privileged,
				},
				Args: []string{
					"/bin/bash",
					"-c",
					"set -e",
					"/cfs/bin/start.sh master",
					"sleep 999999999d",
				},
				Env: []corev1.EnvVar{
					{Name: "CBFS_CLUSTER_NAME", Value: m.clusterName},
					{Name: "CBFS_PORT", Value: fmt.Sprintf("%d", m.port)},
					{Name: "CBFS_PROF", Value: fmt.Sprintf("%d", m.prof)},
					{Name: "CBFS_MASTER_PEERS", Value: m.getMasterPeers()},
					{Name: "CBFS_RETAIN_LOGS", Value: fmt.Sprintf("%d", m.retainLogs)},
					{Name: "CBFS_LOG_LEVEL", Value: m.logLevel},
					{Name: "CBFS_EXPORTER_PORT", Value: fmt.Sprintf("%d", m.exporterPort)},
					{Name: "CBFS_CONSUL_ADDR", Value: m.getConsulUrl()},
					{Name: "CBFS_METANODE_RESERVED_MEM", Value: fmt.Sprintf("%d", m.metanodeReservedMem)},
					k8sutil.PodIPEnvVar("POD_IP"),
					k8sutil.NameEnvVar(),
				},
				Ports: []corev1.ContainerPort{
					{Name: "port", ContainerPort: m.port, Protocol: corev1.ProtocolTCP},
					{Name: "prof", ContainerPort: m.prof, Protocol: corev1.ProtocolTCP},
					{Name: "exporter-port", ContainerPort: m.exporterPort, Protocol: corev1.ProtocolTCP},
				},
				VolumeMounts: []corev1.VolumeMount{
					{Name: volumeNameForLogPath, MountPath: defaultLogPathInContainer},
					{Name: volumeNameForDataPath, MountPath: defaultDataPathInContainer},
				},
				Resources: m.masterObj.Resource,
			},
		},
		Volumes: []corev1.Volume{
			{
				Name:         volumeNameForLogPath,
				VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: m.logDirHostPath, Type: &pathType}},
			},
			{
				Name:         volumeNameForDataPath,
				VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: m.dataDirHostPath, Type: &pathType}},
			},
		},
	}
}

func (m *Master) getConsulUrl() string {
	return fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", consul.ServiceName, m.namespace, m.clusterObj.Spec.Consul.Port)
}

func (m *Master) newMasterService() *corev1.Service {
	labels := commons.MasterLabels(ServiceName, m.clusterObj.Name)
	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       reflect.TypeOf(corev1.Service{}).Name(),
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            ServiceName,
			Namespace:       m.namespace,
			OwnerReferences: []metav1.OwnerReference{m.ownerRef},
			Labels:          labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "port", Port: m.port, Protocol: corev1.ProtocolTCP,
				},
			},
			Selector: matchLabels,
		},
	}
	return service
}

func (m *Master) getMasterPeers() string {
	urls := make([]string, 0)
	for i := 0; i < int(m.replicas); i++ {
		urls = append(urls, fmt.Sprintf("%d:%s-%d.%s.%s.svc.cluster.local:%d", i+1, InstanceName, i, ServiceName, m.namespace, m.port))
	}

	return strings.Join(urls, ",")
}

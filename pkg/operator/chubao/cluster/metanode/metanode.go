package metanode

import (
	"fmt"
	"github.com/pkg/errors"
	chubaoapi "github.com/rook/rook/pkg/apis/chubao.rook.io/v1alpha1"
	"github.com/rook/rook/pkg/clusterd"
	"github.com/rook/rook/pkg/operator/chubao/cluster/consul"
	"github.com/rook/rook/pkg/operator/chubao/cluster/master"
	"github.com/rook/rook/pkg/operator/chubao/commons"
	"github.com/rook/rook/pkg/operator/chubao/constants"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/tools/record"
	"reflect"
)

const (
	instanceName             = "metanode"
	defaultServerImage       = "chubaofs/cfs-server:0.0.1"
	defaultDataDirHostPath   = "/var/lib/chubaofs"
	defaultLogDirHostPath    = "/var/log/chubaofs"
	defaultLogLevel          = "error"
	defaultPort              = 17210
	defaultProf              = 17220
	defaultRaftHeartbeatPort = 17230
	defaultRaftReplicaPort   = 17240
	defaultExporterPort      = 17250
	defaultTotalMem          = 1073741824

	volumeNameForLogPath       = "pod-log-path"
	volumeNameForDataPath      = "pod-data-path"
	defaultDataPathInContainer = "/cfs/data"
	defaultLogPathInContainer  = "/cfs/logs"
)

type MetaNode struct {
	clusterObj          *chubaoapi.ChubaoCluster
	context             *clusterd.Context
	metaNodeObj         chubaoapi.MetaNodeSpec
	kubeInformerFactory kubeinformers.SharedInformerFactory
	ownerRef            metav1.OwnerReference
	recorder            record.EventRecorder
	namespace           string
	serverImage         string
	imagePullPolicy     corev1.PullPolicy
	dataDirHostPath     string
	logDirHostPath      string
	logLevel            string
	port                int32
	prof                int32
	raftHeartbeatPort   int32
	raftReplicaPort     int32
	exporterPort        int32
	totalMem            int64
}

func New(
	context *clusterd.Context,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	recorder record.EventRecorder,
	clusterObj *chubaoapi.ChubaoCluster,
	ownerRef metav1.OwnerReference) *MetaNode {
	spec := clusterObj.Spec
	metaNodeObj := spec.MetaNode
	return &MetaNode{
		context:             context,
		kubeInformerFactory: kubeInformerFactory,
		recorder:            recorder,
		clusterObj:          clusterObj,
		ownerRef:            ownerRef,
		namespace:           clusterObj.Namespace,
		metaNodeObj:         metaNodeObj,
		serverImage:         commons.GetStringValue(spec.CFSVersion.ServerImage, defaultServerImage),
		imagePullPolicy:     commons.GetImagePullPolicy(spec.CFSVersion.ImagePullPolicy),
		dataDirHostPath:     commons.GetStringValue(spec.DataDirHostPath, defaultDataDirHostPath),
		logDirHostPath:      commons.GetStringValue(spec.LogDirHostPath, defaultLogDirHostPath),
		logLevel:            commons.GetStringValue(metaNodeObj.LogLevel, defaultLogLevel),
		port:                commons.GetIntValue(metaNodeObj.Port, defaultPort),
		prof:                commons.GetIntValue(metaNodeObj.Prof, defaultProf),
		raftHeartbeatPort:   commons.GetIntValue(metaNodeObj.RaftHeartbeatPort, defaultRaftHeartbeatPort),
		raftReplicaPort:     commons.GetIntValue(metaNodeObj.RaftReplicaPort, defaultRaftReplicaPort),
		exporterPort:        commons.GetIntValue(metaNodeObj.ExporterPort, defaultExporterPort),
		totalMem:            commons.GetInt64Value(metaNodeObj.TotalMem, defaultTotalMem),
	}
}

func (mn *MetaNode) Deploy() error {
	labels := metaNodeLabels(mn.clusterObj.Name)
	clientSet := mn.context.Clientset
	daemonSet := mn.newMetaNodeDaemonSet(labels)
	msg := fmt.Sprintf("%s/%s", daemonSet.Namespace, daemonSet.Name)
	if _, err := clientSet.AppsV1().DaemonSets(mn.namespace).Create(daemonSet); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return errors.Wrap(err, fmt.Sprintf("failed to create DaemonSets for metanode[%s]", msg))
		}

		_, err := clientSet.AppsV1().DaemonSets(mn.namespace).Update(daemonSet)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to update DaemonSets for metanode[%s]", msg))
		}
	}
	return nil
}

func metaNodeLabels(clusterName string) map[string]string {
	return commons.CommonLabels(constants.ComponentMetaNode, clusterName)
}

func (mn *MetaNode) newMetaNodeDaemonSet(labels map[string]string) *appsv1.DaemonSet {
	daemonSet := &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       reflect.TypeOf(appsv1.DaemonSet{}).Name(),
			APIVersion: appsv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            instanceName,
			Namespace:       mn.namespace,
			OwnerReferences: []metav1.OwnerReference{mn.ownerRef},
			Labels:          labels,
		},
		Spec: appsv1.DaemonSetSpec{
			UpdateStrategy: mn.metaNodeObj.UpdateStrategy,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: createPodSpec(mn),
			},
		},
	}

	return daemonSet
}

func createPodSpec(mn *MetaNode) corev1.PodSpec {
	privileged := true
	nodeSelector := make(map[string]string)
	nodeSelector[fmt.Sprintf("%s-%s", mn.namespace, constants.ComponentMetaNode)] = "enabled"
	pathType := corev1.HostPathDirectoryOrCreate
	return corev1.PodSpec{
		NodeSelector: nodeSelector,
		HostNetwork:  true,
		HostPID:      true,
		DNSPolicy:    corev1.DNSClusterFirstWithHostNet,
		Containers: []corev1.Container{
			{
				Name:            "metanode-pod",
				Image:           mn.serverImage,
				ImagePullPolicy: mn.imagePullPolicy,
				SecurityContext: &corev1.SecurityContext{
					Privileged: &privileged,
				},
				Command: []string{
					"/bin/bash",
				},
				Args: []string{
					"-c",
					"set -e; /cfs/bin/start.sh metanode; sleep 999999999d",
				},
				Env: []corev1.EnvVar{
					{Name: "CBFS_PORT", Value: fmt.Sprintf("%d", mn.port)},
					{Name: "CBFS_PROF", Value: fmt.Sprintf("%d", mn.prof)},
					{Name: "CBFS_RAFT_HEARTBEAT_PORT", Value: fmt.Sprintf("%d", mn.raftHeartbeatPort)},
					{Name: "CBFS_RAFT_REPLICA_PORT", Value: fmt.Sprintf("%d", mn.raftReplicaPort)},
					{Name: "CBFS_EXPORTER_PORT", Value: fmt.Sprintf("%d", mn.exporterPort)},
					{Name: "CBFS_MASTER_ADDRS", Value: master.GetMasterAddr(mn.clusterObj)},
					{Name: "CBFS_LOG_LEVEL", Value: mn.logLevel},
					{Name: "CBFS_CONSUL_ADDR", Value: consul.GetConsulUrl(mn.clusterObj)},
				},
				Ports: []corev1.ContainerPort{
					// Port Name must be no more than 15 characters
					{Name: "port", ContainerPort: mn.port, Protocol: corev1.ProtocolTCP},
					{Name: "prof", ContainerPort: mn.prof, Protocol: corev1.ProtocolTCP},
					{Name: "heartbeat-port", ContainerPort: mn.raftHeartbeatPort, Protocol: corev1.ProtocolTCP},
					{Name: "replica-port", ContainerPort: mn.raftReplicaPort, Protocol: corev1.ProtocolTCP},
					{Name: "exporter-port", ContainerPort: mn.exporterPort, Protocol: corev1.ProtocolTCP},
				},
				VolumeMounts: []corev1.VolumeMount{
					{Name: volumeNameForLogPath, MountPath: defaultLogPathInContainer},
					{Name: volumeNameForDataPath, MountPath: defaultDataPathInContainer},
				},
				Resources: mn.metaNodeObj.Resource,
			},
		},
		Volumes: []corev1.Volume{
			{
				Name:         volumeNameForDataPath,
				VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: mn.dataDirHostPath, Type: &pathType}},
			},
			{
				Name:         volumeNameForLogPath,
				VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: mn.logDirHostPath, Type: &pathType}},
			},
		},
	}
}

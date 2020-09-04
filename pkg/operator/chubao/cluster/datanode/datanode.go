package datanode

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
	"strings"
)

const (
	instanceName             = "datanode"
	defaultServerImage       = "chubaofs/cfs-server:0.0.1"
	defaultDataDirHostPath   = "/var/lib/chubaofs"
	defaultLogDirHostPath    = "/var/log/chubaofs"
	defaultLogLevel          = "error"
	defaultPort              = 17310
	defaultProf              = 17320
	defaultRaftHeartbeatPort = 17330
	defaultRaftReplicaPort   = 17340
	defaultExporterPort      = 17350

	volumeNameForLogPath       = "pod-log-path"
	volumeNameForDataPath      = "pod-data-path"
	defaultDataPathInContainer = "/cfs/data"
	defaultLogPathInContainer  = "/cfs/logs"
)

type DataNode struct {
	clusterObj          *chubaoapi.ChubaoCluster
	context             *clusterd.Context
	dataNodeObj         chubaoapi.DataNodeSpec
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
	disks               []string
	zone                string
}

func New(
	context *clusterd.Context,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	recorder record.EventRecorder,
	clusterObj *chubaoapi.ChubaoCluster,
	ownerRef metav1.OwnerReference) *DataNode {
	spec := clusterObj.Spec
	dataNodeObj := spec.DataNode
	return &DataNode{
		context:             context,
		kubeInformerFactory: kubeInformerFactory,
		recorder:            recorder,
		clusterObj:          clusterObj,
		ownerRef:            ownerRef,
		namespace:           clusterObj.Namespace,
		dataNodeObj:         dataNodeObj,
		serverImage:         commons.GetStringValue(spec.CFSVersion.ServerImage, defaultServerImage),
		imagePullPolicy:     commons.GetImagePullPolicy(spec.CFSVersion.ImagePullPolicy),
		dataDirHostPath:     commons.GetStringValue(spec.LogDirHostPath, defaultDataDirHostPath),
		logDirHostPath:      commons.GetStringValue(spec.LogDirHostPath, defaultLogDirHostPath),
		logLevel:            commons.GetStringValue(dataNodeObj.LogLevel, defaultLogLevel),
		port:                commons.GetIntValue(dataNodeObj.Port, defaultPort),
		prof:                commons.GetIntValue(dataNodeObj.Prof, defaultProf),
		raftHeartbeatPort:   commons.GetIntValue(dataNodeObj.RaftHeartbeatPort, defaultRaftHeartbeatPort),
		raftReplicaPort:     commons.GetIntValue(dataNodeObj.RaftReplicaPort, defaultRaftReplicaPort),
		exporterPort:        commons.GetIntValue(dataNodeObj.ExporterPort, defaultExporterPort),
		disks:               dataNodeObj.Disks,
		zone:                dataNodeObj.Zone,
	}
}

func (dn *DataNode) Deploy() error {
	labels := dataNodeLabels(dn.clusterObj.Name)
	clientSet := dn.context.Clientset
	daemonSet := dn.newDataNodeDaemonSet(labels)
	msg := fmt.Sprintf("%s/%s", daemonSet.Namespace, daemonSet.Name)
	if _, err := clientSet.AppsV1().DaemonSets(dn.namespace).Create(daemonSet); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return errors.Wrap(err, fmt.Sprintf("failed to create DaemonSets for datanode[%s]", msg))
		}

		_, err := clientSet.AppsV1().DaemonSets(dn.namespace).Update(daemonSet)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to update DaemonSets for datanode[%s]", msg))
		}
	}
	return nil
}

func dataNodeLabels(clusterName string) map[string]string {
	return commons.CommonLabels(constants.ComponentDataNode, clusterName)
}

func (dn *DataNode) newDataNodeDaemonSet(labels map[string]string) *appsv1.DaemonSet {
	daemonSet := &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       reflect.TypeOf(appsv1.DaemonSet{}).Name(),
			APIVersion: appsv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            instanceName,
			Namespace:       dn.namespace,
			OwnerReferences: []metav1.OwnerReference{dn.ownerRef},
			Labels:          labels,
		},
		Spec: appsv1.DaemonSetSpec{
			UpdateStrategy: dn.dataNodeObj.UpdateStrategy,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: createPodSpec(dn),
			},
		},
	}

	return daemonSet
}

func createPodSpec(dn *DataNode) corev1.PodSpec {
	privileged := true
	nodeSelector := make(map[string]string)
	nodeSelector[fmt.Sprintf("%s-%s", dn.namespace, constants.ComponentDataNode)] = "enabled"
	pathType := corev1.HostPathDirectoryOrCreate
	pod := corev1.PodSpec{
		NodeSelector: nodeSelector,
		HostNetwork:  true,
		HostPID:      true,
		DNSPolicy:    corev1.DNSClusterFirstWithHostNet,
		Containers: []corev1.Container{
			{
				Name:            "datanode-pod",
				Image:           dn.serverImage,
				ImagePullPolicy: dn.imagePullPolicy,
				SecurityContext: &corev1.SecurityContext{
					Privileged: &privileged,
				},
				Command: []string{
					"/bin/bash",
				},
				Args: []string{
					"-c",
					"set -e; /cfs/bin/start.sh datanode; sleep 999999999d",
				},
				Env: []corev1.EnvVar{
					{Name: "CBFS_PORT", Value: fmt.Sprintf("%d", dn.port)},
					{Name: "CBFS_PROF", Value: fmt.Sprintf("%d", dn.prof)},
					{Name: "CBFS_RAFT_HEARTBEAT_PORT", Value: fmt.Sprintf("%d", dn.raftHeartbeatPort)},
					{Name: "CBFS_RAFT_REPLICA_PORT", Value: fmt.Sprintf("%d", dn.raftReplicaPort)},
					{Name: "CBFS_EXPORTER_PORT", Value: fmt.Sprintf("%d", dn.exporterPort)},
					{Name: "CBFS_MASTER_ADDRS", Value: master.GetMasterAddr(dn.clusterObj)},
					{Name: "CBFS_LOG_LEVEL", Value: dn.logLevel},
					{Name: "CBFS_CONSUL_ADDR", Value: consul.GetConsulUrl(dn.clusterObj)},
					{Name: "CBFS_DISKS", Value: strings.Join(dn.disks, ",")},
					{Name: "CBFS_ZONE", Value: dn.zone},
				},
				Ports: []corev1.ContainerPort{
					// Port Name must be no more than 15 characters
					{Name: "port", ContainerPort: dn.port, Protocol: corev1.ProtocolTCP},
					{Name: "prof", ContainerPort: dn.prof, Protocol: corev1.ProtocolTCP},
					{Name: "heartbeat-port", ContainerPort: dn.raftHeartbeatPort, Protocol: corev1.ProtocolTCP},
					{Name: "replica-port", ContainerPort: dn.raftReplicaPort, Protocol: corev1.ProtocolTCP},
					{Name: "exporter-port", ContainerPort: dn.exporterPort, Protocol: corev1.ProtocolTCP},
				},
				VolumeMounts: []corev1.VolumeMount{
					{Name: volumeNameForLogPath, MountPath: defaultLogPathInContainer},
					{Name: volumeNameForDataPath, MountPath: defaultDataPathInContainer},
				},
				Resources: dn.dataNodeObj.Resource,
			},
		},
		Volumes: []corev1.Volume{
			{
				Name:         volumeNameForDataPath,
				VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: dn.dataDirHostPath, Type: &pathType}},
			},
			{
				Name:         volumeNameForLogPath,
				VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: dn.logDirHostPath, Type: &pathType}},
			},
		},
	}

	setVolume(dn, &pod)
	return pod
}

func setVolume(dn *DataNode, pod *corev1.PodSpec) {
	pathType := corev1.HostPathDirectoryOrCreate
	for i, diskAndRetainSize := range dn.disks {
		arr := strings.Split(diskAndRetainSize, ":")
		disk := arr[0]
		name := fmt.Sprintf("disk-%d", i)
		vol := corev1.Volume{
			Name:         name,
			VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: disk, Type: &pathType}},
		}

		volMount := corev1.VolumeMount{Name: name, MountPath: disk}
		pod.Volumes = append(pod.Volumes, vol)
		pod.Containers[0].VolumeMounts = append(pod.Containers[0].VolumeMounts, volMount)
	}
}

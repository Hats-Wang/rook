/*
Copyright 2019 The Rook Authors. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package v1alpha1

import (
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ***************************************************************************
// IMPORTANT FOR CODE GENERATION
// If the types in this file are updated, you will need to run
// `make codegen` to generate the new types under the client/clientset folder.
// ***************************************************************************

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ChubaoCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              ClusterSpec   `json:"spec"`
	Status            ClusterStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ChubaoClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ChubaoCluster `json:"items"`
}

type ConditionType string

const (
	ConditionIgnored     ConditionType = "Ignored"
	ConditionConnecting  ConditionType = "Connecting"
	ConditionConnected   ConditionType = "Connected"
	ConditionProgressing ConditionType = "Progressing"
	ConditionReady       ConditionType = "Ready"
	ConditionUpdating    ConditionType = "Updating"
	ConditionFailure     ConditionType = "Failure"
	ConditionUpgrading   ConditionType = "Upgrading"
	ConditionDeleting    ConditionType = "Deleting"
)

type ClusterState string

const (
	ClusterStateCreating   ClusterState = "Creating"
	ClusterStateCreated    ClusterState = "Created"
	ClusterStateUpdating   ClusterState = "Updating"
	ClusterStateConnecting ClusterState = "Connecting"
	ClusterStateConnected  ClusterState = "Connected"
	ClusterStateError      ClusterState = "Error"
)

type CleanupPolicy string

const (
	CleanupPolicyDeleteLog        CleanupPolicy = "DeleteLog"
	CleanupPolicyDeleteData       CleanupPolicy = "DeleteData"
	CleanupPolicyDeleteDataAndLog CleanupPolicy = "DeleteDataAndLog"
	CleanupPolicyDeleteDiskData   CleanupPolicy = "DeleteDiskData"
	CleanupPolicyDeleteAll        CleanupPolicy = "DeleteAll"
)

type ClusterStatus struct {
	State        ClusterState  `json:"state,omitempty"`
	Phase        ConditionType `json:"phase,omitempty"`
	Message      string        `json:"message,omitempty"`
	Conditions   []Condition   `json:"conditions,omitempty"`
	ChubaoStatus *ChubaoStatus `json:"chubao,omitempty"`
}

type Condition struct {
	Type               ConditionType      `json:"type,omitempty"`
	Status             v1.ConditionStatus `json:"status,omitempty"`
	Reason             string             `json:"reason,omitempty"`
	Message            string             `json:"message,omitempty"`
	LastHeartbeatTime  metav1.Time        `json:"lastHeartbeatTime,omitempty"`
	LastTransitionTime metav1.Time        `json:"lastTransitionTime,omitempty"`
}

type ClusterSpec struct {
	CFSVersion      CFSVersionSpec `json:"cfsVersion,omitempty"`
	DataDirHostPath string         `json:"dataDirHostPath"`
	LogDirHostPath  string         `json:"logDirHostPath"`
	Master          MasterSpec     `json:"master,omitempty"`
	MetaNode        MetaNodeSpec   `json:"metaNode,omitempty"`
	DataNode        DataNodeSpec   `json:"dataNode,omitempty"`
	Consul          ConsulSpec     `json:"consul,omitempty"`

	// Indicates user intent when deleting a cluster; blocks orchestration and should not be set if cluster
	// deletion is not imminent.
	CleanupPolicy CleanupPolicy `json:"cleanupPolicy,omitempty"`
}

type ConsulSpec struct {
	Port            int32                   `json:"port,omitempty"`
	Image           string                  `json:"image,omitempty"`
	ImagePullPolicy v1.PullPolicy           `json:"imagePullPolicy,omitempty"`
	Resources       v1.ResourceRequirements `json:"resources,omitempty"`
}

type ChubaoStatus struct {
	Health         string                         `json:"health,omitempty"`
	Details        map[string]ChubaoHealthMessage `json:"details,omitempty"`
	LastChecked    string                         `json:"lastChecked,omitempty"`
	LastChanged    string                         `json:"lastChanged,omitempty"`
	PreviousHealth string                         `json:"previousHealth,omitempty"`
}

type ChubaoHealthMessage struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// VersionSpec represents the settings for the cfs-server version that Rook is orchestrating.
type CFSVersionSpec struct {
	// Docker image name for ChubaoFS Server.
	// More info: https://kubernetes.io/docs/concepts/containers/images
	// This field is optional to allow higher level config management to default or override
	// container images in workload controllers like Deployments and StatefulSets.
	// +optional
	ServerImage string `json:"serverImage"`

	// Image pull policy.
	// One of Always, Never, IfNotPresent.
	// Defaults to Always if :latest tag is specified, or IfNotPresent otherwise.
	// Cannot be updated.
	// +optional
	ImagePullPolicy v1.PullPolicy `json:"imagePullPolicy,omitempty" protobuf:"bytes,14,opt,name=imagePullPolicy,casttype=PullPolicy"`
}

type DataNodeSpec struct {
	LogLevel          string                         `json:"logLevel,omitempty"`
	Port              int32                          `json:"port,omitempty"`
	Prof              int32                          `json:"prof,omitempty"`
	ExporterPort      int32                          `json:"exporterPort,omitempty"`
	RaftHeartbeatPort int32                          `json:"raftHeartbeatPort,omitempty"`
	RaftReplicaPort   int32                          `json:"raftReplicaPort,omitempty"`
	Disks             []string                       `json:"disks"`
	Zone              string                         `json:"zone,omitempty"`
	UpdateStrategy    appsv1.DaemonSetUpdateStrategy `json:"updateStrategy,omitempty"`
	Resource          v1.ResourceRequirements        `json:"resource,omitempty"`
}

type MetaNodeSpec struct {
	LogLevel          string                         `json:"logLevel,omitempty"`
	TotalMem          int64                          `json:"totalMem,omitempty"`
	Port              int32                          `json:"port,omitempty"`
	Prof              int32                          `json:"prof,omitempty"`
	ExporterPort      int32                          `json:"exporterPort,omitempty"`
	RaftHeartbeatPort int32                          `json:"raftHeartbeatPort,omitempty"`
	RaftReplicaPort   int32                          `json:"raftReplicaPort,omitempty"`
	Zone              string                         `json:"zone,omitempty"`
	UpdateStrategy    appsv1.DaemonSetUpdateStrategy `json:"updateStrategy,omitempty"`
	Resource          v1.ResourceRequirements        `json:"resource,omitempty"`
}

type MasterSpec struct {
	Replicas            int32                            `json:"replicas,omitempty"`
	LogLevel            string                           `json:"logLevel,omitempty"`
	RetainLogs          int32                            `json:"retainLogs,omitempty"`
	Port                int32                            `json:"port,omitempty"`
	Prof                int32                            `json:"prof,omitempty"`
	ExporterPort        int32                            `json:"exporterPort,omitempty"`
	MetaNodeReservedMem int64                            `json:"metaNodeReservedMem,omitempty"`
	UpdateStrategy      appsv1.StatefulSetUpdateStrategy `json:"updateStrategy,omitempty"`
	Resource            v1.ResourceRequirements          `json:"resource,omitempty"`
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ChubaoMonitor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              MonitorSpec   `json:"spec"`
	Status            MonitorStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ChubaoMonitorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ChubaoMonitor `json:"items"`
}

type MonitorSpec struct {
	ConsulUrl string `json:"consulUrl"`
	Password  string `json:"password"`
}

type GrafanaStatus string

const (
	GrafanaStatusReady   GrafanaStatus = "Ready"
	GrafanaStatusFailure GrafanaStatus = "Failure"
	GrafanaStatusUnknown GrafanaStatus = "Unknown"
)

type PrometheusStatus string

const (
	PrometheusStatusReady   PrometheusStatus = "Ready"
	PrometheusStatusFailure PrometheusStatus = "Failure"
	PrometheusStatusUnknown PrometheusStatus = "Unknown"
)

type MonitorStatus struct {
	Grafana    GrafanaStatus    `json:"grafana"`
	Prometheus PrometheusStatus `json:"prometheus"`
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ChubaoObjectStore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              ObjectStoreSpec   `json:"spec"`
	Status            ObjectStoreStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ChubaoObjectStoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ChubaoMonitor `json:"items"`
}

type ObjectStoreSpec struct {
	Replicas     int32                   `json:"replicas,omitempty"`
	MasterAddr   string                  `json:"masterAddr"`
	LogLevel     string                  `json:"logLevel,omitempty"`
	Port         int32                   `json:"port,omitempty"`
	Prof         int32                   `json:"prof,omitempty"`
	ExporterPort int32                   `json:"exporterPort,omitempty"`
	Region       string                  `json:"region"`
	Domains      string                  `json:"domains,omitempty"`
	Host         string                  `json:"host,omitempty"`
	Resource     v1.ResourceRequirements `json:"resource,omitempty"`
}

type ObjectStoreStatus struct {
}

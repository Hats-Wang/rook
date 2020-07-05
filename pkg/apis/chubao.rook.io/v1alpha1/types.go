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

type ClusterStatus struct {
	State       ClusterState    `json:"state,omitempty"`
	Phase       ConditionType   `json:"phase,omitempty"`
	Message     string          `json:"message,omitempty"`
	Conditions  []Condition     `json:"conditions,omitempty"`
	ChubaoStatus  *ChubaoStatus     `json:"chubao,omitempty"`
	ChubaoVersion *ClusterVersion `json:"version,omitempty"`
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
	DataDirHostPath string         `json:"dataDirHostPath,omitempty"`
	LogDirHostPath  string         `json:"logDirHostPath,omitempty"`
	Master          MasterSpec     `json:"master"`
	MetaNode        MetaNodeSpec   `json:"metaNode"`
	DataNode        DataNodeSpec   `json:"dataNode"`
}


type ChubaoStatus struct {
	Health         string                       `json:"health,omitempty"`
	Details        map[string]ChubaoHealthMessage `json:"details,omitempty"`
	LastChecked    string                       `json:"lastChecked,omitempty"`
	LastChanged    string                       `json:"lastChanged,omitempty"`
	PreviousHealth string                       `json:"previousHealth,omitempty"`
}

type ClusterVersion struct {
	Image   string `json:"image,omitempty"`
	Version string `json:"version,omitempty"`
}

type ChubaoHealthMessage struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// VersionSpec represents the settings for the cfs-server version that Rook is orchestrating.
type CFSVersionSpec struct {
	ServerImage string `json:"serverImage,omitempty"`
	ClientImage string `json:"clientImage,omitempty"`
}

type DataNodeSpec struct {
	LogLevel      string                  `json:"logLevel,omitempty"`
	Port          int32                   `json:"port,omitempty"`
	Prof          int32                   `json:"prof,omitempty"`
	ExporterPort  int32                   `json:"exporterPort,omitempty"`
	RaftHeartbeat int32                   `json:"raftHeartbeat,omitempty"`
	RaftReplica   int32                   `json:"raftReplica,omitempty"`
	Disks         []string                `json:"disks,omitempty"`
	ZoneName      string                  `json:"zoneName,omitempty"`
	NodeSelector  v1.NodeSelector         `json:"nodeSelector,omitempty"`
	Resource      v1.ResourceRequirements `json:"resource,omitempty"`
}

type MetaNodeSpec struct {
	LogLevel      string                  `json:"logLevel,omitempty"`
	TotalMem      int64                   `json:"totalMem,omitempty"`
	Port          int32                   `json:"port,omitempty"`
	Prof          int32                   `json:"prof,omitempty"`
	ExporterPort  int32                   `json:"exporterPort,omitempty"`
	RaftHeartbeat int32                   `json:"raftHeartbeat,omitempty"`
	RaftReplica   int32                   `json:"raftReplica,omitempty"`
	ZoneName      string                  `json:"zoneName,omitempty"`
	NodeSelector  v1.NodeSelector         `json:"nodeSelector,omitempty"`
	Resource      v1.ResourceRequirements `json:"resource,omitempty"`
}

type MasterSpec struct {
	Replicas            int32                   `json:"replicas,omitempty"`
	Cluster             string                  `json:"cluster,omitempty"`
	LogLevel            string                  `json:"logLevel,omitempty"`
	RetainLogs          int32                   `json:"retainLogs,omitempty"`
	Port                int32                   `json:"port,omitempty"`
	Prof                int32                   `json:"prof,omitempty"`
	ExporterPort        int32                   `json:"exporterPort,omitempty"`
	MetaNodeReservedMem int32                   `json:"metaNodeReservedMem,omitempty"`
	NodeSelector        v1.NodeSelector         `json:"nodeSelector,omitempty"`
	Resource            v1.ResourceRequirements `json:"resource,omitempty"`
}

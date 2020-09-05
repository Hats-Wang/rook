package commons

import (
	"github.com/rook/rook/pkg/apis/chubao.rook.io/v1alpha1"
	"github.com/rook/rook/pkg/operator/chubao/constants"
	"reflect"
)

func recommendedLabels() map[string]string {
	return map[string]string{
		"app":                          constants.AppName,
		"app.kubernetes.io/name":       constants.AppName,
		"app.kubernetes.io/managed-by": constants.OperatorAppName,
	}
}

func CommonLabels(component, clusterName string) map[string]string {
	labels := recommendedLabels()
	labels[constants.ComponentLabel] = component
	labels[constants.ManagedByLabel] = reflect.TypeOf(v1alpha1.ChubaoCluster{}).Name()
	labels[constants.ClusterNameLabel] = clusterName
	return labels
}

func LabelsForMonitor(component, monitorName string) map[string]string {
	labels := recommendedLabels()
	labels[constants.ComponentLabel] = component
	labels[constants.ManagedByLabel] = reflect.TypeOf(v1alpha1.ChubaoMonitor{}).Name()
	labels[constants.MonitorNameLabel] = monitorName
	return labels
}

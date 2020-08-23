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

func MasterLabels(instance, clusterName string) map[string]string {
	return commonLabels(constants.ComponentMaster, instance, clusterName)
}

func ConsulLabels(instance, clusterName string) map[string]string {
	return commonLabels(constants.ComponentConsul, instance, clusterName)
}

func GrafanaLabels(instance, monitorName string) map[string]string {
	return labelsForMonitor(constants.ComponentGrafana, instance, monitorName)
}

func commonLabels(component, instance, clusterName string) map[string]string {
	labels := recommendedLabels()
	labels[constants.ComponentLabel] = component
	labels[constants.InstanceLabel] = instance
	labels[constants.ManagedByLabel] = reflect.TypeOf(v1alpha1.ChubaoCluster{}).Name()
	labels[constants.ClusterNameLabel] = clusterName
	return labels
}

func labelsForMonitor(component, instance, monitorName string) map[string]string {
	labels := recommendedLabels()
	labels[constants.ComponentLabel] = component
	labels[constants.InstanceLabel] = instance
	labels[constants.ManagedByLabel] = reflect.TypeOf(v1alpha1.ChubaoMonitor{}).Name()
	labels[constants.MonitorNameLabel] = monitorName
	return labels
}

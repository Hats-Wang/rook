package commons

import (
	v1 "k8s.io/api/core/v1"
)

func GetInt64Value(value int64, defalutValue int64) int64 {
	if value != 0 {
		return value
	}

	return defalutValue
}

func GetIntValue(value int32, defalutValue int32) int32 {
	if value != 0 {
		return value
	}

	return defalutValue
}

func GetStringValue(value string, defalutValue string) string {
	if value != "" {
		return value
	}

	return defalutValue
}

func GetImagePullPolicy(policy v1.PullPolicy) v1.PullPolicy {
	if policy != "" {
		return policy
	}

	return v1.PullIfNotPresent
}

func GetHostPath(hostPath *v1.HostPathVolumeSource) *v1.HostPathVolumeSource {
	if hostPath.Path != "" {
		return hostPath
	}

	return &v1.HostPathVolumeSource{Path: "/prometheus-data"}
}

func GetPassword(Custom, Default string) string {
	if Custom != "" {
		return Custom
	}
	return Default
}

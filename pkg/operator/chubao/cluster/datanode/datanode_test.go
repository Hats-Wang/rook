package datanode

import (
	corev1 "k8s.io/api/core/v1"
	"testing"
)

func Test_setVolume(t *testing.T) {
	pathType := corev1.HostPathDirectoryOrCreate

	type args struct {
		dn  *DataNode
		pod corev1.PodSpec
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "test-1",
			args: args{
				dn: &DataNode{
					disks: []string{
						"/data0:52428800",
						"/data1:52428800",
					},
				},
				pod: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							VolumeMounts: []corev1.VolumeMount{
								{Name: volumeNameForLogPath, MountPath: defaultLogPathInContainer},
								{Name: volumeNameForDataPath, MountPath: defaultDataPathInContainer},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name:         volumeNameForDataPath,
							VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/var/lib/chubao", Type: &pathType}},
						},
						{
							Name:         volumeNameForLogPath,
							VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/var/log/chubao", Type: &pathType}},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setVolume(tt.args.dn, &tt.args.pod)
			if len(tt.args.pod.Volumes) != 4 || len(tt.args.pod.Containers[0].VolumeMounts) != 4 {
				t.Fatal("set volume fail")
			}
		})
	}
}

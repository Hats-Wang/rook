package master

import (
	chubaoapi "github.com/rook/rook/pkg/apis/chubao.rook.io/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"testing"
)

func TestMaster_getMasterPeers(t *testing.T) {
	m := &Master{
		replicas:  3,
		namespace: "test-namespace",
		port:      9000,
	}

	peers := m.getMasterPeers()
	assert.Equal(t, "1:master-0.master-service.test-namespace.svc.cluster.local:9000,2:master-1.master-service.test-namespace.svc.cluster.local:9000,3:master-2.master-service.test-namespace.svc.cluster.local:9000", peers)
}

func TestGetMasterAddrs(t *testing.T) {
	type args struct {
		clusterObj *chubaoapi.ChubaoCluster
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "test-1",
			args: args{
				clusterObj: &chubaoapi.ChubaoCluster{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "rook-chubao",
					},
					Spec: chubaoapi.ClusterSpec{
						Master: chubaoapi.MasterSpec{
							Replicas: 3,
							Port:     defaultPort,
						},
					},
				},
			},
			want: "master-0.master-service.rook-chubao.svc.cluster.local:17110,master-1.master-service.rook-chubao.svc.cluster.local:17110,master-2.master-service.rook-chubao.svc.cluster.local:17110",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetMasterAddr(tt.args.clusterObj); got != tt.want {
				t.Errorf("GetMasterAddrs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_convertToClusterInfo(t *testing.T) {
	type args struct {
		bytes []byte
	}
	tests := []struct {
		name  string
		args  args
		want  error
		want1 *ClusterInfo
	}{
		{
			name: "test-1",
			args: args{
				bytes: []byte("{\"code\":0,\"msg\":\"success\",\"data\":{\"Name\":\"my-cluster\",\"LeaderAddr\":\"master-1.master-service.chubaofs.svc.cluster.local:17010\"}}"),
			},
			want: nil,
			want1: &ClusterInfo{
				Name:       "my-cluster",
				LeaderAddr: "master-1.master-service.chubaofs.svc.cluster.local:17010",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := convertToClusterInfo(tt.args.bytes)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("convertToClusterInfo() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("convertToClusterInfo() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

//func Test_queryClusterInfo(t *testing.T) {
//	err, bytes := queryClusterInfo("test.chubaofs.jd.local")
//	if err != nil {
//		fmt.Println(err)
//		return
//	}
//
//	fmt.Println(string(bytes))
//}

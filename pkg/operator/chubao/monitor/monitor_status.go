package monitor

import (
	"context"
	"fmt"
	chubaoapi "github.com/rook/rook/pkg/apis/chubao.rook.io/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

var monitorInstance *chubaoapi.ChubaoMonitor

func (e *MonitorEventHandler) startMonitoring(stopCh chan struct{}, mon *chubaoapi.ChubaoMonitor) {
	monitorInstance = mon
	checkStatus()
	go wait.Until(checkStatus, time.Second*10, stopCh)
}

func checkStatus() {
	clt, err := client.New(ctrl.GetConfigOrDie(), client.Options{})
	if err != nil {
		fmt.Println("failed to create client")
	}
	checkConfigmapStatus(clt)
	checkPromStatus()
	checkGrafanaStatus()
	clt.Update(context.Background(), monitorInstance)
}

func checkConfigmapStatus(clt client.Client) {
	cfg := &corev1.ConfigMap{}
	clt.Get(context.Background(), types.NamespacedName{Name: "monitor-config", Namespace: monitor.Namespace}, cfg)
	monitorInstance.Status.Configmap = chubaoapi.ConfigmapStatusFailure

	content := []string{"prometheus.yml", "datasource.yml", "grafana.ini", "init.sh", "chubaofs.json", "dashboard.yml"}
	for _, value := range content {
		_, check := cfg.Data[value]
		if !check {
			monitorInstance.Status.Configmap = chubaoapi.ConfigmapStatusFailure
			return
		}
	}
	monitorInstance.Status.Configmap = chubaoapi.ConfigmapStatusReady
}

func checkPromStatus() {
	resp, err := http.Get("http://prometheus-service.rook-chubao.svc.cluster.local:9090")
	if err != nil {
		monitorInstance.Status.Prometheus = chubaoapi.PrometheusStatusFailure
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		monitorInstance.Status.Prometheus = chubaoapi.PrometheusStatusFailure
	} else {
		monitorInstance.Status.Prometheus = chubaoapi.PrometheusStatusReady
	}

}

func checkGrafanaStatus() {
	resp, err := http.Get("http://grafana-service.rook-chubao.svc.cluster.local:3000")
	if err != nil {
		monitorInstance.Status.Grafana = chubaoapi.GrafanaStatusFailure
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		monitorInstance.Status.Grafana = chubaoapi.GrafanaStatusFailure
	} else {
		monitorInstance.Status.Grafana = chubaoapi.GrafanaStatusReady
	}
}

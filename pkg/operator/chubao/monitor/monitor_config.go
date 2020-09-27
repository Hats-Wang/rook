package monitor

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/go-yaml/yaml"
	chubaoapi "github.com/rook/rook/pkg/apis/chubao.rook.io/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type datasource struct {
	ApiVersion        int                `yaml:"apiVersion,omitempty"`
	DeleteDatasources []deleteDatasource `yaml:"deleteDatasources,omitempty"`
	Datasources       []dataSource       `yaml:"datasources,omitempty"`
}

type deleteDatasource struct {
	Name  string `yaml:"name,omitempty"`
	OrgId int    `yaml:"orgId,omitempty"`
}
type dataSource struct {
	Name              string          `yaml:"name,omitempty"`
	Type              string          `yaml:"type,omitempty"`
	Access            string          `yaml:"access,omitempty"`
	OrgId             int             `yaml:"orgId,omitempty"`
	Url               string          `yaml:"url,omitempty"`
	password          string          `yaml:"url,omitempty"`
	User              string          `yaml:"user,omitempty"`
	Database          bool            `yaml:"database,omitempty"`
	BasicAuth         bool            `yaml:"basicAuth,omitempty"`
	BasicAuthUser     string          `yaml:"basicAuthUser,omitempty"`
	BasicAuthPassword string          `yaml:"basicAuthPassword,omitempty"`
	WithCredentials   bool            `yaml:"withCredentials,omitempty"`
	IsDefault         bool            `yaml:"isDefault,omitempty"`
	JsonData          jsonData        `yaml:"jsonData,omitempty"`
	SecutreJsonData   secutreJsonData `yaml:"secutreJsonData,omitempty"`
	Version           int             `yaml:"version,omitempty"`
	Editable          bool            `yaml:"editable,omitempty"`
}

type jsonData struct {
	GraphiteVersion   string `yaml:"graphiteVersion,omitempty"`
	TlsAuth           bool   `yaml:"tlsAuth,omitempty"`
	TlsAuthWithCACert bool   `yaml:"tlsAuthWithCACert,omitempty"`
}

type secutreJsonData struct {
	TlsCACert     string `yaml:"tlsCACert,omitempty"`
	TlsClientCert string `yaml:"tlsClientCert,omitempty"`
	TlsClientKey  string `yaml:"tlsClientKey,omitempty"`
}

type prometheusyml struct {
	Global        global          `yaml:"global,omitempty"`
	ScrapeConfigs []scrapeConfigs `yaml:"scrape_configs,omitempty"`
}

type global struct {
	ScrapeInterval     string `yaml:"scrape_interval,omitempty"`
	EvaluationInterval string `yaml:"evaluation_interval,omitempty"`
}

type scrapeConfigs struct {
	JobNmaes        string            `yaml:"job_name,omitempty"`
	RelabelConfigs  []relabelConfigs  `yaml:"relabel_configs,omitempty"`
	MetricsPath     string            `yaml:"metrics_path,omitempty"`
	Scheme          string            `yaml:"scheme,omitempty"`
	ConsulSdConfigs []consulSdConfigs `yaml:"consul_sd_configs,omitempty"`
}

type relabelConfigs struct {
	SourceLabels []string `yaml:"source_labels,omitempty,flow"`
	Action       string   `yaml:"action,omitempty"`
	Regex        string   `yaml:"regex,omitempty"`
	Replacement  string   `yaml:"replacement,omitempty"`
	TargetLabel  string   `yaml:"target_label,omitempty"`
}

type consulSdConfigs struct {
	Server   string   `yaml:"server,omitempty"`
	Services []string `yaml:"services,omitempty"`
}

var monitor *chubaoapi.ChubaoMonitor

func CreateNewConfigmap(mon *chubaoapi.ChubaoMonitor) error {
	cfg := &corev1.ConfigMap{}
	monitor = mon
	cfg.Namespace = monitor.Namespace
	cfg.Name = "monitor-config"
	cfg.Data = make(map[string]string)

	clt, err := client.New(ctrl.GetConfigOrDie(), client.Options{})
	if err != nil {
		fmt.Println("failed to create client")
		return err
	}

	err = addPrometheusYml(cfg)
	if err != nil {
		return err
	}
	err = addDatasourceYml(cfg)
	if err != nil {
		return err
	}
	err = addNewFile("/etc/monitor/grafana/grafana.ini", "grafana.ini", cfg)
	if err != nil {
		return err
	}

	err = addNewFile("/etc/monitor/grafana/init.sh", "init.sh", cfg)
	if err != nil {
		return err
	}

	err = addNewFile("/etc/monitor/grafana/dashboards/chubaofs.json", "chubaofs.json", cfg)
	if err != nil {
		return err
	}

	err = addNewFile("/etc/monitor/grafana/dashboards/dashboard.yml", "dashboard.yml", cfg)
	if err != nil {
		return err
	}

	err = clt.Create(context.Background(), cfg)
	if err != nil {
		monitor.Status.Configmap = chubaoapi.ConfigmapStatusFailure
		fmt.Println(err)
		return err
	}
	monitor.Status.Configmap = chubaoapi.ConfigmapStatusReady

	return nil
}

func addPrometheusYml(cfg *corev1.ConfigMap) error {
	promCfg, err := ioutil.ReadFile("/etc/monitor/prometheus/prometheus.yml")
	if err != nil {
		return err
	}
	promStruct := prometheusyml{}
	err = yaml.Unmarshal(promCfg, &promStruct)
	if err != nil {
		return err
	}
	promStruct.ScrapeConfigs[0].ConsulSdConfigs[0].Server = monitor.Spec.Prometheus.ConsulUrl

	newPromCfg, err := yaml.Marshal(&promStruct)
	if err != nil {
		return err
	}
	cfg.Data["prometheus.yml"] = string(newPromCfg)
	return nil

}

func addDatasourceYml(cfg *corev1.ConfigMap) error {
	grafDatasourceCfg, err := ioutil.ReadFile("/etc/monitor/grafana/datasources/datasource.yml")
	if err != nil {
		return err
	}
	datasourceCfg := datasource{}
	err = yaml.Unmarshal(grafDatasourceCfg, &datasourceCfg)
	if err != nil {
		return err
	}
	datasourceCfg.Datasources[0].Url = monitor.Spec.Grafana.PrometheusUrl

	newDatasourceCfg, err := yaml.Marshal(&datasourceCfg)
	if err != nil {
		return err
	}
	cfg.Data["datasource.yml"] = string(newDatasourceCfg)
	return nil
}

//The first parameter is the location of file in operator container. The second parameter is the index of the file in configmap and the name of the file.
func addNewFile(filePath, nameInCfg string, cfg *corev1.ConfigMap) error {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}
	cfg.Data[nameInCfg] = string(data)
	return nil
}

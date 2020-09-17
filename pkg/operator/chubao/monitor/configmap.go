package monitor

import (
	"errors"
	"fmt"
	"github.com/go-yaml/yaml"
	"github.com/rook/rook/pkg/operator/chubao/monitor/prometheus"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"net/http"
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

func AddDataToConfigmap(cfg *corev1.ConfigMap) error {

	err := addDashboardsymlFromGithub(cfg)
	if err != nil {
		return err
	}
	err = addChubaoFSJsonFromGithub(cfg)
	if err != nil {
		return err
	}
	err = addDatasource(cfg)
	if err != nil {
		return err
	}

	return nil
}

func addDashboardsymlFromGithub(cfg *corev1.ConfigMap) error {

	dbUrl := "https://raw.githubusercontent.com/chubaofs/chubaofs/master/docker/monitor/grafana/provisioning/dashboards/dashboard.yml"
	resp, err := http.Get(dbUrl)
	if err != nil {
		return errors.New(fmt.Sprintf("cannot request %s", dbUrl))
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("request failed with status %v", resp.StatusCode)
	}

	dashboard, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to get the response content")
	}
	dashboardyml := string(dashboard)
	cfg.Data["dashboard.yml"] = dashboardyml
	return nil
}

func addChubaoFSJsonFromGithub(cfg *corev1.ConfigMap) error {

	dbUrl := "https://raw.githubusercontent.com/chubaofs/chubaofs/master/docker/monitor/grafana/provisioning/dashboards/chubaofs.json"
	resp, err := http.Get(dbUrl)
	if err != nil {
		return errors.New(fmt.Sprintf("cannot request %s", dbUrl))
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("request failed with status %v", resp.StatusCode)
	}

	chubao, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to get the response content")
	}
	chubaoJson := string(chubao)
	cfg.Data["chubaofs.json"] = chubaoJson
	return nil
}

func addDatasource(cfg *corev1.ConfigMap) error {
	t := datasource{
		ApiVersion: 1,
		DeleteDatasources: []deleteDatasource{
			{
				Name:  "Prometheus",
				OrgId: 1,
			},
		},
		Datasources: []dataSource{
			{
				Name:              "Promethues",
				Type:              "prometheus",
				Access:            "proxy",
				OrgId:             1,
				Url:               prometheus.PrometheusServiceUrl,
				password:          "",
				BasicAuth:         false,
				BasicAuthUser:     "admin",
				BasicAuthPassword: "123456",
				IsDefault:         true,
				JsonData: jsonData{
					GraphiteVersion:   "1.1",
					TlsAuth:           false,
					TlsAuthWithCACert: false,
				},
				SecutreJsonData: secutreJsonData{
					TlsCACert:     `"..."`,
					TlsClientCert: `"..."`,
					TlsClientKey:  `"..."`,
				},
				Version:  1,
				Editable: true,
			},
		},
	}
	datasourceyml, err := yaml.Marshal(&t)
	if err != nil {
		return fmt.Errorf("failed to add datasource.yml")
	}

	cfg.Data["datasource.yml"] = string(datasourceyml)
	return nil
}

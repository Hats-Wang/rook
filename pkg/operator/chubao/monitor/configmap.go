package monitor

import (
	"errors"
	"fmt"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"net/http"
)

func ProcessConfigmap(cfg *corev1.ConfigMap) error {

	err := addDashboardsyml(cfg)
	if err != nil {
		return err
	}
	err = addChubaoFSJson(cfg)
	if err != nil {
		return err
	}

	return nil
}

func addDashboardsyml(cfg *corev1.ConfigMap) error {

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
	cfg.Data["dashboard.yml"] = string(dashboardyml)
	return nil
}

func addChubaoFSJson(cfg *corev1.ConfigMap) error {

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

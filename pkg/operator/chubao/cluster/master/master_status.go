package master

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
)

type responseData struct {
	Code int          `json:"code"`
	Msg  string       `json:"msg,omitempty"`
	Data MasterStatus `json:"data,omitempty"`
}

type MasterStatus struct {
	DataNodeStatInfo dataNodeStatInfo     `json:"dataNodeStatInfo,omitempty"`
	MetaNodeStatInfo metaNodeStatInfo     `json:"metaNodeStatInfo,omitempty"`
	ZoneStatInfo     map[string]*zoneStat `json:"ZoneStatInfo,omitempty"`
}

type dataNodeStatInfo struct {
	TotalGB     float32 `json:"TotalGB,omitempty"`
	UsedGB      float32 `json:"UsedGB,omitempty"`
	IncreasedGB float32 `json:"IncreasedGB,omitempty"`
	UsedRatio   float32 `json:"UsedRatio,omitempty"`
}

type metaNodeStatInfo struct {
	TotalGB     float32 `json:"TotalGB,omitempty"`
	UsedGB      float32 `json:"UsedGB,omitempty"`
	IncreasedGB float32 `json:"IncreasedGB,omitempty"`
	UsedRatio   float32 `json:"UsedRatio,omitempty"`
}

type zoneStat struct {
	DataNodeStat dataNodeStat `json:"dataNodeStat,omitempty"`
	MetaNodeStat metaNodeStat `json:"metaNodeStat,omitempty"`
}

type dataNodeStat struct {
	TotalGB       float32 `json:"TotalGB,omitempty"`
	UsedGB        float32 `json:"UsedGB,omitempty"`
	AvailGB       float32 `json:"AvailGB,omitempty"`
	UsedRatio     float32 `json:"UsedRatio,omitempty"`
	TotalNodes    int     `json:"TotalNodes,omitempty"`
	WritableNodes int     `json:"WritableNodes,omitempty"`
}

type metaNodeStat struct {
	TotalGB       float32 `json:"TotalGB,omitempty"`
	UsedGB        float32 `json:"UsedGB,omitempty"`
	AvailGB       float32 `json:"AvailGB,omitempty"`
	UsedRatio     float32 `json:"UsedRatio,omitempty"`
	TotalNodes    int     `json:"TotalNodes,omitempty"`
	WritableNodes int     `json:"WritableNodes,omitempty"`
}

func getMasterStatusURL(serviceName, namespace string, port int32) string {
	return fmt.Sprintf("http://%s.%s.svc.cluster.local/cluster/stat:%d", serviceName, namespace, port)
}

func (m *Master) GetStatus() (*MasterStatus, error) {
	masterStatusURL := getMasterStatusURL(defaultMasterServiceName, m.namespace, m.port)
	return getStatus(masterStatusURL)
}

func getStatus(masterStatusURL string) (*MasterStatus, error) {
	req, err := http.NewRequest(http.MethodGet, masterStatusURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to build the request for query chubao cluster status. url:%s", masterStatusURL))
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to query chubao cluster status. url:%s", masterStatusURL))
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Wrap(err, fmt.Sprintf("get failed StatusCode[%d] from chubao cluster. url:%s", resp.StatusCode, masterStatusURL))
	}

	responseData := &responseData{}
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to read chubao cluster status i/o. url:%s", masterStatusURL))
	}

	err = json.Unmarshal(bytes, responseData)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to unmarshal masterStatus. url:%s content:%s", masterStatusURL, string(bytes)))
	}

	if responseData.Code != 0 {
		return nil, fmt.Errorf("chubao cluster inner error, code:%d msg:%s", responseData.Code, responseData.Msg)
	}

	return &responseData.Data, nil
}

func (ms *MasterStatus) IsAvailable() bool {
	for _, value := range ms.ZoneStatInfo {
		if value.DataNodeStat.WritableNodes >= 3 && value.MetaNodeStat.WritableNodes >= 3 {
			return true
		}
	}

	return false
}

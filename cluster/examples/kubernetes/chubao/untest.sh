#/bin/bash

kubectl delete -f common.yaml

kubectl  delete  -f operator.yaml

kubectl  delete  -f cluster.yaml

kubectl  delete  -f ~/go/src/github.com/rook/rook/pkg/operator/chubao/monitor/chubaofsmonitor_configmap.yaml

kubectl  delete  -f monitor.yaml

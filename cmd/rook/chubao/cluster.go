/*
Copyright 2016 The Rook Authors. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package chubao

import (
	"github.com/pkg/errors"
	"github.com/rook/rook/cmd/rook/rook"
	"github.com/rook/rook/pkg/clusterd"
	"github.com/rook/rook/pkg/operator/chubao"
	"github.com/rook/rook/pkg/operator/k8sutil"
	"github.com/rook/rook/pkg/util/flags"
	"github.com/spf13/cobra"
	"os"
)

var clusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Deploys and runs the chubao cluster. The cluster contains Master, MetaNode, DataNode, and Consul components",
}

func init() {
	flags.SetFlagsFromEnv(clusterCmd.Flags(), rook.RookEnvVarPrefix)
	flags.SetLoggingFlags(clusterCmd.Flags())
	clusterCmd.RunE = startCluster
}

func startCluster(cmd *cobra.Command, args []string) error {
	rook.SetLogLevel()
	rook.LogStartupInfo(clusterCmd.Flags())

	operatorNamespace := os.Getenv(k8sutil.PodNamespaceEnvVar)
	if operatorNamespace == "" {
		rook.TerminateFatal(errors.Errorf("rook operator namespace is not provided. expose it via downward API in the rook operator manifest file using environment variable %q", k8sutil.PodNamespaceEnvVar))
	}

	logger.Info("starting Rook Chubao operator")
	context := createContext()
	context.NetworkInfo = clusterd.NetworkInfo{}
	context.ConfigDir = k8sutil.DataDir
	operator := chubao.New(context, operatorNamespace)
	err := operator.Run()
	if err != nil {
		rook.TerminateFatal(errors.Wrap(err, "failed to run operator\n"))
	}

	return nil
}

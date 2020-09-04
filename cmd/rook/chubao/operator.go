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
	"github.com/rook/rook/pkg/operator/chubao/controller"
	"github.com/rook/rook/pkg/operator/k8sutil"
	"github.com/rook/rook/pkg/util/flags"
	"github.com/spf13/cobra"
	"k8s.io/apiserver/pkg/server"
)

var operatorCmd = &cobra.Command{
	Use:   "operator",
	Short: "Runs the Chubao operator for orchestrating and managing Chubao storage in a Kubernetes cluster",
	Long: `Runs the Chubao operator for orchestrating and managing Chubao storage in a Kubernetes cluster
https://github.com/rook/rook`,
}

func init() {
	flags.SetFlagsFromEnv(operatorCmd.Flags(), rook.RookEnvVarPrefix)
	flags.SetLoggingFlags(operatorCmd.Flags())
	operatorCmd.RunE = startCluster
}

func startCluster(cmd *cobra.Command, args []string) error {
	rook.SetLogLevel()
	rook.LogStartupInfo(operatorCmd.Flags())

	//operatorNamespace := os.Getenv(k8sutil.PodNamespaceEnvVar)
	//if operatorNamespace == "" {
	//	rook.TerminateFatal(errors.Errorf("rook operator namespace is not provided. expose it via downward API in the rook operator manifest file using environment variable %q", k8sutil.PodNamespaceEnvVar))
	//}

	// Create a channel to receive OS signals
	stopCh := server.SetupSignalHandler()

	logger.Info("starting Rook Chubao operator")
	context := rook.NewContext()
	context.ConfigDir = k8sutil.DataDir
	context.LogLevel = rook.Cfg.LogLevel
	//operator := chubao.New(context, operatorNamespace)
	//err := operator.Run(stopCh)
	//if err != nil {
	//	rook.TerminateFatal(errors.Wrap(err, "failed to run operator\n"))
	//}

	c := controller.New(context, "rook-chubao-controller")
	err := c.Run(1, stopCh)
	if err != nil {
		rook.TerminateFatal(errors.Wrap(err, "failed to run operator\n"))
	}

	// Signal handler to stop the operator
	<-stopCh
	logger.Info("shutdown signal received, exiting...")
	return nil
}

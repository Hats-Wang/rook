/*
Copyright 2018 The Rook Authors. All rights reserved.

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
	"github.com/coreos/pkg/capnslog"
	"github.com/spf13/cobra"

	"github.com/rook/rook/cmd/rook/rook"
	"github.com/rook/rook/pkg/clusterd"
)

// Cmd is the main command for operator and daemons.
var Cmd = &cobra.Command{
	Use:   "chubao",
	Short: "Main command for Chubao operator and daemons.",
}

var (
	cfg    = &config{}
	logger = capnslog.NewPackageLogger("github.com/rook/rook", "chubaocmd")
)

type config struct {
	devices            string
	metadataDevice     string
	dataDir            string
	forceFormat        bool
	location           string
	cephConfigOverride string
	networkInfo        clusterd.NetworkInfo
	monEndpoints       string
	nodeName           string
	pvcBacked          bool
}

func init() {
	Cmd.AddCommand(
		clusterCmd,
		objectStoreCmd,
		monitorCmd,
		consoleCmd)
}

func createContext() *clusterd.Context {
	context := rook.NewContext()
	context.ConfigDir = cfg.dataDir
	context.ConfigFileOverride = cfg.cephConfigOverride
	context.NetworkInfo = cfg.NetworkInfo()
	return context
}

func (c *config) NetworkInfo() clusterd.NetworkInfo {
	return c.networkInfo.Simplify()
}

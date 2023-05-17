/*
Copyright 2023 KubeCube Authors

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

package e2e

import (
	"flag"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kubecube-io/kubecube/pkg/clog"

	// test sources
	_ "github.com/kubecube-io/kubecube-e2e/e2e/cloudshell_new"
	_ "github.com/kubecube-io/kubecube-e2e/e2e/cluster"
	_ "github.com/kubecube-io/kubecube-e2e/e2e/config/configmap_new"
	_ "github.com/kubecube-io/kubecube-e2e/e2e/config/secret_new"
	_ "github.com/kubecube-io/kubecube-e2e/e2e/crd_new"
	_ "github.com/kubecube-io/kubecube-e2e/e2e/ingress_new"
	_ "github.com/kubecube-io/kubecube-e2e/e2e/node"
	_ "github.com/kubecube-io/kubecube-e2e/e2e/service_new"
	_ "github.com/kubecube-io/kubecube-e2e/e2e/storageclass_new"
	_ "github.com/kubecube-io/kubecube-e2e/e2e/tenantquota_new"
	_ "github.com/kubecube-io/kubecube-e2e/e2e/workloads_new"

	"github.com/kubecube-io/kubecube-e2e/e2e/framework"
)

var output = flag.String("MCOutput", "", "multi config output config")
var runUsingDefault = flag.Bool("runDefault", false, "run using default output config")
var master = flag.Bool("master", false, "whether to init and clear resource")
var runningUser = flag.String("runAs", "admin", "run using default output config")

// entrance
func TestMain(m *testing.M) {
	flag.Parse()
	if len(*output) > 0 {
		err := framework.OutputMultiUserTestConfig(*output)
		if err != nil {
			os.Exit(1)
		}
		return
	}

	if *runUsingDefault {
		err := framework.OutputMultiUserTestConfig(framework.MultiConfig)
		if err != nil {
			os.Exit(1)
		}
	}

	split := strings.Split(*runningUser, ",")
	for _, s := range split {
		if len(s) > 0 {
			framework.TestUser = append(framework.TestUser, s)
		}
	}
	clog.Info("running user %+v", framework.TestUser)

	isMaster = *master

	if err := InitAll(); err != nil {
		clog.Error(err.Error())
		os.Exit(1)
	}

	if err := Start(); err != nil {
		clog.Error(err.Error())
		err := End()
		if err != nil {
			clog.Error(err.Error())
		}
		os.Exit(1)
	}
	rand.Seed(time.Now().UnixNano())
	m.Run()
	err := End()
	if err != nil {
		clog.Error(err.Error())
		os.Exit(1)
	}
}

// start e2e test
func TestE2E(t *testing.T) {
	RunE2ETests(t)
}

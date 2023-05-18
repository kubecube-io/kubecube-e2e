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

package secret

import (
	"github.com/kubecube-io/kubecube-e2e/e2e/framework"
	"github.com/kubecube-io/kubecube/pkg/multicluster/client"
)

var (
	clusterName string
	httpHelper  *framework.HttpHelper
	namespace   string
	cli         client.Client

	secretName             = "e2e-test-docker-secret"
	podName                = "e2e-pod-docker-secret-test"
	opaqueSecretName       = "e2e-test-opaque-secret"
	opaqueSecretPodName    = "e2e-pod-secret-test"
	opaqueSecretEnvPodName = "e2e-pod-secret-env-test"
)

func initParam() {
	clusterName = framework.TargetClusterName
	httpHelper = framework.NewSingleHttpHelper()
	namespace = framework.NamespaceName
	cli = framework.TargetClusterClient
}

func init() {
	framework.RegisterByDefault(multiUserOpaqueSecretTest)
	framework.RegisterByDefault(multiUserDockerConfigJsonSecretTest)
}

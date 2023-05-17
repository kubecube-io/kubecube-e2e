package secret_new

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

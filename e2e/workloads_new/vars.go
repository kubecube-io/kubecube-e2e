package workloads_new

import (
	"github.com/kubecube-io/kubecube/pkg/multicluster/client"

	"github.com/kubecube-io/kubecube-e2e/e2e/framework"
)

var (
	httpHelper   *framework.HttpHelper
	targetClient client.Client

	cronJobName         = "hellocronjob"
	cronJobNameWithUser string

	daemonSetName         = "hellods"
	daemonSetNameWithUser string

	deployName = "e2e-test-deploy"
	pv1Name    = "pv1"
	pv2Name    = "pv2"

	deployNameWithUser string
	pv1NameWithUser    string
	pv2NameWithUser    string

	jobName         = "hellojob"
	jobNameWithUser string

	stsName         = "sts1"
	stsNameWithUser string
)

func initParam() {
	httpHelper = framework.NewSingleHttpHelper()
	targetClient = framework.TargetClusterClient
}

func init() {
	framework.RegisterByDefault(multiUserCronjobTest)
	framework.RegisterByDefault(multiUserJobTest)
	framework.RegisterByDefault(multiUserDeployTest)
	framework.RegisterByDefault(multiUserStsTest)
	framework.RegisterByDefault(multiUserDsTest)
}

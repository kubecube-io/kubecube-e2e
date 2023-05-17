package service_new

import (
	"context"
	"time"

	v12 "k8s.io/api/apps/v1"
	v13 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubecube-io/kubecube-e2e/e2e/framework"
)

var (
	ctx          = context.Background()
	deploy1      *v12.Deployment
	deploy2      *v12.Deployment
	svc1         *v13.Service
	ingress1     *v1beta1.Ingress
	ns1          *v13.Namespace
	httpHelper   *framework.HttpHelper
	waitInterval time.Duration
	waitTimeout  time.Duration
	cli          client.Client

	nsExist = true

	deploy1Name         = "deploy1"
	deploy2Name         = "deploy2"
	deploy1NameWithUser string
	deploy2NameWithUser string

	service1Name         = "service1"
	service1NameWithUser string
	service2Name         = "service2"
	service2NameWithUser string

	host         = "test.e2e.%s.svc"
	hostWithName string

	ingress1Name         = "ingress"
	ingress1NameWithUser string

	portMap = map[string]int{
		framework.UserAdmin:        1111,
		framework.UserProjectAdmin: 1121,
		framework.UserTenantAdmin:  1131,
		framework.UserNormal:       1141,
	}
	newportMap = map[string]int{
		framework.UserAdmin:        1114,
		framework.UserProjectAdmin: 1124,
		framework.UserTenantAdmin:  1134,
		framework.UserNormal:       1144,
	}
)

func initParam() {
	httpHelper = framework.NewSingleHttpHelper()
	waitInterval = framework.WaitInterval
	waitTimeout = framework.WaitTimeout
	cli = framework.PivotClusterClient.Direct()
}

func init() {
	framework.RegisterByDefault(multiUserServiceCRUDTest)
	framework.RegisterByDefault(multiUserServiceEventTest)
	framework.RegisterByDefault(multiUserServiceNodeportTest)
}

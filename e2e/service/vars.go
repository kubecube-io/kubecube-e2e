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

package service

import (
	"context"
	"time"

	v12 "k8s.io/api/apps/v1"
	v13 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubecube-io/kubecube-e2e/e2e/framework"
)

var (
	ctx          = context.Background()
	deploy1      *v12.Deployment
	deploy2      *v12.Deployment
	svc1         *v13.Service
	ingress1     *networkingv1.Ingress
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

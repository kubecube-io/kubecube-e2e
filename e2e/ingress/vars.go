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

package ingress

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
	httpHelper *framework.HttpHelper

	deploy1  *v12.Deployment
	svc1     *v13.Service
	ingress1 *v1beta1.Ingress
	ingress2 *v1beta1.Ingress

	ctx = context.Background()

	deploy1Name          = "deploy1"
	svc1Name             = "svc1"
	ingress1Name         = "ingress1"
	ingress2Name         = "ingress"
	ingressAddr          = "e2e.test"
	deploy1NameWithUser  string
	svc1NameWithUser     string
	ingress1NameWithUser string
	ingress2NameWithUser string

	waitInterval time.Duration
	waitTimeout  time.Duration
	cli          client.Client
)

func initParam() {
	httpHelper = framework.NewSingleHttpHelper()
	waitInterval = framework.WaitInterval
	waitTimeout = framework.WaitTimeout
	cli = framework.PivotClusterClient.Direct()
}

func init() {
	framework.RegisterByDefault(multiUserIngressCRUDTest)
	framework.RegisterByDefault(multiUserIngressEventTest)
	framework.RegisterByDefault(multiUserIngressFunctionTest)
}

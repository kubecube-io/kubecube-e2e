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

package tenantquota

import (
	"context"

	quotav1 "github.com/kubecube-io/kubecube/pkg/apis/quota/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kubecube-io/kubecube-e2e/e2e/framework"
)

func updateTenantQuota(user string) framework.TestResp {
	ctx := context.TODO()
	memberClient := framework.TargetClusterClient
	cubeQuota := &quotav1.CubeResourceQuota{}
	err := memberClient.Direct().Get(ctx, types.NamespacedName{Name: framework.CubeResourceQuota}, cubeQuota)
	framework.ExpectNoError(err, "get cube resource quota should success")

	cubeQuota.Spec.Hard[corev1.ResourceRequestsCPU] = resource.MustParse("11")
	cubeQuota.Spec.Hard[corev1.ResourceRequestsMemory] = resource.MustParse("11Gi")
	cubeQuota.Spec.Hard[corev1.ResourceLimitsCPU] = resource.MustParse("11")
	cubeQuota.Spec.Hard[corev1.ResourceRequestsMemory] = resource.MustParse("11Gi")
	cubeQuota.Spec.Hard[corev1.ResourceRequestsStorage] = resource.MustParse("31Gi")

	err = memberClient.Direct().Update(ctx, cubeQuota)
	framework.ExpectNoError(err, "update tenant quota should success")
	return framework.SucceedResp
}

var multiUserTest = framework.MultiUserTest{
	TestName:        "[配额和空间管理][9382713]租户配额管理",
	ContinueIfError: false,
	Skipfunc:        nil,
	SkipUsers:       []string{framework.UserProjectAdmin, framework.UserTenantAdmin, framework.UserNormal},
	ErrorFunc:       framework.PermissionErrorFunc,
	AfterEach:       nil,
	BeforeEach:      nil,
	InitStep:        nil,
	FinalStep:       nil,
	Steps: []framework.MultiUserTestStep{
		{
			Name:        "正常调整配额应该成功",
			Description: "1.查看租户配额\n2.调节cpu、mem、storage数值",
			StepFunc:    updateTenantQuota,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
	},
}

func init() {
	framework.RegisterByDefault(multiUserTest)
}

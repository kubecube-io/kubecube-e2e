package tenantquota_new

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

	cubeQuota.Spec.Hard[corev1.ResourceRequestsCPU] = resource.MustParse("3")
	cubeQuota.Spec.Hard[corev1.ResourceRequestsMemory] = resource.MustParse("3Gi")
	cubeQuota.Spec.Hard[corev1.ResourceLimitsCPU] = resource.MustParse("3")
	cubeQuota.Spec.Hard[corev1.ResourceRequestsMemory] = resource.MustParse("3Gi")
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

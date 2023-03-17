package ingress_new

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/kubecube-io/kubecube/pkg/clog"
	v12 "k8s.io/api/apps/v1"
	v13 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/kubecube-io/kubecube-e2e/e2e/framework"
)

func createDeployAndSvcAndIngress(user string) framework.TestResp {
	initParam()
	deploy1NameWithUser = framework.NameWithUser(deploy1Name, user)
	svc1NameWithUser = framework.NameWithUser(svc1Name, user)
	ingress1NameWithUser = framework.NameWithUser(ingress1Name, user)
	replicas := int32(1)
	deploy1 = &v12.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name:      deploy1NameWithUser,
			Namespace: framework.NamespaceName,
		},
		Spec: v12.DeploymentSpec{
			Selector: &v1.LabelSelector{
				MatchLabels: map[string]string{"kubecube.io/app": deploy1NameWithUser},
			},
			Replicas: &replicas,
			Template: v13.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{"kubecube.io/app": deploy1NameWithUser},
				},
				Spec: v13.PodSpec{
					Containers: []v13.Container{
						{
							Name:  "nginx",
							Image: framework.TestImage,
						},
					},
					ImagePullSecrets: []v13.LocalObjectReference{{Name: framework.ImagePullSecret}},
				},
			},
		},
	}
	err := framework.TargetClusterClient.Direct().Create(ctx, deploy1)
	framework.ExpectNoError(err)

	svc1 = &v13.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:      svc1NameWithUser,
			Namespace: framework.NamespaceName,
		},
		Spec: v13.ServiceSpec{
			Selector: map[string]string{"kubecube.io/app": deploy1NameWithUser},
			Ports: []v13.ServicePort{
				{Name: "port1", Protocol: "TCP", Port: 80, TargetPort: intstr.FromInt(80)},
			},
		},
	}
	err = framework.TargetClusterClient.Direct().Create(ctx, svc1)
	framework.ExpectNoError(err)

	ingress1 = &v1beta1.Ingress{
		ObjectMeta: v1.ObjectMeta{
			Name:      ingress1NameWithUser,
			Namespace: framework.NamespaceName,
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				{Host: "test.e2e", IngressRuleValue: v1beta1.IngressRuleValue{
					HTTP: &v1beta1.HTTPIngressRuleValue{
						Paths: []v1beta1.HTTPIngressPath{
							{
								Backend: v1beta1.IngressBackend{
									ServiceName: svc1NameWithUser,
									ServicePort: intstr.FromInt(80),
								},
							},
						},
					},
				}},
			},
		},
	}
	err = framework.TargetClusterClient.Direct().Create(ctx, ingress1)
	framework.ExpectNoError(err)
	return framework.SucceedResp
}

func deleteDeployAndSvcAndIngress(user string) framework.TestResp {
	err := framework.TargetClusterClient.Direct().Delete(ctx, deploy1)
	framework.ExpectNoError(err)
	err = framework.TargetClusterClient.Direct().Delete(ctx, svc1)
	framework.ExpectNoError(err)
	err = framework.TargetClusterClient.Direct().Delete(ctx, ingress1)
	framework.ExpectNoError(err)
	return framework.SucceedResp
}

func checkEvents(user string) framework.TestResp {
	url := "/api/v1/cube/proxy/clusters/" + framework.TargetClusterName + "/api/v1/namespaces/" + framework.NamespaceName + "/events?fieldSelector=involvedObject.kind=Ingress,involvedObject.name=" + ingress1NameWithUser
	resp, err := httpHelper.RequestByUser(http.MethodGet, framework.KubecubeHost+url, "", user, nil)
	framework.ExpectNoError(err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	framework.ExpectNoError(err)
	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(errors.New("fail to list ingress events"), resp.StatusCode)
	}

	framework.ExpectEqual(resp.StatusCode, http.StatusOK)

	var eventList v13.EventList
	err = json.Unmarshal(body, &eventList)
	framework.ExpectNoError(err)
	return framework.SucceedResp
}

var multiUserIngressEventTest = framework.MultiUserTest{
	TestName:        "[ingress][9386673]负载均衡事件检查",
	ContinueIfError: false,
	SkipUsers:       []string{},
	Skipfunc:        nil,
	ErrorFunc:       framework.PermissionErrorFunc,
	AfterEach:       nil,
	BeforeEach:      nil,
	InitStep: &framework.MultiUserTestStep{
		Name:        "创建 deploy svc ingress",
		Description: "创建 deploy svc ingress",
		StepFunc:    createDeployAndSvcAndIngress,
	},
	FinalStep: &framework.MultiUserTestStep{
		Name:        "删除 deploy svc ingress",
		Description: "删除 deploy svc ingress",
		StepFunc:    deleteDeployAndSvcAndIngress,
	},
	Steps: []framework.MultiUserTestStep{
		{
			Name:        "可罗列所有的 service events",
			Description: "查看ingress事件",
			StepFunc:    checkEvents,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
	},
}

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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/kubecube-io/kubecube/pkg/clog"
	v12 "k8s.io/api/apps/v1"
	v13 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/kubecube-io/kubecube-e2e/e2e/framework"
)

func createDeployAndService(user string) framework.TestResp {
	initParam()
	deploy1NameWithUser = framework.NameWithUser(deploy1Name, user)
	service1NameWithUser = framework.NameWithUser(service1Name, user)
	hostWithName = fmt.Sprintf(host, user)
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
			Name:      service1NameWithUser,
			Namespace: framework.NamespaceName,
		},
		Spec: v13.ServiceSpec{
			Selector: map[string]string{"kubecube.io/app": deploy1NameWithUser},
			Ports: []v13.ServicePort{
				{Name: "port1", Protocol: "TCP", Port: 8080, TargetPort: intstr.FromInt(8080)},
			},
		},
	}
	err = framework.TargetClusterClient.Direct().Create(ctx, svc1)
	framework.ExpectNoError(err)

	pathType := networkingv1.PathTypeImplementationSpecific

	ingress1 = &networkingv1.Ingress{
		ObjectMeta: v1.ObjectMeta{
			Name:      ingress1NameWithUser,
			Namespace: framework.NamespaceName,
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{Host: hostWithName, IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{
							{
								PathType: &pathType,
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: service1NameWithUser,
										Port: networkingv1.ServiceBackendPort{
											Number: 8080,
										},
									},
								},
							},
						},
					},
				}},
			},
		},
	}
	err = framework.TargetConvertClient.Create(ctx, ingress1)
	framework.ExpectNoError(err)
	return framework.SucceedResp
}

func deleteDeployAndService(user string) framework.TestResp {
	err := framework.TargetClusterClient.Direct().Delete(ctx, deploy1)
	framework.ExpectNoError(err)
	err = framework.TargetClusterClient.Direct().Delete(ctx, svc1)
	framework.ExpectNoError(err)
	err = framework.TargetConvertClient.Delete(ctx, ingress1)
	framework.ExpectNoError(err)
	return framework.SucceedResp
}

func checkServiceEvent(user string) framework.TestResp {
	url := "/api/v1/cube/proxy/clusters/" + framework.TargetClusterName + "/api/v1/namespaces/" + framework.NamespaceName + "/events?fieldSelector=involvedObject.kind=Service,involvedObject.name=" + service1NameWithUser
	resp, err := httpHelper.RequestByUser(http.MethodGet, framework.KubecubeHost+url, "", user, nil)
	framework.ExpectNoError(err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	framework.ExpectNoError(err)
	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(errors.New("fail to get svc events"), resp.StatusCode)
	}

	framework.ExpectEqual(resp.StatusCode, http.StatusOK)

	var eventList v13.EventList
	err = json.Unmarshal(body, &eventList)
	framework.ExpectNoError(err)
	return framework.SucceedResp
}

var multiUserServiceEventTest = framework.MultiUserTest{
	TestName:        "[service][9386664]服务事件检查",
	ContinueIfError: false,
	Skipfunc:        nil,
	ErrorFunc:       framework.PermissionErrorFunc,
	AfterEach:       nil,
	BeforeEach:      nil,
	InitStep: &framework.MultiUserTestStep{
		Name:        "创建 deploy service",
		Description: "创建 deploy service",
		StepFunc:    createDeployAndService,
	},
	FinalStep: &framework.MultiUserTestStep{
		Name:        "删除 deploy service",
		Description: "删除 deploy service",
		StepFunc:    deleteDeployAndService,
	},
	Steps: []framework.MultiUserTestStep{
		{
			Name:        "可罗列所有的 service events",
			Description: "查看服务事件",
			StepFunc:    checkServiceEvent,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
	},
}

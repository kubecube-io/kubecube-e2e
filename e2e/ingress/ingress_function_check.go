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
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kubecube-io/kubecube/pkg/clog"
	v12 "k8s.io/api/apps/v1"
	v13 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/kubecube-io/kubecube-e2e/e2e/framework"
)

func createDeployAndSvc2(user string) framework.TestResp {
	initParam()
	replicas := int32(2)
	deploy1NameWithUser = framework.NameWithUser(deploy1Name, user)
	svc1NameWithUser = framework.NameWithUser(svc1Name, user)
	ingress2NameWithUser = framework.NameWithUser(ingress2Name, user)
	err := wait.Poll(waitInterval, waitTimeout,
		func() (bool, error) {
			var namespace v13.Namespace
			errInfo := cli.Get(ctx, types.NamespacedName{Name: framework.NamespaceName}, &namespace)
			if errInfo == nil {
				return true, nil
			} else {
				return false, nil
			}
		})
	framework.ExpectNoError(err)

	cpu := resource.MustParse("100m")
	memory := resource.MustParse("100Mi")
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
							Resources: v13.ResourceRequirements{
								Requests: v13.ResourceList{
									v13.ResourceCPU:    cpu,
									v13.ResourceMemory: memory,
								},
								Limits: v13.ResourceList{
									v13.ResourceCPU:    cpu,
									v13.ResourceMemory: memory,
								},
							},
						},
					},
					ImagePullSecrets: []v13.LocalObjectReference{{Name: framework.ImagePullSecret}},
				},
			},
		},
	}
	err = cli.Create(ctx, deploy1)
	framework.ExpectNoError(err)

	svc1 = &v13.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:      svc1NameWithUser,
			Namespace: framework.NamespaceName,
		},
		Spec: v13.ServiceSpec{
			Selector: map[string]string{"kubecube.io/app": deploy1NameWithUser},
			Ports: []v13.ServicePort{
				{
					Name:       "demo-port",
					Port:       80,
					TargetPort: intstr.IntOrString{IntVal: 80},
				},
			},
		},
	}
	err = cli.Create(ctx, svc1)
	framework.ExpectNoError(err)
	return framework.SucceedResp
}

func deleteDeployAndSvc2(user string) framework.TestResp {
	framework.ExpectNoError(cli.Delete(ctx, deploy1))
	framework.ExpectNoError(cli.Delete(ctx, svc1))

	return framework.SucceedResp
}

func createIngress2(user string) framework.TestResp {
	url := "/api/v1/cube/proxy/clusters/" + framework.PivotClusterName + "/apis/networking.k8s.io/v1/namespaces/" + framework.NamespaceName + "/ingresses"
	postJson := fmt.Sprintf("{\"apiVersion\":\"networking.k8s.io/v1\",\"kind\":\"Ingress\",\"metadata\":{\"name\":\"%s\",\"annotations\":{\"nginx.ingress.kubernetes.io/load-balance\":\"round_robin\"},\"labels\":{}},\"spec\":{\"rules\":[{\"host\":\"%s\",\"http\":{\"paths\":[{\"pathType\":\"ImplementationSpecific\",\"path\":\"/%s\",\"backend\":{\"service\":{\"name\":\"%s\",\"port\":{\"number\":80}}}}]}}],\"tls\":[]}}",
		ingress2NameWithUser, ingressAddr, user, svc1NameWithUser)
	resp, err := httpHelper.RequestByUser(http.MethodPost, framework.KubecubeHost+url, postJson, user, nil)
	framework.ExpectNoError(err)
	defer resp.Body.Close()

	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to create ingress %s", ingress2NameWithUser), resp.StatusCode)
	}

	framework.ExpectEqual(resp.StatusCode, http.StatusCreated)
	ingress2 = &v1beta1.Ingress{}
	err = wait.Poll(waitInterval, waitTimeout,
		func() (bool, error) {
			err = cli.Get(ctx, types.NamespacedName{Name: ingress2NameWithUser, Namespace: framework.NamespaceName}, ingress2)
			framework.ExpectNoError(err)
			if ingress2.Name == ingress2NameWithUser {
				return true, nil
			} else {
				return false, nil
			}
		})
	framework.ExpectNoError(err)
	time.Sleep(time.Second * 30)
	return framework.SucceedResp
}

func accessIngress(user string) framework.TestResp {
	url := fmt.Sprintf("http://%s/%s", ingressAddr, user)
	resp, err := httpHelper.Get(url, nil)
	framework.ExpectNoError(err)
	defer resp.Body.Close()
	framework.ExpectEqual(resp.StatusCode, http.StatusOK)
	body, err := io.ReadAll(resp.Body)
	framework.ExpectNoError(err)
	bodyList := strings.Split(string(body), "\n")
	address := strings.TrimPrefix(bodyList[0], "Server address: ")
	err = wait.Poll(waitInterval, waitTimeout,
		func() (bool, error) {
			resp, err = httpHelper.Get(url, nil)
			if err == nil && resp.StatusCode == http.StatusOK {
				defer resp.Body.Close()
				body, err = io.ReadAll(resp.Body)
				framework.ExpectNoError(err)
				bodyList = strings.Split(string(body), "\n")
				a := strings.TrimPrefix(bodyList[0], "Server address: ")
				if a != address {
					return true, nil
				} else {
					return false, nil
				}
			} else {
				return false, nil
			}
		})
	framework.ExpectNoError(err)
	return framework.SucceedResp
}

func updateIngress2(user string) framework.TestResp {
	url := "/api/v1/cube/proxy/clusters/" + framework.PivotClusterName + "/apis/networking.k8s.io/v1/namespaces/" + framework.NamespaceName + "/ingresses/" + ingress2NameWithUser
	err := wait.Poll(waitInterval, waitTimeout,
		func() (bool, error) {
			err := cli.Get(ctx, types.NamespacedName{Name: ingress2NameWithUser, Namespace: framework.NamespaceName}, ingress2)
			framework.ExpectNoError(err)
			if ingress2.Name == ingress2NameWithUser {
				return true, nil
			} else {
				return false, nil
			}
		})
	framework.ExpectNoError(err)
	postJson := fmt.Sprintf("{\"metadata\":{\"namespace\":\"%s\",\"pureLabels\":{},\"resourceVersion\":\"%s\",\"uid\":\"%s\",\"name\":\"%s\",\"annotations\":{\"nginx.ingress.kubernetes.io/load-balance\":\"round_robin\",\"nginx.ingress.kubernetes.io/affinity\":\"cookie\",\"nginx.ingress.kubernetes.io/session-cookie-hash\":\"md5\",\"nginx.ingress.kubernetes.io/session-cookie-name\":\"qz_t\"},\"labels\":{}},\"spec\":{\"rules\":[{\"host\":\"%s\",\"http\":{\"paths\":[{\"pathType\":\"ImplementationSpecific\",\"path\":\"/%s\",\"backend\":{\"service\":{\"name\":\"%s\",\"port\":{\"number\":80}}}}]}}],\"tls\":[]}}",
		framework.NamespaceName, ingress2.ResourceVersion, ingress2.UID, ingress2NameWithUser, ingressAddr, user, svc1NameWithUser)
	resp, err := httpHelper.RequestByUser(http.MethodPut, framework.KubecubeHost+url, postJson, user, nil)
	framework.ExpectNoError(err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	clog.Debug("get ingress cookie response: %s", string(body))
	framework.ExpectNoError(err)

	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to update ingress %s", ingress2NameWithUser), resp.StatusCode)
	}

	framework.ExpectEqual(resp.StatusCode, http.StatusOK)
	err = wait.Poll(waitInterval, waitTimeout,
		func() (bool, error) {
			err = cli.Get(ctx, types.NamespacedName{Name: ingress2NameWithUser, Namespace: framework.NamespaceName}, ingress2)
			framework.ExpectNoError(err)
			if ingress2.Annotations["nginx.ingress.kubernetes.io/affinity"] == "cookie" {
				return true, nil
			} else {
				return false, nil
			}
		})
	framework.ExpectNoError(err)
	return framework.SucceedResp
}

func accessIngress2(user string) framework.TestResp {
	url := fmt.Sprintf("http://%s/%s", ingressAddr, user)
	resp, err := httpHelper.Get(url, nil)
	framework.ExpectNoError(err)
	defer resp.Body.Close()
	framework.ExpectEqual(resp.StatusCode, http.StatusOK)
	body, err := io.ReadAll(resp.Body)
	framework.ExpectNoError(err)
	bodyList := strings.Split(string(body), "\n")
	address2 := strings.TrimPrefix(bodyList[0], "Server address: ")
	for i := 0; i < 5; i++ {
		resp, err = httpHelper.Get(url, nil)
		framework.ExpectNoError(err)
		framework.ExpectEqual(resp.StatusCode, http.StatusOK)
		body, err = io.ReadAll(resp.Body)
		framework.ExpectNoError(err)
		bodyList = strings.Split(string(body), "\n")
		a := strings.TrimPrefix(bodyList[0], "Server address: ")
		framework.ExpectEqual(a, address2)
	}
	defer resp.Body.Close()
	framework.ExpectNoError(err)
	return framework.SucceedResp
}

func deleteIngress2(user string) framework.TestResp {
	framework.ExpectNoError(cli.Delete(ctx, ingress2))
	return framework.SucceedResp
}

var multiUserIngressFunctionTest = framework.MultiUserTest{
	TestName:        "[ingress][9386668]负载均衡功能点检查",
	ContinueIfError: false,
	SkipUsers:       []string{},
	Skipfunc:        nil,
	ErrorFunc:       framework.PermissionErrorFunc,
	AfterEach:       nil,
	BeforeEach:      nil,
	InitStep: &framework.MultiUserTestStep{
		Name:        "创建 deploy 和 svc",
		Description: "创建 deploy 和 svc",
		StepFunc:    createDeployAndSvc2,
	},
	FinalStep: &framework.MultiUserTestStep{
		Name:        "删除 deploy 和 svc",
		Description: "删除 deploy 和 svc",
		StepFunc:    deleteDeployAndSvc2,
	},
	Steps: []framework.MultiUserTestStep{
		{
			Name:        "1. 创建ingress2，设置调度算法为round robin",
			Description: "1. 创建ingress2，设置调度算法为round robin",
			StepFunc:    createIngress2,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
		{
			Name:        "2. 客户端访问此ingress",
			Description: "2. 客户端访问此ingress",
			StepFunc:    accessIngress,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
		{
			Name:        "3. 设置ingress会话保持开启，设置任意Cookie名称",
			Description: "3. 设置ingress会话保持开启，设置任意Cookie名称",
			StepFunc:    updateIngress2,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
		{
			Name:        "4. 重复第2步",
			Description: "4. 重复第2步",
			StepFunc:    accessIngress2,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
		{
			Name:        "删除成功",
			Description: "删除ingress2",
			StepFunc:    deleteIngress2,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
	},
}

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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/kubecube-io/kubecube/pkg/clog"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/kubecube-io/kubecube-e2e/e2e/framework"
)

func createDeployAndSvc1(user string) framework.TestResp {
	initParam()
	deploy1NameWithUser = framework.NameWithUser(deploy1Name, user)
	svc1NameWithUser = framework.NameWithUser(svc1Name, user)
	ingress1NameWithUser = framework.NameWithUser(ingress1Name, user)
	replicas := int32(1)
	deploy1 = &appsv1.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name:      deploy1NameWithUser,
			Namespace: framework.NamespaceName,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &v1.LabelSelector{
				MatchLabels: map[string]string{"kubecube.io/app": deploy1NameWithUser},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{"kubecube.io/app": deploy1NameWithUser},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: framework.TestImage,
						},
					},
					ImagePullSecrets: []corev1.LocalObjectReference{{Name: framework.ImagePullSecret}},
				},
			},
		},
	}
	err := framework.TargetClusterClient.Direct().Create(ctx, deploy1)
	framework.ExpectNoError(err)

	svc1 = &corev1.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:      svc1NameWithUser,
			Namespace: framework.NamespaceName,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"kubecube.io/app": deploy1NameWithUser},
			Ports: []corev1.ServicePort{
				{
					Name:       "demo-port",
					Port:       8080,
					TargetPort: intstr.IntOrString{IntVal: 8080},
				},
			},
		},
	}
	err = framework.TargetClusterClient.Direct().Create(ctx, svc1)
	framework.ExpectNoError(err)
	return framework.SucceedResp
}

func deleteDeployAndSvc1(user string) framework.TestResp {
	err := framework.TargetClusterClient.Direct().Delete(ctx, deploy1)
	framework.ExpectNoError(err)

	err = framework.TargetClusterClient.Direct().Delete(ctx, svc1)
	framework.ExpectNoError(err)
	return framework.SucceedResp
}

func createIngress(user string) framework.TestResp {
	url := "/api/v1/cube/proxy/clusters/" + framework.TargetClusterName + "/apis/networking.k8s.io/v1/namespaces/" + framework.NamespaceName + "/ingresses"
	postJson := fmt.Sprintf(`{
	"apiVersion": "networking.k8s.io/v1",
	"kind": "Ingress",
	"metadata": {
		"name": "%s",
		"annotations": {
			"nginx.ingress.kubernetes.io/load-balance": "round_robin",
			"kubernetes.io/ingress.class": "istio"
		},
		"labels": {}
	},
	"spec": {
		"rules": [{
			"host": "poctest",
			"http": {
				"paths": [{
					"path": "/test",
					"pathType": "ImplementationSpecific",
					"backend": {
						"service": {
							"name": "%s",
							"port": {
								"number": 8080
							}
						}
					}
				}]
			}
		}],
		"tls": []
	}
}`, ingress1NameWithUser, svc1NameWithUser)
	resp, err := httpHelper.RequestByUser(http.MethodPost, framework.KubecubeHost+url, postJson, user, nil)
	framework.ExpectNoError(err)
	defer resp.Body.Close()

	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to create ingress %s", ingress1NameWithUser), resp.StatusCode)
	}

	// check return success
	framework.ExpectEqual(resp.StatusCode, http.StatusCreated)
	return framework.SucceedResp
}

func listIngress(user string) framework.TestResp {
	url := "/api/v1/cube/proxy/clusters/" + framework.TargetClusterName + "/apis/networking.k8s.io/v1/namespaces/" + framework.NamespaceName + "/ingresses?pageNum=1&pageSize=10"
	resp, err := httpHelper.RequestByUser(http.MethodGet, framework.KubecubeHost+url, "", user, nil)
	framework.ExpectNoError(err)
	defer resp.Body.Close()

	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(errors.New("fail to list ingress"), resp.StatusCode)
	}

	// check return success
	framework.ExpectEqual(resp.StatusCode, http.StatusOK)
	body, err := io.ReadAll(resp.Body)
	framework.ExpectNoError(err)
	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	framework.ExpectNoError(err)
	framework.ExpectNotEqual(int(result["total"].(float64)), 0)
	return framework.SucceedResp
}

func updateIngress(user string) framework.TestResp {
	ingress := &networkingv1.Ingress{}
	err := framework.TargetConvertClient.Get(ctx, types.NamespacedName{Name: ingress1NameWithUser, Namespace: framework.NamespaceName}, ingress)
	framework.ExpectNoError(err)

	url := "/api/v1/cube/proxy/clusters/" + framework.TargetClusterName + "/apis/networking.k8s.io/v1/namespaces/" + framework.NamespaceName + "/ingresses/" + ingress1NameWithUser
	postJson := fmt.Sprintf(`{
	"apiVersion": "networking.k8s.io/v1",
	"kind": "Ingress",
	"metadata": {
		"namespace": "%s",
		"pureLabels": {},
		"resourceVersion": "%s",
		"uid": "%s",
		"name": "%s",
		"annotations": {
			"kubernetes.io/ingress.class": "istio",
			"nginx.ingress.kubernetes.io/load-balance": "round_robin"
		},
		"labels": {}
	},
	"spec": {
		"rules": [{
			"host": "poctest",
			"http": {
				"paths": [{
					"path": "/test2",
					"pathType": "ImplementationSpecific",
					"backend": {
						"service": {
							"name": "%s",
							"port": {
								"number": 8080
							}
						}
					}
				}]
			}
		}],
		"tls": []
	}
}`, framework.NamespaceName, ingress.ResourceVersion, ingress.UID, ingress1NameWithUser, svc1NameWithUser)
	resp, err := httpHelper.RequestByUser(http.MethodPut, framework.KubecubeHost+url, postJson, user, nil)
	framework.ExpectNoError(err)
	defer resp.Body.Close()

	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to update ingress %s", ingress1NameWithUser), resp.StatusCode)
	}

	// check return success
	framework.ExpectEqual(resp.StatusCode, http.StatusOK)
	return framework.SucceedResp
}

func checkIngress(user string) framework.TestResp {
	url := "/api/v1/cube/proxy/clusters/" + framework.TargetClusterName + "/apis/networking.k8s.io/v1/namespaces/" + framework.NamespaceName + "/ingresses/" + ingress1NameWithUser
	resp, err := httpHelper.RequestByUser(http.MethodGet, framework.KubecubeHost+url, "", user, nil)
	framework.ExpectNoError(err)
	defer resp.Body.Close()

	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to get ingress %s", ingress1NameWithUser), resp.StatusCode)
	}

	// check return success
	framework.ExpectEqual(resp.StatusCode, http.StatusOK)
	body, err := io.ReadAll(resp.Body)
	framework.ExpectNoError(err)
	ingress := networkingv1.Ingress{}
	err = json.Unmarshal(body, &ingress)
	framework.ExpectNoError(err)
	framework.ExpectEqual(ingress.Name, ingress1NameWithUser)
	framework.ExpectEqual(ingress.Spec.Rules[0].HTTP.Paths[0].Path, "/test2")
	return framework.SucceedResp
}

func deleteIngress(user string) framework.TestResp {
	url := "/api/v1/cube/proxy/clusters/" + framework.TargetClusterName + "/apis/networking.k8s.io/v1/namespaces/" + framework.NamespaceName + "/ingresses/" + ingress1NameWithUser
	resp, err := httpHelper.RequestByUser(http.MethodDelete, framework.KubecubeHost+url, "", user, nil)
	framework.ExpectNoError(err)
	defer resp.Body.Close()

	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to delete ingress %s", ingress1NameWithUser), resp.StatusCode)
	}

	// check return success
	framework.ExpectEqual(resp.StatusCode, http.StatusOK)
	ingress := networkingv1.Ingress{}
	err = framework.TargetConvertClient.Get(ctx, types.NamespacedName{Name: ingress1NameWithUser, Namespace: framework.NamespaceName}, &ingress)
	framework.ExpectEqual(true, kerrors.IsNotFound(err))
	return framework.SucceedResp
}

var multiUserIngressCRUDTest = framework.MultiUserTest{
	TestName:        "[ingress][9386667]创建负载均衡检查",
	ContinueIfError: false,
	SkipUsers:       []string{},
	Skipfunc:        nil,
	ErrorFunc:       framework.PermissionErrorFunc,
	AfterEach:       nil,
	BeforeEach:      nil,
	InitStep: &framework.MultiUserTestStep{
		Name:        "创建 deploy 和 svc",
		Description: "创建 deploy 和 svc",
		StepFunc:    createDeployAndSvc1,
	},
	FinalStep: &framework.MultiUserTestStep{
		Name:        "删除 deploy 和 svc",
		Description: "删除 deploy 和 svc",
		StepFunc:    deleteDeployAndSvc1,
	},
	Steps: []framework.MultiUserTestStep{
		{
			Name:        "创建ingress成功",
			Description: "创建ingress1: 域名设置为poctest; 路径设置为/test，选中服务service1，端口选择8080",
			StepFunc:    createIngress,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
		{
			Name:        "查询成功，查询结果列表长度为1",
			Description: "查询ingress列表",
			StepFunc:    listIngress,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
		{
			Name:        "更新成功",
			Description: "更新ingress1的配置路径设置为/test2",
			StepFunc:    updateIngress,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
		{
			Name:        "查询成功，查询结果与更新信息一致",
			Description: "查询ingress1的详情",
			StepFunc:    checkIngress,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
		{
			Name:        "删除成功",
			Description: "删除ingress1",
			StepFunc:    deleteIngress,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
	},
}

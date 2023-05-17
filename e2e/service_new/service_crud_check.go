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

package service_new

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/kubecube-io/kubecube/pkg/clog"
	v12 "k8s.io/api/apps/v1"
	v13 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubecube-io/kubecube-e2e/e2e/framework"
)

func createDeploy1AndDeploy2(user string) framework.TestResp {
	initParam()
	deploy1NameWithUser = framework.NameWithUser(deploy1Name, user)
	deploy2NameWithUser = framework.NameWithUser(deploy2Name, user)
	service1NameWithUser = framework.NameWithUser(service1Name, user)
	service2NameWithUser = framework.NameWithUser(service2Name, user)
	replica := int32(1)
	deploy1 = &v12.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name:      deploy1NameWithUser,
			Namespace: framework.NamespaceName,
		},
		Spec: v12.DeploymentSpec{
			Selector: &v1.LabelSelector{
				MatchLabels: map[string]string{"kubecube.io/app": deploy1NameWithUser},
			},
			Replicas: &replica,
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

	deploy2 = &v12.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name:      deploy2NameWithUser,
			Namespace: framework.NamespaceName,
		},
		Spec: v12.DeploymentSpec{
			Selector: &v1.LabelSelector{
				MatchLabels: map[string]string{"kubecube.io/app": deploy2NameWithUser},
			},
			Replicas: &replica,
			Template: v13.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{"kubecube.io/app": deploy2NameWithUser},
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
	err = framework.TargetClusterClient.Direct().Create(ctx, deploy2)
	framework.ExpectNoError(err)
	return framework.SucceedResp
}

func deleteDeploy1AndDeploy2(user string) framework.TestResp {
	err := framework.TargetClusterClient.Direct().Delete(ctx, deploy1)
	framework.ExpectNoError(err)
	err = framework.TargetClusterClient.Direct().Delete(ctx, deploy2)
	framework.ExpectNoError(err)
	return framework.SucceedResp
}

func createService1(user string) framework.TestResp {
	url := "/api/v1/cube/proxy/clusters/" + framework.TargetClusterName + "/api/v1/namespaces/" + framework.NamespaceName + "/services"
	postJson := fmt.Sprintf("{\"apiVersion\":\"v1\",\"kind\":\"Service\",\"metadata\":{\"name\":\"%s\",\"annotations\":{},\"labels\":{}},\"spec\":{\"ports\":[{\"name\":\"port1\",\"protocol\":\"TCP\",\"port\":8080,\"targetPort\":8080}],\"type\":\"ClusterIP\",\"selector\":{\"kubecube.io/app\":\"nginx\"},\"sessionAffinity\":\"None\"}}", service1NameWithUser)
	resp, err := httpHelper.RequestByUser(http.MethodPost, framework.KubecubeHost+url, postJson, user, nil)
	framework.ExpectNoError(err)
	defer resp.Body.Close()

	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to create svc %s", service1NameWithUser), resp.StatusCode)
	}
	// check return success
	framework.ExpectEqual(resp.StatusCode, http.StatusCreated)
	service := v13.Service{}
	err = framework.TargetClusterClient.Direct().Get(ctx, types.NamespacedName{Name: service1NameWithUser, Namespace: framework.NamespaceName}, &service)
	framework.ExpectNoError(err)
	framework.ExpectEqual(service.Name, service1NameWithUser)
	return framework.SucceedResp
}

func checkService1(user string) framework.TestResp {
	url := "/api/v1/cube/proxy/clusters/" + framework.TargetClusterName + "/api/v1/namespaces/" + framework.NamespaceName + "/services/" + service1NameWithUser
	resp, err := httpHelper.RequestByUser(http.MethodGet, framework.KubecubeHost+url, "", user, nil)
	framework.ExpectNoError(err)
	defer resp.Body.Close()

	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to get svc %s", service1NameWithUser), resp.StatusCode)
	}

	// check return success
	framework.ExpectEqual(resp.StatusCode, http.StatusOK)
	body, err := io.ReadAll(resp.Body)
	framework.ExpectNoError(err)
	svc := v13.Service{}
	err = json.Unmarshal(body, &svc)
	framework.ExpectNoError(err)
	framework.ExpectEqual(svc.Name, service1NameWithUser)
	framework.ExpectEqual(svc.Spec.Selector["kubecube.io/app"], "nginx")
	return framework.SucceedResp
}

func createService2(user string) framework.TestResp {
	url := "/api/v1/cube/proxy/clusters/" + framework.TargetClusterName + "/api/v1/namespaces/" + framework.NamespaceName + "/services"
	postJson := fmt.Sprintf("{\"apiVersion\":\"v1\",\"kind\":\"Service\",\"metadata\":{\"name\":\"%s\",\"annotations\":{},\"labels\":{}},\"spec\":{\"ports\":[{\"name\":\"demo-port\",\"protocol\":\"TCP\",\"port\":8080,\"targetPort\":8080}],\"type\":\"ClusterIP\",\"clusterIP\":\"None\",\"selector\":{},\"sessionAffinity\":\"None\"}}", service2NameWithUser)
	resp, err := httpHelper.RequestByUser(http.MethodPost, framework.KubecubeHost+url, postJson, user, nil)
	framework.ExpectNoError(err)
	defer resp.Body.Close()

	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to create svc %s", service2NameWithUser), resp.StatusCode)
	}

	// check return success
	framework.ExpectEqual(resp.StatusCode, http.StatusCreated)
	service := v13.Service{}
	err = framework.TargetClusterClient.Direct().Get(ctx, types.NamespacedName{Name: service2NameWithUser, Namespace: framework.NamespaceName}, &service)
	framework.ExpectNoError(err)
	framework.ExpectEqual(service.Spec.ClusterIP, "None")
	return framework.SucceedResp
}

func listService(user string) framework.TestResp {
	url := "/api/v1/cube/proxy/clusters/" + framework.TargetClusterName + "/api/v1/namespaces/" + framework.NamespaceName + "/services?pageNum=1&pageSize=10"
	resp, err := httpHelper.RequestByUser(http.MethodGet, framework.KubecubeHost+url, "", user, nil)
	framework.ExpectNoError(err)
	defer resp.Body.Close()

	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(errors.New("fail to list svc"), resp.StatusCode)
	}

	// check return success
	framework.ExpectEqual(resp.StatusCode, http.StatusOK)
	body, err := io.ReadAll(resp.Body)
	framework.ExpectNoError(err)
	clog.Debug("svc list %s", string(body))
	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	framework.ExpectNoError(err)
	framework.ExpectNotEqual(int(result["total"].(float64)), 0)

	list, ok := result["items"].([]interface{})
	framework.ExpectEqual(ok, true)

	count := 0
	for _, item := range list {
		name := item.(map[string]interface{})["metadata"].(map[string]interface{})["name"].(string)
		if name == service1NameWithUser || name == service2NameWithUser {
			count++
		}
	}

	framework.ExpectEqual(count, 2)

	return framework.SucceedResp
}

func updateService1(user string) framework.TestResp {
	service := &v13.Service{}
	err := framework.TargetClusterClient.Direct().Get(ctx, types.NamespacedName{Name: service1NameWithUser, Namespace: framework.NamespaceName}, service)
	framework.ExpectNoError(err)

	url := "/api/v1/cube/proxy/clusters/" + framework.TargetClusterName + "/api/v1/namespaces/" + framework.NamespaceName + "/services/" + service1NameWithUser
	postJson := fmt.Sprintf("{\"metadata\":{\"namespace\":\"%s\",\"pureLabels\":{},\"resourceVersion\":\"%s\",\"uid\":\"%s\",\"name\":\"%s\",\"annotations\":{},\"labels\":{}},\"spec\":{\"ports\":[{\"name\":\"port1\",\"protocol\":\"TCP\",\"port\":8080,\"targetPort\":8080},{\"name\":\"port2\",\"protocol\":\"TCP\",\"port\":50000,\"targetPort\":50000}],\"type\":\"ClusterIP\",\"clusterIP\":\"%s\",\"selector\":{\"kubecube.io/app\":\"nginx\"},\"sessionAffinity\":\"None\"}}",
		framework.NamespaceName, service.ResourceVersion, service.UID, service1NameWithUser, service.Spec.ClusterIP)
	resp, err := httpHelper.RequestByUser(http.MethodPut, framework.KubecubeHost+url, postJson, user, nil)
	framework.ExpectNoError(err)
	defer resp.Body.Close()

	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to update svc %s", service1NameWithUser), resp.StatusCode)
	}

	// check return success
	framework.ExpectEqual(resp.StatusCode, http.StatusOK)
	service = &v13.Service{}
	err = framework.TargetClusterClient.Direct().Get(ctx, types.NamespacedName{Name: service1NameWithUser, Namespace: framework.NamespaceName}, service)
	framework.ExpectNoError(err)
	framework.ExpectEqual(len(service.Spec.Ports), 2)
	return framework.SucceedResp
}

func deleteService(user string) framework.TestResp {
	url := "/api/v1/cube/proxy/clusters/" + framework.TargetClusterName + "/api/v1/namespaces/" + framework.NamespaceName + "/services/" + service2NameWithUser
	resp, err := httpHelper.RequestByUser(http.MethodDelete, framework.KubecubeHost+url, "", user, nil)
	framework.ExpectNoError(err)
	defer resp.Body.Close()

	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to delete svc %s", service2NameWithUser), resp.StatusCode)
	}

	// check return success
	framework.ExpectEqual(resp.StatusCode, http.StatusOK)
	service := v13.Service{}
	err = framework.TargetClusterClient.Direct().Get(ctx, types.NamespacedName{Name: service2NameWithUser, Namespace: framework.NamespaceName}, &service)
	framework.ExpectEqual(true, kerrors.IsNotFound(err))

	url = "/api/v1/cube/proxy/clusters/" + framework.TargetClusterName + "/api/v1/namespaces/" + framework.NamespaceName + "/services/" + service1NameWithUser
	resp, err = httpHelper.RequestByUser(http.MethodDelete, framework.KubecubeHost+url, "", user, nil)
	framework.ExpectNoError(err)
	defer resp.Body.Close()

	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to deletesvc %s", service1NameWithUser), resp.StatusCode)
	}

	// check return success
	serviceList := v13.ServiceList{}
	err = framework.TargetClusterClient.Direct().List(ctx, &serviceList, &client.ListOptions{Namespace: framework.NamespaceName})
	framework.ExpectNoError(err)

	count := 0
	for _, item := range serviceList.Items {
		if item.Name == service1NameWithUser || item.Name == service2NameWithUser {
			count++
		}
	}
	framework.ExpectEqual(count, 0)

	return framework.SucceedResp
}

var multiUserServiceCRUDTest = framework.MultiUserTest{
	TestName:        "[service][9386657]服务检查的增删改查",
	ContinueIfError: false,
	Skipfunc:        nil,
	ErrorFunc:       framework.PermissionErrorFunc,
	AfterEach:       nil,
	BeforeEach:      nil,
	InitStep: &framework.MultiUserTestStep{
		Name:        "创建 deploy1 deploy2",
		Description: "创建 deploy1 deploy2",
		StepFunc:    createDeploy1AndDeploy2,
	},
	FinalStep: &framework.MultiUserTestStep{
		Name:        "删除 deploy1 deploy2",
		Description: "删除 deploy1 deploy2",
		StepFunc:    deleteDeploy1AndDeploy2,
	},
	Steps: []framework.MultiUserTestStep{
		{
			Name:        "创建service1成功",
			Description: "创建服务service1，标签选择器选择简单模式，关联负载deploy1",
			StepFunc:    createService1,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
		{
			Name:        "查询成功，查询结果与创建内容一致",
			Description: "查询service1详情",
			StepFunc:    checkService1,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
		{
			Name:        "创建service2成功，service2的ClusterIP为None",
			Description: "创建Headless类型服务service2，关联负载deploy2",
			StepFunc:    createService2,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
		{
			Name:        "空间下的svc总数为2",
			Description: "查询空间下的服务列表",
			StepFunc:    listService,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
		{
			Name:        "更新配置生效",
			Description: "更新service1，端口配置增加 50000：50000",
			StepFunc:    updateService1,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
		{
			Name:        "删除成功，查询列表结果与删除动作一致",
			Description: "删除service1 service2",
			StepFunc:    deleteService,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
	},
}

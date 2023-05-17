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
	"fmt"
	"io"
	"net/http"

	"github.com/kubecube-io/kubecube/pkg/clog"
	"github.com/onsi/ginkgo"
	v12 "k8s.io/api/apps/v1"
	v13 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubecube-io/kubecube-e2e/e2e/framework"
)

func createDeployAndServiceForNodeport(user string) framework.TestResp {
	initParam()
	deploy1NameWithUser = framework.NameWithUser(deploy1Name, user)
	service1NameWithUser = framework.NameWithUser(service1Name, user)
	replica := int32(1)

	ns1 = &v13.Namespace{}
	err := cli.Get(ctx, client.ObjectKey{Name: framework.NamespaceName}, ns1)
	if errors.IsNotFound(err) {
		nsExist = false
		ns1 = &v13.Namespace{
			ObjectMeta: v1.ObjectMeta{
				Name: framework.NamespaceName,
			},
		}
		errInfo := cli.Create(ctx, ns1)
		framework.ExpectNoError(errInfo)
	}

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
	err = cli.Create(ctx, deploy1)
	framework.ExpectNoError(err)

	svc1 = &v13.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:      service1NameWithUser,
			Namespace: framework.NamespaceName,
		},
		Spec: v13.ServiceSpec{
			Selector: map[string]string{"kubecube.io/app": deploy1NameWithUser},
			Ports: []v13.ServicePort{
				{Name: "port1", Protocol: "TCP", Port: 80, TargetPort: intstr.FromInt(80)},
			},
		},
	}
	err = cli.Create(ctx, svc1)
	framework.ExpectNoError(err)
	return framework.SucceedResp
}

func deleteDeployAndServiceForNodeport(user string) framework.TestResp {
	framework.ExpectNoError(cli.Delete(ctx, deploy1))
	framework.ExpectNoError(cli.Delete(ctx, svc1))
	if !nsExist {
		framework.ExpectNoError(cli.Delete(ctx, ns1))
		err := wait.Poll(waitInterval, waitTimeout,
			func() (bool, error) {
				var namespace v13.Namespace
				errInfo := cli.Get(ctx, types.NamespacedName{Name: framework.NamespaceName}, &namespace)
				if errors.IsNotFound(errInfo) {
					return true, nil
				} else {
					return false, nil
				}
			})
		framework.ExpectNoError(err)
	}
	return framework.SucceedResp
}

func checkExternalAccess(user string) framework.TestResp {
	ginkgo.By("1. 设置对外服务端口80：1111")
	url := fmt.Sprintf("/api/v1/cube/extend/clusters/%s/namespaces/%s/externalAccess/%s", framework.PivotClusterName, framework.NamespaceName, service1NameWithUser)
	postJson := fmt.Sprintf("[{\"protocol\":\"TCP\",\"servicePort\":80,\"externalPorts\":[%d]}]", portMap[user])
	resp, err := httpHelper.RequestByUser(http.MethodPost, framework.KubecubeHost+url, postJson, user, nil)
	framework.ExpectNoError(err)
	defer resp.Body.Close()

	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to update svc %s", service1NameWithUser), resp.StatusCode)
	}

	framework.ExpectEqual(resp.StatusCode, http.StatusOK)

	ginkgo.By("2. 在页面提示的访问地址（node ip）中选取一个IP1；登陆到可访问node节点机的部署机器执行：curl http://IP1:1111")
	url = fmt.Sprintf("/api/v1/cube/extend/clusters/%s/namespaces/%s/externalAccessAddress", framework.PivotClusterName, framework.NamespaceName)
	resp, err = httpHelper.Get(framework.KubecubeHost+url, nil)
	framework.ExpectNoError(err)
	defer resp.Body.Close()
	framework.ExpectEqual(resp.StatusCode, http.StatusOK)

	var result []string
	body, err := io.ReadAll(resp.Body)
	clog.Debug("externalAccessAddress %s", string(body))
	err = json.Unmarshal(body, &result)
	framework.ExpectNoError(err)
	var ip string
	if len(result) > 0 {
		ip = result[0]
		url = fmt.Sprintf("http://%s:%d", ip, portMap[user])
		err = wait.Poll(waitInterval, waitTimeout,
			func() (bool, error) {
				resp, err = httpHelper.Get(url, nil)
				if err == nil && resp.StatusCode == http.StatusOK {
					defer resp.Body.Close()
					return true, nil
				} else {
					return false, nil
				}
			})
	}

	ginkgo.By("3. 更改设置对外服务端口从80：1111到80：1114")
	url = fmt.Sprintf("/api/v1/cube/extend/clusters/%s/namespaces/%s/externalAccessAddress", framework.PivotClusterName, framework.NamespaceName)
	postJson = fmt.Sprintf("[{\"protocol\":\"TCP\",\"servicePort\":80,\"externalPorts\":[%d]}]", newportMap[user])
	resp, err = httpHelper.RequestByUser(http.MethodPost, framework.KubecubeHost+url, postJson, user, nil)
	framework.ExpectNoError(err)
	defer resp.Body.Close()

	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to update externalAccessAddress"), resp.StatusCode)
	}

	framework.ExpectEqual(resp.StatusCode, http.StatusOK)

	url = fmt.Sprintf("http://%s:%d", ip, portMap[user])
	err = wait.Poll(waitInterval, waitTimeout,
		func() (bool, error) {
			resp, err = httpHelper.Get(url, nil)
			if err != nil {
				return true, nil
			} else {
				defer resp.Body.Close()
				return false, nil
			}
		})

	url = fmt.Sprintf("http://%s:%d", ip, newportMap[user])
	err = wait.Poll(waitInterval, waitTimeout,
		func() (bool, error) {
			resp, err = httpHelper.Get(url, nil)
			if err == nil && resp.StatusCode == http.StatusOK {
				defer resp.Body.Close()
				return true, nil
			} else {
				return false, nil
			}
		})

	ginkgo.By("4. 更改设置对外服务端口从on到off")
	url = fmt.Sprintf("/api/v1/cube/extend/clusters/%s/namespaces/%s/externalAccessAddress", framework.PivotClusterName, framework.NamespaceName)
	postJson = "[{protocol: \"TCP\", servicePort: 80}]"
	resp, err = httpHelper.RequestByUser(http.MethodPost, framework.KubecubeHost+url, postJson, user, nil)
	framework.ExpectNoError(err)
	defer resp.Body.Close()

	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to update externalAccessAddress"), resp.StatusCode)
	}

	framework.ExpectEqual(resp.StatusCode, http.StatusOK)

	url = fmt.Sprintf("http://%s:%d", ip, newportMap[user])
	err = wait.Poll(waitInterval, waitTimeout,
		func() (bool, error) {
			resp, err = httpHelper.Get(url, nil)
			if err != nil {
				return true, nil
			} else {
				defer resp.Body.Close()
				return false, nil
			}
		})
	return framework.SucceedResp
}

var multiUserServiceNodeportTest = framework.MultiUserTest{
	TestName:        "[service][9386659]服务对容器云外暴露访问检查",
	ContinueIfError: false,
	Skipfunc:        nil,
	SkipUsers:       []string{framework.UserNormal, framework.UserTenantAdmin, framework.UserProjectAdmin},
	ErrorFunc:       framework.PermissionErrorFunc,
	AfterEach:       nil,
	BeforeEach:      nil,
	InitStep: &framework.MultiUserTestStep{
		Name:        "创建 nodeport",
		Description: "创建 nodeport",
		StepFunc:    createDeployAndServiceForNodeport,
	},
	FinalStep: &framework.MultiUserTestStep{
		Name:        "删除 nodeport",
		Description: "删除 nodeport",
		StepFunc:    deleteDeployAndServiceForNodeport,
	},
	Steps: []framework.MultiUserTestStep{
		{
			Name:     "2.可访问到负载部署的应用; 3.curl IP1:1111不通，curl IP1:1114可访问; 4.无对外设置端口curl IP1:1114亦不可访问",
			StepFunc: checkExternalAccess,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
	},
}

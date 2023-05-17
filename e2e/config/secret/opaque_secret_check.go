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

package secret

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kubecube-io/kubecube/pkg/clog"
	v1 "k8s.io/api/core/v1"
	client2 "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubecube-io/kubecube-e2e/e2e/framework"
)

func createOpaqueSecret(user string) framework.TestResp {
	initParam()
	secretNameWithUser := framework.NameWithUser(opaqueSecretName, user)
	postJsonOfCreateSecret := `{"apiVersion":"v1","kind":"Secret","metadata":{"name":"%s","namespace":"%s"},"type":"Opaque","data":{"username":"YWRtaW4=","password":"MTIzNDU2"}}`
	postJsonOfCreateSecret = fmt.Sprintf(postJsonOfCreateSecret, secretNameWithUser, namespace)
	urlOfCreateSecret := "%s/api/v1/cube/proxy/clusters/%s/api/v1/namespaces/%s/secrets"
	urlOfCreateSecret = fmt.Sprintf(urlOfCreateSecret, framework.KubecubeHost, clusterName, namespace)
	respOfCreateSecret, err := httpHelper.RequestByUser(http.MethodPost, urlOfCreateSecret, postJsonOfCreateSecret, user, nil)
	defer respOfCreateSecret.Body.Close()
	_, err = io.ReadAll(respOfCreateSecret.Body)
	framework.ExpectNoError(err)

	if !framework.IsSuccess(respOfCreateSecret.StatusCode) {
		clog.Warn("res code %d", respOfCreateSecret.StatusCode)
		return framework.NewTestResp(errors.New("fail to create opaqueSecret"), respOfCreateSecret.StatusCode)
	}

	checkOfCreateSecret := &v1.Secret{}
	err = cli.Direct().Get(context.Background(), client2.ObjectKey{
		Namespace: namespace,
		Name:      secretNameWithUser,
	}, checkOfCreateSecret)
	framework.ExpectNoError(err, "secret should be created")
	return framework.SucceedResp
}

func createPodWithSecretVolume(user string) framework.TestResp {
	initParam()
	podNameWithUser := framework.NameWithUser(opaqueSecretPodName, user)
	secretNameWithUser := framework.NameWithUser(opaqueSecretName, user)
	postJsonOfCreatePodWithSecret := `{"apiVersion":"v1","kind":"Pod","metadata":{"name":"%s","namespace":"%s"},"spec":{"imagePullSecrets": [{"name": "%s"}],"affinity":{"nodeAffinity":{"requiredDuringSchedulingIgnoredDuringExecution":{"nodeSelectorTerms":[{"matchExpressions":[{"key":"node.kubecube.io/tenant","operator":"In","values":["share"]}]}]}}},"containers":[{"name":"mypod","image":"%s","imagePullPolicy": "IfNotPresent","command":[],"resources":{"limits":{"cpu":"100m","memory":"128Mi"},"requests":{"cpu":"100m","memory":"128Mi"}},"volumeMounts":[{"name":"foo","mountPath":"/mnt/secret"}]}],"volumes":[{"name":"foo","secret":{"secretName":"%s","optional":false}}]}}`
	postJsonOfCreatePodWithSecret = fmt.Sprintf(postJsonOfCreatePodWithSecret, podNameWithUser, namespace, framework.ImagePullSecret, framework.TestImage, secretNameWithUser)
	urlOfCreatePodWithSecret := "%s/api/v1/cube/proxy/clusters/%s/api/v1/namespaces/%s/pods"
	urlOfCreatePodWithSecret = fmt.Sprintf(urlOfCreatePodWithSecret, framework.KubecubeHost, clusterName, namespace)
	respOfCreatePodWithSecret, err := httpHelper.RequestByUser(http.MethodPost, urlOfCreatePodWithSecret, postJsonOfCreatePodWithSecret, user, nil)
	defer respOfCreatePodWithSecret.Body.Close()
	_, err = io.ReadAll(respOfCreatePodWithSecret.Body)
	framework.ExpectNoError(err)

	if !framework.IsSuccess(respOfCreatePodWithSecret.StatusCode) {
		clog.Warn("res code %d", respOfCreatePodWithSecret.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to create pod %s", podNameWithUser), respOfCreatePodWithSecret.StatusCode)
	}

	time.Sleep(time.Second * 30)
	checkOfCreatePodWithSecret := &v1.Pod{}
	err = cli.Direct().Get(context.Background(), client2.ObjectKey{
		Namespace: namespace,
		Name:      podNameWithUser,
	}, checkOfCreatePodWithSecret)
	framework.ExpectNoError(err, "pod should be created")
	framework.ExpectEqual(string(checkOfCreatePodWithSecret.Status.Phase), "Running", "pod should be running")
	return framework.SucceedResp
}

func createPodWithSecretEnv(user string) framework.TestResp {
	initParam()
	podNameWithUser := framework.NameWithUser(opaqueSecretEnvPodName, user)
	secretNameWithUser := framework.NameWithUser(opaqueSecretName, user)
	postJsonOfCreatePodWithSecretENV := `{"apiVersion":"v1","kind":"Pod","metadata":{"name":"%s","namespace":"%s"},"spec":{"imagePullSecrets": [{"name": "%s"}],"affinity":{"nodeAffinity":{"requiredDuringSchedulingIgnoredDuringExecution":{"nodeSelectorTerms":[{"matchExpressions":[{"key":"node.kubecube.io/tenant","operator":"In","values":["share"]}]}]}}},"containers":[{"name":"mypod","image":"%s","env":[{"name":"SECRET_USERNAME","valueFrom":{"secretKeyRef":{"name":"%s","key":"username","optional":false}}},{"name":"SECRET_PASSWORD","valueFrom":{"secretKeyRef":{"name":"%s","key":"password","optional":false}}}],"command":["sleep","2m"],"resources":{"limits":{"cpu":"100m","memory":"128Mi"},"requests":{"cpu":"100m","memory":"128Mi"}}}]}}`
	postJsonOfCreatePodWithSecretENV = fmt.Sprintf(postJsonOfCreatePodWithSecretENV, podNameWithUser, namespace, framework.ImagePullSecret, framework.TestImage, secretNameWithUser, secretNameWithUser)
	urlOfCreatePodWithSecretENV := "%s/api/v1/cube/proxy/clusters/%s/api/v1/namespaces/%s/pods"
	urlOfCreatePodWithSecretENV = fmt.Sprintf(urlOfCreatePodWithSecretENV, framework.KubecubeHost, clusterName, namespace)
	respOfCreatePodWithSecretENV, err := httpHelper.RequestByUser(http.MethodPost, urlOfCreatePodWithSecretENV, postJsonOfCreatePodWithSecretENV, user, nil)
	defer respOfCreatePodWithSecretENV.Body.Close()
	_, err = io.ReadAll(respOfCreatePodWithSecretENV.Body)
	framework.ExpectNoError(err)

	if !framework.IsSuccess(respOfCreatePodWithSecretENV.StatusCode) {
		clog.Warn("res code %d", respOfCreatePodWithSecretENV.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to create pod %s", podNameWithUser), respOfCreatePodWithSecretENV.StatusCode)
	}

	time.Sleep(time.Second * 30)
	checkOfCreatePodWithSecretENV := &v1.Pod{}
	err = cli.Direct().Get(context.Background(), client2.ObjectKey{
		Namespace: namespace,
		Name:      podNameWithUser,
	}, checkOfCreatePodWithSecretENV)
	framework.ExpectNoError(err, "pod should be created")
	framework.ExpectEqual(string(checkOfCreatePodWithSecretENV.Status.Phase), "Running", "pod should be running")

	return framework.SucceedResp
}

func deletePodWithSecretVolume(user string) framework.TestResp {
	initParam()
	podNameWithUser := framework.NameWithUser(opaqueSecretEnvPodName, user)
	urlOfDeletePodWithSecret := "%s/api/v1/cube/proxy/clusters/%s/api/v1/namespaces/%s/pods/%s"
	urlOfDeletePodWithSecret = fmt.Sprintf(urlOfDeletePodWithSecret, framework.KubecubeHost, clusterName, namespace, podNameWithUser)
	_, err := httpHelper.Delete(urlOfDeletePodWithSecret)
	framework.ExpectNoError(err)
	return framework.SucceedResp
}

func deletePodWithSecretEnv(user string) framework.TestResp {
	initParam()
	podNameWithUser := framework.NameWithUser(opaqueSecretEnvPodName, user)
	urlOfDeletePodWithSecretENV := "%s/api/v1/cube/proxy/clusters/%s/api/v1/namespaces/%s/pods/%s"
	urlOfDeletePodWithSecretENV = fmt.Sprintf(urlOfDeletePodWithSecretENV, framework.KubecubeHost, clusterName, namespace, podNameWithUser)
	_, err := httpHelper.Delete(urlOfDeletePodWithSecretENV)
	framework.ExpectNoError(err)

	return framework.SucceedResp
}

func deleteOpaqueSecret(user string) framework.TestResp {
	initParam()
	secretNameWithUser := framework.NameWithUser(opaqueSecretName, user)
	urlOfDeleteSecret := "%s/api/v1/cube/proxy/clusters/%s/api/v1/namespaces/%s/secrets/%s"
	urlOfDeleteSecret = fmt.Sprintf(urlOfDeleteSecret, framework.KubecubeHost, clusterName, namespace, secretNameWithUser)
	_, err := httpHelper.Delete(urlOfDeleteSecret)
	framework.ExpectNoError(err)
	return framework.SucceedResp
}

var multiUserOpaqueSecretTest = framework.MultiUserTest{
	TestName:        "[配置][9387665]Opaque类型秘钥检查",
	ContinueIfError: false,
	Skipfunc:        nil,
	ErrorFunc:       framework.PermissionErrorFunc,
	AfterEach:       nil,
	BeforeEach:      nil,
	Steps: []framework.MultiUserTestStep{
		{
			Name:        "创建secret",
			Description: "1. 创建Opaque类型秘钥secret2，\nusername=admin\npasswd=123456\n【注意】：实际填入输入框值为base64编码格式，分别输入：\nusername=YWRtaW4=\npasswd=MTIzNDU2",
			StepFunc:    createOpaqueSecret,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
		{
			Name:        "创建挂载secret volume的pod",
			Description: "2. 负载挂载secret\n创建负载》容器》高级模式中挂载数据卷选择secret类型，将secret2挂载到/mnt/secret目录",
			StepFunc:    createPodWithSecretVolume,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
		{
			Name:        "创建注入secret env的pod",
			Description: "3. 负载从secret中读取环境变量\n容器》高级模式》设置容器环境变量，选择Value类型为Secret\n分别添加username、passwd",
			StepFunc:    createPodWithSecretEnv,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
		{
			Name:        "删除挂载secret volume的pod",
			Description: "4. 删除挂载secret volume的pod",
			StepFunc:    deletePodWithSecretVolume,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
		{
			Name:        "删除注入secret env的pod",
			Description: "5. 删除注入secret env的pod",
			StepFunc:    deletePodWithSecretEnv,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
		{
			Name:        "删除secret",
			Description: "6. 删除secret",
			StepFunc:    deleteOpaqueSecret,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
	},
}

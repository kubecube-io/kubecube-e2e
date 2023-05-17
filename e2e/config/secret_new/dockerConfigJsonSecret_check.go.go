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

package secret_new

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

func createDockerConfigJsonSecret(user string) framework.TestResp {
	initParam()
	secretNameWithUser := framework.NameWithUser(secretName, user)
	postJsonOfCreateSecret := `{"apiVersion":"v1","kind":"Secret","metadata":{"name":"%s","namespace":"%s"},"data":{".dockerconfigjson":"eyAiYXV0aHMiOiB7ICJoYXJib3IuY2xvdWQubmV0ZWFzZS5jb20iOiB7ICJhdXRoIjogImMydHBabVk2U2tSRllYY3lNMEJxWm1SWEl6Yz0iIH0gfSwgIkh0dHBIZWFkZXJzIjogeyAiVXNlci1BZ2VudCI6ICJEb2NrZXItQ2xpZW50LzE5LjAzLjEzIChsaW51eCkiIH0gfQo="},"type":"kubernetes.io/dockerconfigjson"}`
	postJsonOfCreateSecret = fmt.Sprintf(postJsonOfCreateSecret, secretNameWithUser, namespace)
	urlOfCreateSecret := "%s/api/v1/cube/proxy/clusters/%s/api/v1/namespaces/%s/secrets"
	urlOfCreateSecret = fmt.Sprintf(urlOfCreateSecret, framework.KubecubeHost, clusterName, namespace)
	respOfCreateSecret, err := httpHelper.RequestByUser(http.MethodPost, urlOfCreateSecret, postJsonOfCreateSecret, user, nil)
	framework.ExpectNoError(err)
	defer respOfCreateSecret.Body.Close()
	body, err := io.ReadAll(respOfCreateSecret.Body)
	framework.ExpectNoError(err)
	clog.Debug("create secret resp %+v", string(body))

	if !framework.IsSuccess(respOfCreateSecret.StatusCode) {
		clog.Warn("res code %d", respOfCreateSecret.StatusCode)
		return framework.NewTestResp(errors.New("fail to create dockerConfigJsonSecret"), respOfCreateSecret.StatusCode)
	}

	return framework.SucceedResp
}

func createPod(user string) framework.TestResp {
	initParam()
	podNameWithUser := framework.NameWithUser(podName, user)
	postJsonOfCreatePodWithSecret := `{"apiVersion":"v1","kind":"Pod","metadata":{"name":"%s","namespace":"%s"},"spec":{"imagePullSecrets":[{"name":"e2e-test-docker-secret"},{"name":"%s"}],"affinity":{"nodeAffinity":{"requiredDuringSchedulingIgnoredDuringExecution":{"nodeSelectorTerms":[{"matchExpressions":[{"key":"node.kubecube.io/tenant","operator":"In","values":["share"]}]}]}}},"containers":[{"name":"mypod","image":"%s","imagePullPolicy": "IfNotPresent","command":[],"resources":{"limits":{"cpu":"100m","memory":"128Mi"},"requests":{"cpu":"100m","memory":"128Mi"}}}]}}`
	postJsonOfCreatePodWithSecret = fmt.Sprintf(postJsonOfCreatePodWithSecret, podNameWithUser, namespace, framework.ImagePullSecret, framework.TestImage)
	urlOfCreatePodWithSecret := "%s/api/v1/cube/proxy/clusters/%s/api/v1/namespaces/%s/pods"
	urlOfCreatePodWithSecret = fmt.Sprintf(urlOfCreatePodWithSecret, framework.KubecubeHost, clusterName, namespace)
	respOfCreatePodWithSecret, err := httpHelper.RequestByUser(http.MethodPost, urlOfCreatePodWithSecret, postJsonOfCreatePodWithSecret, user, nil)
	framework.ExpectNoError(err)
	defer respOfCreatePodWithSecret.Body.Close()
	body, err := io.ReadAll(respOfCreatePodWithSecret.Body)
	framework.ExpectNoError(err)
	clog.Debug("create pod resp %+v", string(body))

	if !framework.IsSuccess(respOfCreatePodWithSecret.StatusCode) {
		clog.Warn("res code %d", respOfCreatePodWithSecret.StatusCode)
		return framework.NewTestResp(errors.New("fail to create pod"), respOfCreatePodWithSecret.StatusCode)
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

func deletePod(user string) framework.TestResp {
	initParam()
	podNameWithUser := framework.NameWithUser(podName, user)
	urlOfDeletePodWithSecret := "%s/api/v1/cube/proxy/clusters/%s/api/v1/namespaces/%s/pods/%s"
	urlOfDeletePodWithSecret = fmt.Sprintf(urlOfDeletePodWithSecret, framework.KubecubeHost, clusterName, namespace, podNameWithUser)
	_, err := httpHelper.Delete(urlOfDeletePodWithSecret)
	framework.ExpectNoError(err)
	return framework.SucceedResp
}

func deleteSecret(user string) framework.TestResp {
	initParam()
	secretNameWithUser := framework.NameWithUser(secretName, user)
	urlOfDeleteSecret := "%s/api/v1/cube/proxy/clusters/%s/api/v1/namespaces/%s/secrets/%s"
	urlOfDeleteSecret = fmt.Sprintf(urlOfDeleteSecret, framework.KubecubeHost, clusterName, namespace, secretNameWithUser)
	_, err := httpHelper.Delete(urlOfDeleteSecret)
	framework.ExpectNoError(err)
	return framework.SucceedResp
}

var multiUserDockerConfigJsonSecretTest = framework.MultiUserTest{
	TestName:        "[配置][9387664]DockerConfigJson类型秘钥检查",
	ContinueIfError: false,
	Skipfunc:        nil,
	ErrorFunc:       framework.PermissionErrorFunc,
	AfterEach:       nil,
	BeforeEach:      nil,
	Steps: []framework.MultiUserTestStep{
		{
			Name:        "创建secret",
			Description: "1. 创建DockerConfigJson类型秘钥secret1\n",
			StepFunc:    createDockerConfigJsonSecret,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
		{
			Name:        "创建pod",
			Description: "2. 创建负载指定secret1",
			StepFunc:    createPod,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
		{
			Name:        "删除pod",
			Description: "3. 删除pod",
			StepFunc:    deletePod,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
		{
			Name:        "删除secret",
			Description: "4. 删除secret",
			StepFunc:    deleteSecret,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
	},
}

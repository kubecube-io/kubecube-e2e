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

package configmap_new

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kubecube-io/kubecube/pkg/clog"
	"github.com/kubecube-io/kubecube/pkg/multicluster/client"
	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	client2 "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubecube-io/kubecube-e2e/e2e/framework"
)

var (
	clusterName string
	httpHelper  *framework.HttpHelper
	namespace   string
	cli         client.Client

	cmName  string
	podName string

	bodyOfCreateCM []byte
)

func initParam() {
	clusterName = framework.TargetClusterName
	httpHelper = framework.NewSingleHttpHelper()
	namespace = framework.NamespaceName
	cli = framework.TargetClusterClient
	cmName = "e2e-cm-test"
	podName = "e2e-pod-cm-test"
}

func createCM(user string) framework.TestResp {
	clog.Info("=========createCM========")
	initParam()
	cmNameByUser := user + "-" + cmName
	postJsonOfCreateCM := `{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"%s","namespace":"%s"},"data":{"key1":"value1","key2":"value2"}}`
	postJsonOfCreateCM = fmt.Sprintf(postJsonOfCreateCM, cmNameByUser, namespace)
	urlOfCreateCM := "%s/api/v1/cube/proxy/clusters/%s/api/v1/namespaces/%s/configmaps"
	urlOfCreateCM = fmt.Sprintf(urlOfCreateCM, framework.KubecubeHost, clusterName, namespace)
	respOfCreateCM, err := httpHelper.RequestByUser(http.MethodPost, urlOfCreateCM, postJsonOfCreateCM, user, nil)
	framework.ExpectNoError(err)
	defer respOfCreateCM.Body.Close()
	bodyOfCreateCM, err = io.ReadAll(respOfCreateCM.Body)
	framework.ExpectNoError(err)
	clog.Info(string(bodyOfCreateCM))

	if !framework.IsSuccess(respOfCreateCM.StatusCode) {
		clog.Warn("res code %d", respOfCreateCM.StatusCode)
		return framework.NewTestResp(errors.New("fail to create cm"), respOfCreateCM.StatusCode)
	}

	checkOfCreateCM := &v1.ConfigMap{}
	err = cli.Direct().Get(context.Background(), client2.ObjectKey{
		Namespace: namespace,
		Name:      cmNameByUser,
	}, checkOfCreateCM)
	framework.ExpectNoError(err, "new configmap should be retrieved")

	return framework.SucceedResp

}

func createPodAndCheck(user string) framework.TestResp {
	clog.Info("=========createPodAndCheck========")
	initParam()
	podNameByUser := user + "-" + podName
	cmNameByUser := user + "-" + cmName
	postJsonOfCreatePodWithCM := `{"apiVersion":"v1","kind":"Pod","metadata":{"name":"%s","namespace":"%s"},"spec":{"imagePullSecrets": [{"name": "%s"}],"containers":[{"name":"mypod","image":"%s","imagePullPolicy": "IfNotPresent","env":[{"name":"KEY1","valueFrom":{"configMapKeyRef":{"name":"%s","key":"key1"}}}],"command":["sleep","2m"],"resources":{"limits":{"cpu":"100m","memory":"128Mi"},"requests":{"cpu":"100m","memory":"128Mi"}}}]}}`
	postJsonOfCreatePodWithCM = fmt.Sprintf(postJsonOfCreatePodWithCM, podNameByUser, namespace, framework.ImagePullSecret, framework.TestImage, cmNameByUser)
	urlOfCreatePodWithCM := "%s/api/v1/cube/proxy/clusters/%s/api/v1/namespaces/%s/pods"
	urlOfCreatePodWithCM = fmt.Sprintf(urlOfCreatePodWithCM, framework.KubecubeHost, clusterName, namespace)
	respOfCreatePodWithCM, err := httpHelper.RequestByUser(http.MethodPost, urlOfCreatePodWithCM, postJsonOfCreatePodWithCM, user, nil)
	framework.ExpectNoError(err)
	defer respOfCreatePodWithCM.Body.Close()

	body, err := io.ReadAll(respOfCreatePodWithCM.Body)
	framework.ExpectNoError(err)
	clog.Info(string(body))

	if !framework.IsSuccess(respOfCreatePodWithCM.StatusCode) {
		clog.Warn("res code %d", respOfCreatePodWithCM.StatusCode)
		return framework.NewTestResp(errors.New("fail to create pod"), respOfCreatePodWithCM.StatusCode)
	}

	time.Sleep(time.Second * 10)
	checkOfCreatePodWithCM := &v1.Pod{}
	err = wait.Poll(framework.WaitInterval, framework.WaitTimeout, func() (done bool, err error) {
		err = cli.Direct().Get(context.Background(), client2.ObjectKey{
			Namespace: namespace,
			Name:      podNameByUser,
		}, checkOfCreatePodWithCM)
		if err != nil || checkOfCreatePodWithCM.Status.Phase != "Running" {
			return false, err
		} else {
			return true, nil
		}
	})

	framework.ExpectNoError(err, "pod should be created")
	framework.ExpectEqual(string(checkOfCreatePodWithCM.Status.Phase), "Running", "pod should be running")

	return framework.SucceedResp
}

func updateConfigMap(user string) framework.TestResp {
	clog.Info("=========updateConfigMap========")
	initParam()

	cmNameByUser := user + "-" + cmName
	var newCM map[string]interface{}
	err := json.Unmarshal(bodyOfCreateCM, &newCM)
	framework.ExpectNoError(err)
	newCM["data"].(map[string]interface{})["key2"] = "newValue"

	postJsonOfUpdateCM, _ := json.Marshal(newCM)
	urlOfUpdateCM := "%s/api/v1/cube/proxy/clusters/%s/api/v1/namespaces/%s/configmaps/%s"
	urlOfUpdateCM = fmt.Sprintf(urlOfUpdateCM, framework.KubecubeHost, clusterName, namespace, cmNameByUser)
	respOfUpdateCM, err := httpHelper.RequestByUser(http.MethodPut, urlOfUpdateCM, string(postJsonOfUpdateCM), user, nil)
	defer respOfUpdateCM.Body.Close()
	framework.ExpectNoError(err)

	body, err := io.ReadAll(respOfUpdateCM.Body)
	framework.ExpectNoError(err)
	clog.Info(string(body))

	if !framework.IsSuccess(respOfUpdateCM.StatusCode) {
		clog.Warn("res code %d", respOfUpdateCM.StatusCode)
		return framework.NewTestResp(errors.New("fail to update cm"), respOfUpdateCM.StatusCode)
	}

	checkOfUpdateCM := &v1.ConfigMap{}
	err = cli.Direct().Get(context.Background(), client2.ObjectKey{
		Namespace: namespace,
		Name:      cmNameByUser,
	}, checkOfUpdateCM)
	framework.ExpectNoError(err, "new configmap should be retrieved")
	framework.ExpectEqual(checkOfUpdateCM.Data["key2"], "newValue", "new value should be updated")

	return framework.SucceedResp
}

func deleteConfigMap(user string) framework.TestResp {
	clog.Info("=========deleteConfigMap========")
	initParam()
	cmNameByUser := user + "-" + cmName
	urlOfDeleteCM := "%s/api/v1/cube/proxy/clusters/%s/api/v1/namespaces/%s/configmaps/%s"
	urlOfDeleteCM = fmt.Sprintf(urlOfDeleteCM, framework.KubecubeHost, clusterName, namespace, cmNameByUser)
	respOfDeleteCM, err := httpHelper.RequestByUser(http.MethodDelete, urlOfDeleteCM, "", user, nil)
	defer respOfDeleteCM.Body.Close()
	framework.ExpectNoError(err)

	body, err := io.ReadAll(respOfDeleteCM.Body)
	framework.ExpectNoError(err)
	clog.Info(string(body))

	if !framework.IsSuccess(respOfDeleteCM.StatusCode) {
		clog.Warn("res code %d", respOfDeleteCM.StatusCode)
		return framework.NewTestResp(errors.New("fail to delete cm"), respOfDeleteCM.StatusCode)
	}

	checkOfDeleteCM := &v1.ConfigMap{}
	err = cli.Direct().Get(context.Background(), client2.ObjectKey{
		Namespace: namespace,
		Name:      cmNameByUser,
	}, checkOfDeleteCM)
	framework.ExpectEqual(kerrors.IsNotFound(err), true, "CM should be deleted")
	return framework.SucceedResp
}

func clearPod(user string) framework.TestResp {
	clog.Info("=========clearpod========")
	initParam()
	podNameByUser := user + "-" + podName
	urlOfDeletePodWithCM := "%s/api/v1/cube/proxy/clusters/%s/api/v1/namespaces/%s/pods/%s"
	urlOfDeletePodWithCM = fmt.Sprintf(urlOfDeletePodWithCM, framework.KubecubeHost, clusterName, namespace, podNameByUser)
	_, err := httpHelper.Delete(urlOfDeletePodWithCM)
	framework.ExpectNoError(err, "should be deleted")
	return framework.SucceedResp
}

var multiUserTest = framework.MultiUserTest{
	TestName:        "[配置][9387667]ConfigMap检查",
	ContinueIfError: false,
	SkipUsers:       []string{},
	Skipfunc:        nil,
	ErrorFunc:       framework.PermissionErrorFunc,
	AfterEach:       nil,
	BeforeEach:      nil,
	InitStep:        nil,
	FinalStep:       nil,
	Steps: []framework.MultiUserTestStep{
		{
			Name:        "创建CM",
			Description: "1. 创建ConfigMap， 设置多个数据\nkey1=value1\nkey2=value2",
			StepFunc:    createCM,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
		{
			Name:        "创建pod并且查看",
			Description: "2. 创建负载》容器》高级模式，设置环境变量Value类型为ConfigMap\n3. 通过webconsole访问容器查看环境变量env",
			StepFunc:    createPodAndCheck,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
		{
			Name:        "设置CM",
			Description: "4. 设置ConfigMap",
			StepFunc:    updateConfigMap,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
		{
			Name:        "删除CM",
			Description: "5. 删除ConfigMap",
			StepFunc:    deleteConfigMap,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
		{
			Name:        "删除pod",
			Description: "6. 删除Pod",
			StepFunc:    clearPod,
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

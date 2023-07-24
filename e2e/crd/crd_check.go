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

package crd

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
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	client2 "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubecube-io/kubecube-e2e/e2e/framework"
)

var (
	clusterName string
	httpHelper  *framework.HttpHelper
	namespace   string
	cli         client.Client

	checkOfCreateCR *unstructured.Unstructured

	crd      = "crontabs.stable.%s.com"
	crdGroup = "stable.%s.com"
	cr       = "crd-nginx"
)

func initParam() {
	clusterName = framework.TargetClusterName
	httpHelper = framework.NewSingleHttpHelper()
	namespace = framework.NamespaceName
	cli = framework.TargetClusterClient
}

func createCRD(user string) framework.TestResp {
	initParam()
	crdWithUser := fmt.Sprintf(crd, user)
	crdGroupWithUser := fmt.Sprintf(crdGroup, user)
	postJsonOfCRD := `{"apiVersion":"apiextensions.k8s.io/v1","kind":"CustomResourceDefinition","metadata":{"name":"%s"},"spec":{"group":"%s","versions":[{"name":"v1","served":true,"storage":true,"schema":{"openAPIV3Schema":{"type":"object","properties":{"spec":{"type":"object","properties":{"cronSpec":{"type":"string"},"image":{"type":"string"},"replicas":{"type":"integer"}}}}}}}],"scope":"Namespaced","names":{"plural":"crontabs","singular":"crontab","kind":"CronTab","shortNames":["ct"]}}}`
	postJsonOfCRD = fmt.Sprintf(postJsonOfCRD, crdWithUser, crdGroupWithUser)
	urlOfCreateCRD := "%s/api/v1/cube/proxy/clusters/%s/apis/apiextensions.k8s.io/v1/customresourcedefinitions"
	urlOfCreateCRD = fmt.Sprintf(urlOfCreateCRD, framework.KubecubeHost, clusterName)
	respOfCreateCRD, err := httpHelper.RequestByUser(http.MethodPost, urlOfCreateCRD, postJsonOfCRD, user, nil)
	defer respOfCreateCRD.Body.Close()
	body, err := io.ReadAll(respOfCreateCRD.Body)
	framework.ExpectNoError(err)

	if !framework.IsSuccess(respOfCreateCRD.StatusCode) && http.StatusConflict != respOfCreateCRD.StatusCode {
		clog.Warn("res code %d, res data: %s", respOfCreateCRD.StatusCode, string(body))
		return framework.NewTestResp(errors.New("fail to create crd"), respOfCreateCRD.StatusCode)
	}

	checkOfCreateCRD := &v1.CustomResourceDefinition{}
	err = cli.Direct().Get(context.Background(), client2.ObjectKey{
		Namespace: namespace,
		Name:      crdWithUser,
	}, checkOfCreateCRD)
	framework.ExpectNoError(err, "new CRD should be retrieved")
	return framework.SucceedResp
}

func createCR(user string) framework.TestResp {
	initParam()
	crdGroupWithUser := fmt.Sprintf(crdGroup, user)
	crWithUser := framework.NameWithUser(cr, user)
	checkOfCreateCR = &unstructured.Unstructured{}
	checkOfCreateCR.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   crdGroupWithUser,
		Version: "v1",
		Kind:    "CronTab",
	})
	err := cli.Direct().Get(context.Background(), client2.ObjectKey{
		Namespace: namespace,
		Name:      crWithUser,
	}, checkOfCreateCR)
	if err == nil {
		err := cli.Direct().Delete(context.Background(), checkOfCreateCR)
		framework.ExpectNoError(err)
	}
	postJsonOfCreateCR := `{"apiVersion":"%s/v1","kind":"CronTab","metadata":{"labels":{"system/project-project1":"true","system/tenant":"tenant1"},"name":"%s","namespace":"%s"},"spec":{"image":"%s","cronSpec":"*****/5"}}`
	postJsonOfCreateCR = fmt.Sprintf(postJsonOfCreateCR, crdGroupWithUser, crWithUser, namespace, framework.TestImage)
	urlOfCreateCR := "%s/api/v1/cube/proxy/clusters/%s/apis/%s/v1/namespaces/%s/crontabs"
	urlOfCreateCR = fmt.Sprintf(urlOfCreateCR, framework.KubecubeHost, clusterName, crdGroupWithUser, namespace)
	respOfCreateCR, err := httpHelper.RequestByUser(http.MethodPost, urlOfCreateCR, postJsonOfCreateCR, user, nil)
	defer respOfCreateCR.Body.Close()
	_, err = io.ReadAll(respOfCreateCR.Body)
	framework.ExpectNoError(err)
	if !framework.IsSuccess(respOfCreateCR.StatusCode) {
		clog.Warn("res code %d", respOfCreateCR.StatusCode)
		return framework.NewTestResp(errors.New("fail to create cr"), respOfCreateCR.StatusCode)
	}
	err = cli.Direct().Get(context.Background(), client2.ObjectKey{
		Namespace: namespace,
		Name:      crWithUser,
	}, checkOfCreateCR)
	framework.ExpectNoError(err, "new CR should be retrieved")
	framework.ExpectEqual(checkOfCreateCR.Object["spec"].(map[string]interface{})["image"], framework.TestImage, "image should be same")
	return framework.SucceedResp
}

func updateCR(user string) framework.TestResp {
	initParam()
	crdGroupWithUser := fmt.Sprintf(crdGroup, user)
	crWithUser := framework.NameWithUser(cr, user)
	imageNew := framework.NameWithUser(framework.TestImage, user)
	checkOfCreateCR.Object["spec"].(map[string]interface{})["image"] = imageNew

	postJsonOfUpdateCR, _ := json.Marshal(checkOfCreateCR.Object)
	urlOfUpdateCR := "%s/api/v1/cube/proxy/clusters/%s/apis/%s/v1/namespaces/%s/crontabs/%s"
	urlOfUpdateCR = fmt.Sprintf(urlOfUpdateCR, framework.KubecubeHost, clusterName, crdGroupWithUser, namespace, crWithUser)
	respOfUpdateCR, err := httpHelper.RequestByUser(http.MethodPut, urlOfUpdateCR, string(postJsonOfUpdateCR), user, nil)
	defer respOfUpdateCR.Body.Close()
	_, err = io.ReadAll(respOfUpdateCR.Body)
	framework.ExpectNoError(err)

	if !framework.IsSuccess(respOfUpdateCR.StatusCode) {
		clog.Warn("res code %d", respOfUpdateCR.StatusCode)
		return framework.NewTestResp(errors.New("fail to update cr"), respOfUpdateCR.StatusCode)
	}

	checkOfUpdateCR := &unstructured.Unstructured{}
	checkOfUpdateCR.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   crdGroupWithUser,
		Version: "v1",
		Kind:    "CronTab",
	})
	err = cli.Direct().Get(context.Background(), client2.ObjectKey{
		Namespace: namespace,
		Name:      crWithUser,
	}, checkOfUpdateCR)
	framework.ExpectNoError(err, "new CR should be retrieved")
	framework.ExpectEqual(checkOfUpdateCR.Object["spec"].(map[string]interface{})["image"], imageNew, "image should be same")
	return framework.SucceedResp
}

func deleteCR(user string) framework.TestResp {
	initParam()
	crdGroupWithUser := fmt.Sprintf(crdGroup, user)
	crWithUser := framework.NameWithUser(cr, user)
	urlOfDeleteCR := "%s/api/v1/cube/proxy/clusters/%s/apis/%s/v1/namespaces/%s/crontabs/%s"
	urlOfDeleteCR = fmt.Sprintf(urlOfDeleteCR, framework.KubecubeHost, clusterName, crdGroupWithUser, namespace, crWithUser)
	respOfDeleteCR, err := httpHelper.RequestByUser(http.MethodDelete, urlOfDeleteCR, "", user, nil)
	defer respOfDeleteCR.Body.Close()
	_, err = io.ReadAll(respOfDeleteCR.Body)
	framework.ExpectNoError(err)

	if !framework.IsSuccess(respOfDeleteCR.StatusCode) {
		clog.Warn("res code %d", respOfDeleteCR.StatusCode)
		return framework.NewTestResp(errors.New("fail to delete cr"), respOfDeleteCR.StatusCode)
	}

	checkOfDeleteCR := &unstructured.Unstructured{}
	checkOfDeleteCR.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   crdGroupWithUser,
		Version: "v1",
		Kind:    "CronTab",
	})
	err = cli.Direct().Get(context.Background(), client2.ObjectKey{
		Namespace: namespace,
		Name:      crWithUser,
	}, checkOfDeleteCR)
	framework.ExpectEqual(kerrors.IsNotFound(err), true, "CR should be deleted")
	return framework.SucceedResp
}

func deleteCRD(user string) framework.TestResp {
	initParam()
	crdWithUser := fmt.Sprintf(crd, user)
	urlOfDeleteCRD := "%s/api/v1/cube/proxy/clusters/%s/apis/apiextensions.k8s.io/v1/customresourcedefinitions/%s"
	urlOfDeleteCRD = fmt.Sprintf(urlOfDeleteCRD, framework.KubecubeHost, clusterName, crdWithUser)
	respOfDeleteCRD, err := httpHelper.RequestByUser(http.MethodDelete, urlOfDeleteCRD, "", user, nil)
	defer respOfDeleteCRD.Body.Close()
	_, err = io.ReadAll(respOfDeleteCRD.Body)
	framework.ExpectNoError(err)

	if !framework.IsSuccess(respOfDeleteCRD.StatusCode) {
		clog.Warn("res code %d", respOfDeleteCRD.StatusCode)
		return framework.NewTestResp(errors.New("fail to delete crd"), respOfDeleteCRD.StatusCode)
	}

	time.Sleep(time.Second * 10)
	checkOfDeleteCRD := &v1.CustomResourceDefinition{}
	err = cli.Direct().Get(context.Background(), client2.ObjectKey{
		Namespace: namespace,
		Name:      crdWithUser,
	}, checkOfDeleteCRD)
	framework.ExpectEqual(kerrors.IsNotFound(err), true, "CRD should be deleted")
	return framework.SucceedResp
}

var multiUserTest = framework.MultiUserTest{
	TestName:        "[自定义资源CRD][9386694]自定义资源CRD检查",
	ContinueIfError: false,
	Skipfunc:        nil,
	ErrorFunc:       framework.PermissionErrorFunc,
	AfterEach:       nil,
	BeforeEach:      nil,
	InitStep:        nil,
	FinalStep:       nil,
	Steps: []framework.MultiUserTestStep{
		{
			Name:        "创建CRD",
			Description: "1. 创建自定义资源crontabs.stable.example.comp",
			StepFunc:    createCRD,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  false,
				framework.UserProjectAdmin: false,
				framework.UserNormal:       false,
			},
		},
		{
			Name:        "创建 CR",
			Description: "2. 进入空间级别》crontabs.stable.example.com》选择v1版本，创建crontabs.stable.example.com实例\n3. 创建 CR",
			StepFunc:    createCR,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  false,
				framework.UserProjectAdmin: false,
				framework.UserNormal:       false,
			},
		},
		{
			Name:        "设置 CR",
			Description: "4.设置CRD实例",
			StepFunc:    updateCR,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  false,
				framework.UserProjectAdmin: false,
				framework.UserNormal:       false,
			},
		},
		{
			Name:        "删除 CR",
			Description: "5. 删除CRD实例",
			StepFunc:    deleteCR,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  false,
				framework.UserProjectAdmin: false,
				framework.UserNormal:       false,
			},
		},
		{
			Name:        "删除 CRD",
			Description: "6. 删除CRD crontabs.stable.example.com",
			StepFunc:    deleteCRD,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  false,
				framework.UserProjectAdmin: false,
				framework.UserNormal:       false,
			},
		},
	},
}

func init() {
	framework.RegisterByDefault(multiUserTest)
}

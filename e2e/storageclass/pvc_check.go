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

package storageclass

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/kubecube-io/kubecube-e2e/e2e/framework"
	"github.com/kubecube-io/kubecube/pkg/clog"
	"github.com/kubecube-io/kubecube/pkg/multicluster/client"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	clusterName string
	httpHelper  *framework.HttpHelper
	namespace   string
	cli         client.Client

	pvc1Name         = "demo1"
	pvc1NameWithUser string

	pvc2Name         = "demo2"
	pvc2NameWithUser string

	podName         = "task-pv-storage"
	podNameWithUser string

	PV string
)

func initParam() {
	clusterName = framework.TargetClusterName
	httpHelper = framework.NewSingleHttpHelper()
	namespace = framework.NamespaceName
	cli = framework.TargetClusterClient
}

func createPVC1(user string) framework.TestResp {
	initParam()
	pvc1NameWithUser = framework.NameWithUser(pvc1Name, user)
	pvc2NameWithUser = framework.NameWithUser(pvc2Name, user)
	podNameWithUser = framework.NameWithUser(podName, user)
	postJsonOfCreatePVC := `{"apiVersion":"v1","kind":"PersistentVolumeClaim","metadata":{"finalizers":["kubernetes.io/pvc-protection"],"name":"%s","namespace":"%s"},"spec":{"accessModes":["ReadWriteOnce"],"resources":{"requests":{"storage":"10Gi"}},"storageClassName":"%s","volumeMode":"Filesystem"}}`
	postJsonOfCreatePVC = fmt.Sprintf(postJsonOfCreatePVC, pvc1NameWithUser, namespace, framework.StorageClass)
	urlOfCreatePVC := "%s/api/v1/cube/proxy/clusters/%s/api/v1/namespaces/%s/persistentvolumeclaims"
	urlOfCreatePVC = fmt.Sprintf(urlOfCreatePVC, framework.KubecubeHost, clusterName, namespace)
	respOfCreatePVC, err := httpHelper.RequestByUser(http.MethodPost, urlOfCreatePVC, postJsonOfCreatePVC, user, nil)
	defer respOfCreatePVC.Body.Close()
	body, err := io.ReadAll(respOfCreatePVC.Body)
	framework.ExpectNoError(err)
	clog.Info("get pvc1: %+v", string(body))

	if !framework.IsSuccess(respOfCreatePVC.StatusCode) {
		clog.Warn("res code %d", respOfCreatePVC.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to create pvc %s", pvc1NameWithUser), respOfCreatePVC.StatusCode)
	}

	checkOfCreatePVC := &v1.PersistentVolumeClaim{}
	err = cli.Direct().Get(context.Background(), ctrlclient.ObjectKey{
		Namespace: namespace,
		Name:      pvc1NameWithUser,
	}, checkOfCreatePVC)
	framework.ExpectNoError(err, "new pvc should be created")

	return framework.SucceedResp
}

func createPVC2(user string) framework.TestResp {
	postJsonOfCreatePVC := `{"apiVersion":"v1","kind":"PersistentVolumeClaim","metadata":{"finalizers":["kubernetes.io/pvc-protection"],"name":"%s","namespace":"%s"},"spec":{"accessModes":["ReadOnlyMany"],"resources":{"requests":{"storage":"20Gi"}},"storageClassName":"%s","volumeMode":"Filesystem"}}`
	postJsonOfCreatePVC = fmt.Sprintf(postJsonOfCreatePVC, pvc2NameWithUser, namespace, framework.StorageClass)
	urlOfCreatePVC := "%s/api/v1/cube/proxy/clusters/%s/api/v1/namespaces/%s/persistentvolumeclaims"
	urlOfCreatePVC = fmt.Sprintf(urlOfCreatePVC, framework.KubecubeHost, clusterName, namespace)
	respOfCreatePVC, err := httpHelper.RequestByUser(http.MethodPost, urlOfCreatePVC, postJsonOfCreatePVC, user, nil)
	defer respOfCreatePVC.Body.Close()
	body, err := io.ReadAll(respOfCreatePVC.Body)
	framework.ExpectNoError(err)
	clog.Info("get pvc2: %+v", string(body))

	if !framework.IsSuccess(respOfCreatePVC.StatusCode) {
		clog.Warn("res code %d", respOfCreatePVC.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to create pvc %s", pvc2NameWithUser), respOfCreatePVC.StatusCode)
	}

	checkOfCreatePVC := &v1.PersistentVolumeClaim{}
	err = cli.Direct().Get(context.Background(), ctrlclient.ObjectKey{
		Namespace: namespace,
		Name:      pvc2NameWithUser,
	}, checkOfCreatePVC)
	framework.ExpectNoError(err, "new pvc should be created")

	return framework.SucceedResp
}

func createPod(user string) framework.TestResp {
	postJsonOfCreatePod := `{"apiVersion":"v1","kind":"Pod","metadata":{"name":"%s","namespace":"%s"},"spec":{"volumes":[{"name":"task-pv-storage","persistentVolumeClaim":{"claimName":"%s"}}],"imagePullSecrets": [{"name": "%s"}],"affinity":{"nodeAffinity":{"requiredDuringSchedulingIgnoredDuringExecution":{"nodeSelectorTerms":[{"matchExpressions":[{"key":"node.kubecube.io/tenant","operator":"In","values":["share"]}]}]}}},"containers":[{"name":"task-pv-container","image":"%s","command":[],"resources":{"limits":{"cpu":"5000m","memory":"5120Mi"},"requests":{"cpu":"500m","memory":"512Mi"}},"volumeMounts":[{"mountPath":"/root/test","name":"task-pv-storage"}]}]}}`
	postJsonOfCreatePod = fmt.Sprintf(postJsonOfCreatePod, podNameWithUser, namespace, pvc1NameWithUser, framework.ImagePullSecret, framework.TestImage)
	urlOfCreatePod := "%s/api/v1/cube/proxy/clusters/%s/api/v1/namespaces/%s/pods"
	urlOfCreatePod = fmt.Sprintf(urlOfCreatePod, framework.KubecubeHost, clusterName, namespace)
	respOfCreatePod, err := httpHelper.RequestByUser(http.MethodPost, urlOfCreatePod, postJsonOfCreatePod, user, nil)
	defer respOfCreatePod.Body.Close()
	body, err := io.ReadAll(respOfCreatePod.Body)
	framework.ExpectNoError(err)
	clog.Info("get pod task-pv-pod: %+v", string(body))

	if !framework.IsSuccess(respOfCreatePod.StatusCode) {
		clog.Warn("res code %d", respOfCreatePod.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to create pod %s", podNameWithUser), respOfCreatePod.StatusCode)
	}

	err = wait.Poll(framework.WaitInterval, framework.WaitTimeout, func() (done bool, err error) {
		checkOfCreatePVC := &v1.PersistentVolumeClaim{}
		err = cli.Direct().Get(context.Background(), ctrlclient.ObjectKey{
			Namespace: namespace,
			Name:      pvc1NameWithUser,
		}, checkOfCreatePVC)
		if err != nil {
			return false, err
		}
		if string(checkOfCreatePVC.Status.Phase) != "Bound" {
			return false, nil
		}
		return true, nil
	})
	framework.ExpectNoError(err, "pvc should be created")

	err = wait.Poll(framework.WaitInterval, framework.WaitTimeout, func() (done bool, err error) {
		checkOfCreatePod := &v1.Pod{}
		err = cli.Direct().Get(context.Background(), ctrlclient.ObjectKey{
			Namespace: namespace,
			Name:      podNameWithUser,
		}, checkOfCreatePod)
		if err != nil {
			return false, err
		}
		return true, nil

	})
	framework.ExpectNoError(err, "pod should be created")

	checkOfPVList := &v1.PersistentVolumeList{}
	err = cli.Direct().List(context.Background(), checkOfPVList, &ctrlclient.ListOptions{Namespace: namespace})
	for _, item := range checkOfPVList.Items {
		if item.Spec.ClaimRef.Name == pvc1NameWithUser {
			PV = checkOfPVList.Items[0].Name
		}
	}
	clog.Info("Pv Got %s", PV)

	return framework.SucceedResp
}

func deletePod(user string) framework.TestResp {
	urlOfDeletePod := "%s/api/v1/cube/proxy/clusters/%s/api/v1/namespaces/%s/pods/%s"
	urlOfDeletePod = fmt.Sprintf(urlOfDeletePod, framework.KubecubeHost, clusterName, namespace, podNameWithUser)
	respOfDeletePod, err := httpHelper.RequestByUser(http.MethodDelete, urlOfDeletePod, "", user, nil)
	defer respOfDeletePod.Body.Close()
	body, err := io.ReadAll(respOfDeletePod.Body)
	framework.ExpectNoError(err)
	clog.Info("delete pod: %+v", string(body))

	if !framework.IsSuccess(respOfDeletePod.StatusCode) {
		clog.Warn("res code %d", respOfDeletePod.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to delete pod %s", podNameWithUser), respOfDeletePod.StatusCode)
	}

	err = wait.Poll(framework.WaitInterval, framework.WaitTimeout, func() (done bool, err error) {
		checkOfDeletePod := &v1.Pod{}
		err = cli.Direct().Get(context.Background(), ctrlclient.ObjectKey{
			Namespace: namespace,
			Name:      podNameWithUser,
		}, checkOfDeletePod)
		if !errors.IsNotFound(err) {
			return false, err
		}
		return true, nil
	})
	framework.ExpectNoError(err, "pod should be delete")

	return framework.SucceedResp
}

func deletePvc(user string) framework.TestResp {
	urlOfDeletePVC := "%s/api/v1/cube/proxy/clusters/%s/api/v1/namespaces/%s/persistentvolumeclaims/%s"
	urlOfDeletePVC = fmt.Sprintf(urlOfDeletePVC, framework.KubecubeHost, clusterName, namespace, pvc1NameWithUser)
	respOfDeletePVC, err := httpHelper.RequestByUser(http.MethodDelete, urlOfDeletePVC, "", user, nil)
	defer respOfDeletePVC.Body.Close()
	body, err := io.ReadAll(respOfDeletePVC.Body)
	clog.Info("delete pvc1: %+v", string(body))

	if !framework.IsSuccess(respOfDeletePVC.StatusCode) {
		clog.Warn("res code %d", respOfDeletePVC.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to delete pvc %s", pvc1NameWithUser), respOfDeletePVC.StatusCode)
	}

	urlOfDeletePVC = "%s/api/v1/cube/proxy/clusters/%s/api/v1/namespaces/%s/persistentvolumeclaims/%s"
	urlOfDeletePVC = fmt.Sprintf(urlOfDeletePVC, framework.KubecubeHost, clusterName, namespace, pvc2NameWithUser)
	respOfDeletePVC, err = httpHelper.RequestByUser(http.MethodDelete, urlOfDeletePVC, "", user, nil)
	defer respOfDeletePVC.Body.Close()
	body, err = io.ReadAll(respOfDeletePVC.Body)
	framework.ExpectNoError(err)
	clog.Info("delete pvc2: %+v", string(body))

	if !framework.IsSuccess(respOfDeletePVC.StatusCode) {
		clog.Warn("res code %d", respOfDeletePVC.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to delete pvc %s", pvc2NameWithUser), respOfDeletePVC.StatusCode)
	}

	err = wait.Poll(framework.WaitInterval, framework.WaitTimeout, func() (done bool, err error) {
		checkOfDeletePVC := &v1.PersistentVolumeClaim{}
		err = cli.Direct().Get(context.Background(), ctrlclient.ObjectKey{
			Namespace: namespace,
			Name:      pvc1NameWithUser,
		}, checkOfDeletePVC)
		if !errors.IsNotFound(err) {
			return false, err
		}
		return true, nil
	})
	framework.ExpectNoError(err, "pvc1 should be deleted")

	err = wait.Poll(framework.WaitInterval, framework.WaitTimeout, func() (done bool, err error) {
		checkOfDeletePVC := &v1.PersistentVolumeClaim{}
		err = cli.Direct().Get(context.Background(), ctrlclient.ObjectKey{
			Namespace: namespace,
			Name:      pvc2NameWithUser,
		}, checkOfDeletePVC)
		if !errors.IsNotFound(err) {
			return false, err
		}
		return true, nil
	})
	framework.ExpectNoError(err, "pvc2 should be deleted")

	return framework.SucceedResp
}

func deletePv(user string) framework.TestResp {
	urlOfDeletePV := "%s/api/v1/cube/proxy/clusters/%s/api/v1/namespaces/%s/persistentvolume/%s"
	urlOfDeletePV = fmt.Sprintf(urlOfDeletePV, framework.KubecubeHost, clusterName, namespace, PV)
	clog.Info("delete pv %s", PV)
	respOfDeletePV, err := httpHelper.RequestByUser(http.MethodDelete, urlOfDeletePV, "", user, nil)
	defer respOfDeletePV.Body.Close()
	body, err := io.ReadAll(respOfDeletePV.Body)
	clog.Info("delete pv %s", string(body))
	framework.ExpectNoError(err)

	return framework.SucceedResp
}

var multiUserTest = framework.MultiUserTest{
	TestName:        "[存储][9387658]存储声明创建检查",
	ContinueIfError: false,
	Skipfunc: func() bool {
		return !framework.PVEnabled
	},
	ErrorFunc:  framework.PermissionErrorFunc,
	AfterEach:  nil,
	BeforeEach: nil,
	InitStep:   nil,
	FinalStep:  nil,
	Steps: []framework.MultiUserTestStep{
		{
			Name:        "create PVC1",
			Description: "1. 进入容器云》存储声明\n2. 创建存储声明pvc1，容量10Gi、创建方式动态持久化存储、独占读写模式",
			StepFunc:    createPVC1,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
		{
			Name:        "create PVC2",
			Description: "3. 创建存储声明pvc2，容量20Gi、创建方式动态持久化存储、只读共享",
			StepFunc:    createPVC2,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
		{
			Name:     "create pod",
			StepFunc: createPod,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
		{
			Name:     "delete pod",
			StepFunc: deletePod,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
		{
			Name:        "delete pvc",
			Description: "4. 删除pvc",
			StepFunc:    deletePvc,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
		{
			Name:        "delete pv",
			Description: "5. 删除pv",
			StepFunc:    deletePv,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
	},
}

func init() {
	framework.RegisterByDefault(multiUserTest)
}

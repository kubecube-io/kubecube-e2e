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

package workloads

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kubecube-io/kubecube/pkg/clog"
	"github.com/onsi/ginkgo"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubecube-io/kubecube-e2e/e2e/framework"
)

func createDs(user string) framework.TestResp {
	initParam()
	daemonSetNameWithUser = framework.NameWithUser(daemonSetName, user)
	dsJson := `{"apiVersion":"apps/v1","kind":"DaemonSet","metadata":{"name":"%s","annotations":{},"labels":{"kubecube.io/app":"%s","system/tenant":"netease.share"}},"spec":{"selector":{"matchLabels":{"kubecube.io/app":"%s"}},"template":{"metadata":{"annotations":{},"labels":{"kubecube.io/app":"%s"}},"spec":{"containers":[{"name":"%s","args":[],"command":[],"env":[],"image":"%s","imagePullPolicy":"IfNotPresent","lifecycle":{"postStart":null,"preStop":null},"livenessProbe":null,"readinessProbe":null,"ports":null,"resources":{"limits":{"cpu":"100m","memory":"128Mi"},"requests":{"cpu":"100m","memory":"128Mi"}},"volumeMounts":[]}],"initContainers":[],"imagePullSecrets":[{"name":"%s"}],"volumes":[],"affinity":{},"restartPolicy":"Always"}}}}`
	dsJson = fmt.Sprintf(dsJson, daemonSetNameWithUser, daemonSetNameWithUser, daemonSetNameWithUser, daemonSetNameWithUser, daemonSetNameWithUser, framework.TestImage, framework.ImagePullSecret)
	url := BuildK8sProxyUrl(framework.KubecubeHost, framework.TargetClusterName, "apis/apps/v1", framework.NamespaceName, "daemonsets", "")
	resp, err := httpHelper.RequestByUser(http.MethodPost, url, dsJson, user, nil)
	framework.ExpectNoError(err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	framework.ExpectNoError(err)
	clog.Debug("create daemonSet %v, %v", daemonSetNameWithUser, string(body))

	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to create ds %s", daemonSetNameWithUser), resp.StatusCode)
	}

	time.Sleep(time.Second * 20)
	return framework.SucceedResp
}

func checkDs(user string) framework.TestResp {
	ds := v1.DaemonSet{}
	err := wait.Poll(framework.WaitInterval, framework.WaitTimeout,
		func() (bool, error) {
			err := targetClient.Cache().Get(context.TODO(), types.NamespacedName{
				Name:      daemonSetNameWithUser,
				Namespace: framework.NamespaceName,
			}, &ds)
			if err != nil {
				return false, nil
			} else {
				return true, nil
			}
		})
	framework.ExpectNoError(err)
	clog.Debug("daemonset status: %v", ds.Status)
	return framework.SucceedResp
}

func checkDsList(user string) framework.TestResp {
	dsList := v1.DaemonSetList{}
	err := targetClient.Cache().List(context.TODO(), &dsList, &client.ListOptions{
		Namespace:     framework.NamespaceName,
		LabelSelector: labels.Set{"kubecube.io/app": daemonSetNameWithUser}.AsSelector(),
	})
	framework.ExpectNoError(err)
	clog.Debug("ds list: %v", dsList.Items)
	framework.ExpectEqual(len(dsList.Items), 1)
	return framework.SucceedResp
}

func checkDsStatus(user string) framework.TestResp {
	podList := corev1.PodList{}
	err := wait.Poll(framework.WaitInterval, framework.WaitTimeout,
		func() (bool, error) {
			err := targetClient.Cache().List(context.TODO(), &podList, &client.ListOptions{
				Namespace:     framework.NamespaceName,
				LabelSelector: labels.Set{"kubecube.io/app": daemonSetNameWithUser}.AsSelector(),
			})
			if err != nil {
				return false, nil
			} else {
				return true, nil
			}
		})
	framework.ExpectNoError(err)
	framework.ExpectNotEqual(len(podList.Items), 0)
	ginkgo.By("查看容器详情")
	pod := podList.Items[0]
	framework.ExpectEqual(len(pod.Spec.Containers), 1)
	container := pod.Spec.Containers[0]
	framework.ExpectEqual(container.Name, daemonSetNameWithUser)
	framework.ExpectEqual(container.Image, framework.TestImage)
	ginkgo.By("查看容器日志")
	url := BuildLogUrl(framework.KubecubeHost, framework.TargetClusterName, framework.NamespaceName, pod.Name, container.Name)
	logResp, err := httpHelper.RequestByUser(http.MethodGet, url, "", user, nil)
	framework.ExpectNoError(err)
	if !framework.IsSuccess(logResp.StatusCode) {
		clog.Warn("res code %d", logResp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to get pod %s log", pod.Name), logResp.StatusCode)
	}

	framework.ExpectEqual(logResp.StatusCode, 200)
	return framework.SucceedResp
}

func checkDsEvent(user string) framework.TestResp {
	ds := v1.DaemonSet{}
	err := targetClient.Cache().Get(context.TODO(), types.NamespacedName{
		Name:      daemonSetNameWithUser,
		Namespace: framework.NamespaceName,
	}, &ds)
	framework.ExpectNoError(err)
	url := BuildEventUrl(framework.KubecubeHost, framework.TargetClusterName, framework.NamespaceName, string(ds.UID))
	resp, err := httpHelper.RequestByUser(http.MethodGet, url, "", user, nil)
	framework.ExpectNoError(err)
	body, err := io.ReadAll(resp.Body)
	framework.ExpectNoError(err)

	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to get ds %s event", daemonSetNameWithUser), resp.StatusCode)
	}

	eventList := corev1.EventList{}
	err = json.Unmarshal(body, &eventList)
	framework.ExpectNoError(err)
	framework.ExpectNotEqual(len(eventList.Items), 0)
	return framework.SucceedResp
}

func checkDsPodEvent(user string) framework.TestResp {
	podList := corev1.PodList{}
	err := targetClient.Cache().List(context.TODO(), &podList, &client.ListOptions{
		Namespace:     framework.NamespaceName,
		LabelSelector: labels.Set{"kubecube.io/app": daemonSetNameWithUser}.AsSelector(),
	})
	framework.ExpectNoError(err)
	framework.ExpectNotEqual(len(podList.Items), 0)
	pod := podList.Items[0]
	url := BuildEventUrl(framework.KubecubeHost, framework.TargetClusterName, framework.NamespaceName, string(pod.UID))
	resp, err := httpHelper.RequestByUser(http.MethodGet, url, "", user, nil)
	framework.ExpectNoError(err)
	body, err := io.ReadAll(resp.Body)
	framework.ExpectNoError(err)

	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to get ds pod %s event", pod.Name), resp.StatusCode)
	}

	eventList := corev1.EventList{}
	err = json.Unmarshal(body, &eventList)
	framework.ExpectNotEqual(len(eventList.Items), 0)
	return framework.SucceedResp
}

func checkDsPerformance(user string) framework.TestResp {
	// TODO
	return framework.SucceedResp
}

func checkDsCondition(user string) framework.TestResp {
	podList := corev1.PodList{}
	err := targetClient.Cache().List(context.TODO(), &podList, &client.ListOptions{
		Namespace:     framework.NamespaceName,
		LabelSelector: labels.Set{"kubecube.io/app": daemonSetNameWithUser}.AsSelector(),
	})
	framework.ExpectNoError(err)
	framework.ExpectNotEqual(len(podList.Items), 0)
	pod := podList.Items[0]
	framework.ExpectNotEqual(len(pod.Status.Conditions), 0)
	return framework.SucceedResp
}

func checkDsUpdate(user string) framework.TestResp {
	updateJson := `{"apiVersion":"apps/v1","kind":"DaemonSet","metadata":{"labels":{"kubecube.io/app":"%s","system/tenant":"netease.share"},"name":"%s","namespace":"%s"},"spec":{"minReadySeconds":30,"revisionHistoryLimit":10,"selector":{"matchLabels":{"kubecube.io/app":"%s"}},"template":{"metadata":{"annotations":{"annotation1":"annotation1"},"creationTimestamp":null,"labels":{"kubecube.io/app":"%s","label1":"label1"}},"spec":{"affinity":{},"containers":[{"name":"%s","args":[],"command":[],"env":[],"image":"%s","imagePullPolicy":"IfNotPresent","lifecycle":{"postStart":null,"preStop":null},"livenessProbe":null,"readinessProbe":null,"ports":null,"resources":{"limits":{"cpu":"500m","memory":"512Mi"},"requests":{"cpu":"500m","memory":"512Mi"}},"volumeMounts":[]}],"dnsPolicy":"ClusterFirst","restartPolicy":"Always","schedulerName":"default-scheduler","securityContext":{},"terminationGracePeriodSeconds":30,"initContainers":[],"imagePullSecrets":[{"name":"%s"}],"tolerations":[{"key":"example-key","operator":"Equal","value":"example-value","effect":"NoExecute","tolerationSeconds":30}],"volumes":[]}},"updateStrategy":{"rollingUpdate":{"maxUnavailable":10},"type":"RollingUpdate"}}}`
	updateJson = fmt.Sprintf(updateJson, daemonSetNameWithUser, daemonSetNameWithUser, framework.NamespaceName, daemonSetNameWithUser, daemonSetNameWithUser, daemonSetNameWithUser, framework.TestImage, framework.ImagePullSecret)
	url := BuildK8sProxyUrl(framework.KubecubeHost, framework.TargetClusterName, "apis/apps/v1", framework.NamespaceName, "daemonsets", daemonSetNameWithUser)
	updateResp, err := httpHelper.RequestByUser(http.MethodPut, url, updateJson, user, nil)
	framework.ExpectNoError(err)
	_, err = io.ReadAll(updateResp.Body)
	framework.ExpectNoError(err)

	if !framework.IsSuccess(updateResp.StatusCode) {
		clog.Warn("res code %d", updateResp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to update ds %s", daemonSetNameWithUser), updateResp.StatusCode)
	}

	time.Sleep(time.Minute)
	ds := v1.DaemonSet{}
	err = targetClient.Cache().Get(context.TODO(), types.NamespacedName{
		Name:      daemonSetNameWithUser,
		Namespace: framework.NamespaceName,
	}, &ds)
	framework.ExpectNoError(err)
	podList := corev1.PodList{}
	err = targetClient.Cache().List(context.TODO(), &podList, &client.ListOptions{
		Namespace:     framework.NamespaceName,
		LabelSelector: labels.Set{"kubecube.io/app": daemonSetNameWithUser}.AsSelector(),
	})
	framework.ExpectNoError(err)
	framework.ExpectNotEqual(len(podList.Items), 0)
	pod := podList.Items[0]

	ginkgo.By("修改部署策略")
	tolerationCheck := false
	for _, toleration := range pod.Spec.Tolerations {
		if toleration.Key == "example-key" {
			framework.ExpectEqual(toleration.Effect, corev1.TaintEffectNoExecute)
			framework.ExpectEqual(toleration.Operator, corev1.TolerationOpEqual)
			framework.ExpectEqual(toleration.Value, "example-value")
			tolerationCheck = true
			break
		}
	}
	framework.ExpectEqual(tolerationCheck, true)

	ginkgo.By("修改更新策略")
	var i int32
	i = 10
	framework.ExpectEqual(ds.Spec.UpdateStrategy.RollingUpdate.MaxUnavailable.IntVal, i)

	ginkgo.By("修改配置")
	framework.ExpectEqual(len(pod.Spec.Containers), 1)
	framework.ExpectEqual(pod.Spec.Containers[0].Resources.Requests.Cpu().String(), "500m")
	framework.ExpectEqual(pod.Spec.Containers[0].Resources.Requests.Memory().String(), "512Mi")

	ginkgo.By("修改标签")
	labelCheck := false
	if val, ok := pod.Labels["label1"]; ok {
		if val == "label1" {
			labelCheck = true
		}
	}
	framework.ExpectEqual(labelCheck, true)

	ginkgo.By("修改注释")
	annotationCheck := false
	if val, ok := pod.Annotations["annotation1"]; ok {
		if val == "annotation1" {
			annotationCheck = true
		}
	}
	framework.ExpectEqual(annotationCheck, true)
	return framework.SucceedResp
}

func deleteDs(user string) framework.TestResp {
	url := BuildK8sProxyUrl(framework.KubecubeHost, framework.TargetClusterName, "apis/apps/v1", framework.NamespaceName, "daemonsets", daemonSetNameWithUser)
	resp, err := httpHelper.RequestByUser(http.MethodDelete, url, "", user, nil)
	defer resp.Body.Close()

	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to delete ds %s", daemonSetNameWithUser), resp.StatusCode)
	}

	framework.ExpectNoError(err)
	time.Sleep(time.Minute)
	return framework.SucceedResp
}

var multiUserDsTest = framework.MultiUserTest{
	TestName:        "[工作负载][9478780]创建DaemonSet",
	ContinueIfError: false,
	Skipfunc: func() bool {
		return !framework.DaemonSetEnable
	},
	ErrorFunc:  framework.PermissionErrorFunc,
	AfterEach:  nil,
	BeforeEach: nil,
	InitStep:   nil,
	FinalStep:  nil,
	Steps: []framework.MultiUserTestStep{
		{
			Name: "创建DaemonSet",
			Description: "1、进入工作负载》Daemonsets菜单，点击部署" +
				"2、填写正确的负载名称、容器名称、镜像名称，点击立即创建",
			StepFunc: createDs,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: false,
				framework.UserNormal:       false,
			},
		},
		{
			Name:        "创建DaemonSet成功",
			Description: "创建成功",
			StepFunc:    checkDs,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: false,
				framework.UserNormal:       false,
			},
		},
		{
			Name:     "列表中展示创建的DaemonSet",
			StepFunc: checkDsList,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
		{
			Name:        "都可以查看到准确的对应信息",
			Description: "查看副本基本信息",
			StepFunc:    checkDsStatus,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
		{
			Name:        "DaemonSet事件前端页面和后台k8s命令显示一致",
			Description: "查看DaemonSet事件",
			StepFunc:    checkDsEvent,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
		{
			Name:        "副本事件前端页面和后台k8s命令显示一致",
			Description: "查看副本事件",
			StepFunc:    checkDsPodEvent,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
		{
			Name:        "正确返回负载各项性能指标",
			Description: "查看副本的性能指标",
			StepFunc:    checkDsPerformance,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
		{
			Name:        "DaemonSet和副本的conditions与k8s查询一致",
			Description: "查看副本的condition详情与k8s的是否一致",
			StepFunc:    checkDsCondition,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
		{
			Name:        "修改DaemonSet",
			Description: "以上修改均能成功",
			StepFunc:    checkDsUpdate,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: false,
				framework.UserNormal:       false,
			},
		},
		{
			Name:        "clean DaemonSet",
			Description: "clean DaemonSet",
			StepFunc:    deleteDs,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: false,
				framework.UserNormal:       false,
			},
		},
	},
}

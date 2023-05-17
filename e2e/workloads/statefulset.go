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

func createStatefulsetWithPvc(user string) framework.TestResp {
	ginkgo.By("1.创建Statefulset，名称sts1 》副本数为2》" +
		"2.开启存储声明pvc1，存储选择StorageClass1》" +
		"3.镜像填写tomcat或选择tomcat》容器配置，基础配置选择高性能 》" +
		"4.点击 高级模式，容器挂载存储模板pvc1" +
		"5.启用容器运行探针 》脚本方式设置执行脚本为命令行，如：echo test 》" +
		"6.打开部署策略 》节点亲和性Key填写“kubernetes.io/hostname”，values填写某个节点ipxx.xx 》提交设置")
	stsJson := `{"apiVersion":"apps/v1","kind":"StatefulSet","metadata":{"name":"%s","annotations":{},"labels":{"kubecube.io/app":"%s"}},"spec":{"selector":{"matchLabels":{"kubecube.io/app":"%s"}},"template":{"metadata":{"annotations":{},"labels":{"kubecube.io/app":"%s"}},"spec":{"containers":[{"name":"%s","args":[],"command":[],"env":[],"image":"%s","imagePullPolicy":"IfNotPresent","lifecycle":{"postStart":null,"preStop":null},"livenessProbe":{"exec":{"command":["/bin/echo","test"]},"failureThreshold":1,"initialDelaySeconds":0,"periodSeconds":10,"successThreshold":1,"timeoutSeconds":1},"readinessProbe":null,"ports":null,"resources":{"limits":{"cpu":"100m","memory":"128Mi"},"requests":{"cpu":"100m","memory":"128Mi"}},"volumeMounts":[{"name":"pv1","mountPath":"/mnt1"}]}],"initContainers":[],"imagePullSecrets":[{"name":"%s"}],"volumes":[],"affinity":{"nodeAffinity":{"requiredDuringSchedulingIgnoredDuringExecution":{"nodeSelectorTerms":[{"matchExpressions":[{"key":"kubernetes.io/hostname","operator":"In","values":["%s"]}]}]}}},"restartPolicy":"Always","tolerations":[]}},"replicas":2,"serviceName":"sts-svc","volumeClaimTemplates":[{"metadata":{"name":"pv1"},"spec":{"storageClassName":"%s","resources":{"requests":{"storage":"100Mi"}},"accessModes":["ReadWriteOnce"]}}]}}`
	stsJson = fmt.Sprintf(stsJson, stsNameWithUser, stsNameWithUser, stsNameWithUser, stsNameWithUser, stsNameWithUser, framework.TestImage, framework.ImagePullSecret, framework.NodeHostName, framework.StorageClass)
	url := BuildK8sProxyUrl(framework.KubecubeHost, framework.TargetClusterName, "apis/apps/v1", framework.NamespaceName, "statefulsets", "")
	resp, err := httpHelper.RequestByUser(http.MethodPost, url, stsJson, user, nil)
	framework.ExpectNoError(err)
	body, err := io.ReadAll(resp.Body)
	framework.ExpectNoError(err)
	clog.Debug("create statefulsets %v, %v", stsNameWithUser, string(body))
	defer resp.Body.Close()
	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to create statefulset %s", stsNameWithUser), resp.StatusCode)
	}

	time.Sleep(time.Second * 20)
	return framework.SucceedResp
}

func createStatefulsetWithoutPvc(user string) framework.TestResp {
	ginkgo.By("1.创建Statefulset，名称sts1 》副本数为2》" +
		"2.开启存储声明pvc1，存储选择StorageClass1》" +
		"3.镜像填写tomcat或选择tomcat》容器配置，基础配置选择高性能 》" +
		"4.点击 高级模式，容器挂载存储模板pvc1" +
		"5.启用容器运行探针 》脚本方式设置执行脚本为命令行，如：echo test 》" +
		"6.打开部署策略 》节点亲和性Key填写“kubernetes.io/hostname”，values填写某个节点ipxx.xx 》提交设置")
	stsJson := `{"apiVersion":"apps/v1","kind":"StatefulSet","metadata":{"name":"%s","annotations":{},"labels":{"kubecube.io/app":"%s"}},"spec":{"selector":{"matchLabels":{"kubecube.io/app":"%s"}},"template":{"metadata":{"annotations":{},"labels":{"kubecube.io/app":"%s"}},"spec":{"containers":[{"name":"%s","args":[],"command":[],"env":[],"image":"%s","imagePullPolicy":"IfNotPresent","lifecycle":{"postStart":null,"preStop":null},"livenessProbe":{"exec":{"command":["/bin/echo","test"]},"failureThreshold":1,"initialDelaySeconds":0,"periodSeconds":10,"successThreshold":1,"timeoutSeconds":1},"readinessProbe":null,"ports":null,"resources":{"limits":{"cpu":"100m","memory":"128Mi"},"requests":{"cpu":"100m","memory":"128Mi"}}}],"initContainers":[],"imagePullSecrets":[{"name":"%s"}],"volumes":[],"affinity":{"nodeAffinity":{"requiredDuringSchedulingIgnoredDuringExecution":{"nodeSelectorTerms":[{"matchExpressions":[{"key":"kubernetes.io/hostname","operator":"In","values":["%s"]}]}]}}},"restartPolicy":"Always","tolerations":[]}},"replicas":2,"serviceName":"sts-svc"}}`
	stsJson = fmt.Sprintf(stsJson, stsNameWithUser, stsNameWithUser, stsNameWithUser, stsNameWithUser, stsNameWithUser, framework.TestImage, framework.ImagePullSecret, framework.NodeHostName)
	url := BuildK8sProxyUrl(framework.KubecubeHost, framework.TargetClusterName, "apis/apps/v1", framework.NamespaceName, "statefulsets", "")
	resp, err := httpHelper.RequestByUser(http.MethodPost, url, stsJson, user, nil)
	framework.ExpectNoError(err)
	body, err := io.ReadAll(resp.Body)
	framework.ExpectNoError(err)
	clog.Debug("create statefulsets %v, %v", stsNameWithUser, string(body))
	defer resp.Body.Close()
	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to create statefulset %s", stsNameWithUser), resp.StatusCode)
	}
	time.Sleep(time.Second * 20)
	return framework.SucceedResp
}

func createStatefulset(user string) framework.TestResp {
	initParam()
	stsNameWithUser = framework.NameWithUser(stsName, user)
	if framework.PVEnabled {
		clog.Debug("createStatefulsetWithPv")
		return createStatefulsetWithPvc(user)
	} else {
		clog.Debug("createStatefulsetWithoutPv")
		return createStatefulsetWithoutPvc(user)
	}
}

func checkStatefulsetVolume(user string) framework.TestResp {
	if !framework.PVEnabled {
		return framework.SucceedResp
	}

	sts := v1.StatefulSet{}
	err := targetClient.Cache().Get(context.TODO(), types.NamespacedName{
		Name:      stsNameWithUser,
		Namespace: framework.NamespaceName,
	}, &sts)
	framework.ExpectNoError(err)
	framework.ExpectEqual(len(sts.Spec.Template.Spec.Containers), 1)
	container := sts.Spec.Template.Spec.Containers[0]
	volumeCheck := false
	for _, volumeMount := range container.VolumeMounts {
		if volumeMount.Name == "pv1" {
			framework.ExpectEqual(volumeMount.MountPath, "/mnt1")
			volumeCheck = true
			break
		}
	}
	framework.ExpectEqual(volumeCheck, true)
	return framework.SucceedResp
}

func checkStatefulsetRunning(user string) framework.TestResp {
	podList := corev1.PodList{}
	err := targetClient.Cache().List(context.TODO(), &podList, &client.ListOptions{
		Namespace:     framework.NamespaceName,
		LabelSelector: labels.Set{"kubecube.io/app": stsNameWithUser}.AsSelector(),
	})
	framework.ExpectNoError(err)
	framework.ExpectEqual(len(podList.Items), 2)
	pod1Check := false
	pod2Check := false
	for _, pod := range podList.Items {
		if pod.Name == stsNameWithUser+"-0" {
			framework.ExpectEqual(pod.Status.Phase, corev1.PodRunning)
			framework.ExpectEqual(pod.Status.HostIP, framework.NodeHostIp)
			pod1Check = true
		}
		if pod.Name == stsNameWithUser+"-1" {
			framework.ExpectEqual(pod.Status.Phase, corev1.PodRunning)
			framework.ExpectEqual(pod.Status.HostIP, framework.NodeHostIp)
			pod2Check = true
		}
	}
	framework.ExpectEqual(pod1Check, true)
	framework.ExpectEqual(pod2Check, true)
	return framework.SucceedResp
}

func checkStatefulsetDetail(user string) framework.TestResp {
	podList := corev1.PodList{}
	err := targetClient.Cache().List(context.TODO(), &podList, &client.ListOptions{
		Namespace:     framework.NamespaceName,
		LabelSelector: labels.Set{"kubecube.io/app": stsNameWithUser}.AsSelector(),
	})
	framework.ExpectNoError(err)
	framework.ExpectEqual(len(podList.Items), 2)
	pod := podList.Items[0]
	framework.ExpectEqual(len(pod.Spec.Containers), 1)
	container := pod.Spec.Containers[0]
	framework.ExpectEqual(container.Name, stsNameWithUser)
	framework.ExpectEqual(container.Image, framework.TestImage)
	ginkgo.By("查看容器日志")
	url := BuildLogUrl(framework.KubecubeHost, framework.TargetClusterName, framework.NamespaceName, pod.Name, container.Name)
	logResp, err := httpHelper.RequestByUser(http.MethodGet, url, "", user, nil)
	framework.ExpectNoError(err)

	defer logResp.Body.Close()
	if !framework.IsSuccess(logResp.StatusCode) {
		clog.Warn("res code %d", logResp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to get pod %s log", pod.Name), logResp.StatusCode)
	}

	framework.ExpectEqual(logResp.StatusCode, 200)
	return framework.SucceedResp
}

func checkStatefulsetInfo(user string) framework.TestResp {
	// TODO
	return framework.SucceedResp
}

func checkStatefulsetPerformance(user string) framework.TestResp {
	// TODO
	return framework.SucceedResp
}

func checkStatefulsetCondition(user string) framework.TestResp {
	podList := corev1.PodList{}
	err := targetClient.Cache().List(context.TODO(), &podList, &client.ListOptions{
		Namespace:     framework.NamespaceName,
		LabelSelector: labels.Set{"kubecube.io/app": stsNameWithUser}.AsSelector(),
	})
	framework.ExpectNoError(err)
	framework.ExpectEqual(len(podList.Items), 2)
	pod := podList.Items[0]
	framework.ExpectNotEqual(len(pod.Status.Conditions), 0)
	return framework.SucceedResp
}

func checkStatefulsetEvents(user string) framework.TestResp {
	sts := v1.StatefulSet{}
	err := targetClient.Cache().Get(context.TODO(), types.NamespacedName{
		Name:      stsNameWithUser,
		Namespace: framework.NamespaceName,
	}, &sts)
	framework.ExpectNoError(err)
	url := BuildEventUrl(framework.KubecubeHost, framework.TargetClusterName, framework.NamespaceName, string(sts.UID))
	resp, err := httpHelper.RequestByUser(http.MethodGet, url, "", user, nil)
	framework.ExpectNoError(err)
	body, err := io.ReadAll(resp.Body)
	framework.ExpectNoError(err)

	defer resp.Body.Close()
	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to get sts %s events", stsNameWithUser), resp.StatusCode)
	}

	eventList := corev1.EventList{}
	err = json.Unmarshal(body, &eventList)
	framework.ExpectNoError(err)
	framework.ExpectNotEqual(len(eventList.Items), 0)
	return framework.SucceedResp
}

func checkStatefulsetPodEvents(user string) framework.TestResp {
	podList := corev1.PodList{}
	err := targetClient.Cache().List(context.TODO(), &podList, &client.ListOptions{
		Namespace:     framework.NamespaceName,
		LabelSelector: labels.Set{"kubecube.io/app": stsNameWithUser}.AsSelector(),
	})
	framework.ExpectNoError(err)
	framework.ExpectEqual(len(podList.Items), 2)
	pod := podList.Items[0]
	url := BuildEventUrl(framework.KubecubeHost, framework.TargetClusterName, framework.NamespaceName, string(pod.UID))
	resp, err := httpHelper.Get(url, nil)
	framework.ExpectNoError(err)
	body, err := io.ReadAll(resp.Body)
	framework.ExpectNoError(err)

	defer resp.Body.Close()
	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to get pod %s events", pod.Name), resp.StatusCode)
	}

	eventList := corev1.EventList{}
	err = json.Unmarshal(body, &eventList)
	framework.ExpectNotEqual(len(eventList.Items), 0)
	return framework.SucceedResp
}

func restStatefulsetReplica(user string) framework.TestResp {
	patchJson := "{\"spec\":{\"replicas\":1}}"
	header := map[string]string{
		"Content-Type": "application/strategic-merge-patch+json",
	}
	deployUrl := BuildK8sProxyUrl(framework.KubecubeHost, framework.TargetClusterName, "apis/apps/v1", framework.NamespaceName, "statefulsets", stsNameWithUser)
	resp, err := httpHelper.RequestByUser(http.MethodPatch, deployUrl, patchJson, user, header)
	framework.ExpectNoError(err)

	defer resp.Body.Close()
	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to reset sts %s replica", stsNameWithUser), resp.StatusCode)
	}

	sts := v1.StatefulSet{}
	err = wait.Poll(framework.WaitInterval, framework.WaitTimeout,
		func() (bool, error) {
			err := targetClient.Cache().Get(context.TODO(), types.NamespacedName{
				Name:      stsNameWithUser,
				Namespace: framework.NamespaceName,
			}, &sts)
			if err != nil || sts.Status.Replicas != int32(1) {
				return false, err
			} else {
				return true, nil
			}
		})
	framework.ExpectNoError(err)
	clog.Debug("update statefulset  status: %v", sts.Status)
	framework.ExpectEqual(sts.Status.Replicas, int32(1))
	return framework.SucceedResp
}

func createStatefulsetHpa(user string) framework.TestResp {
	hpaJson := `{"apiVersion":"autoscaling/v2beta1","kind":"HorizontalPodAutoscaler","metadata":{"annotations":{},"labels":{},"name":"%s"},"spec":{"maxReplicas":2,"minReplicas":1,"metrics":[{"type":"Resource","resource":{"name":"memory","targetAverageValue":"1024"}}],"scaleTargetRef":{"apiVersion":"apps/v1","kind":"StatefulSet","name":"%s"}}}`
	postJson := fmt.Sprintf(hpaJson, stsNameWithUser, stsNameWithUser)
	hpaUrl := BuildK8sProxyUrl(framework.KubecubeHost, framework.TargetClusterName, "apis/autoscaling/v2beta1", framework.NamespaceName, "horizontalpodautoscalers", "")
	resp, err := httpHelper.RequestByUser(http.MethodPost, hpaUrl, postJson, user, nil)
	framework.ExpectNoError(err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	framework.ExpectNoError(err)
	clog.Debug("get hap response, %v", string(body))

	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to create hpa %s", stsNameWithUser), resp.StatusCode)
	}

	return framework.SucceedResp
}

func checkStatefulsetHpa(user string) framework.TestResp {
	sts := v1.StatefulSet{}
	err := wait.Poll(framework.WaitInterval, framework.WaitTimeout,
		func() (bool, error) {
			err := targetClient.Cache().Get(context.TODO(), types.NamespacedName{
				Name:      stsNameWithUser,
				Namespace: framework.NamespaceName,
			}, &sts)
			if err != nil || sts.Status.Replicas != int32(2) {
				return false, err
			} else {
				return true, nil
			}
		})
	framework.ExpectNoError(err)
	clog.Debug("hpa statefulset  status: %v", sts.Status)
	framework.ExpectEqual(sts.Status.Replicas, int32(2))
	return framework.SucceedResp
}

func deleteStatefulsetHpa(user string) framework.TestResp {
	hpaUrl := BuildK8sProxyUrl(framework.KubecubeHost, framework.TargetClusterName, "apis/autoscaling/v2beta1", framework.NamespaceName, "horizontalpodautoscalers", stsNameWithUser)
	resp, err := httpHelper.RequestByUser(http.MethodDelete, hpaUrl, "", user, nil)
	framework.ExpectNoError(err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	clog.Debug("delete hpa: %+v", string(body))

	defer resp.Body.Close()
	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to delete hpa %s", stsNameWithUser), resp.StatusCode)
	}

	return framework.SucceedResp
}

func deleteStatefulset(user string) framework.TestResp {
	url := BuildK8sProxyUrl(framework.KubecubeHost, framework.TargetClusterName, "apis/apps/v1", framework.NamespaceName, "statefulsets", stsNameWithUser)
	resp, err := httpHelper.RequestByUser(http.MethodDelete, url, "", user, nil)
	framework.ExpectNoError(err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	clog.Debug("delete sts: %+v", string(body))
	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to delete sts %s", stsNameWithUser), resp.StatusCode)
	}

	if framework.PVEnabled {
		pv1Url := BuildK8sProxyUrl(framework.KubecubeHost, framework.TargetClusterName, "api/v1", framework.NamespaceName, "persistentvolumeclaims", "pv1-"+stsNameWithUser+"-0")
		resp, err = httpHelper.RequestByUser(http.MethodDelete, pv1Url, "", user, nil)
		framework.ExpectNoError(err)
		defer resp.Body.Close()
		body, err = io.ReadAll(resp.Body)
		clog.Debug("delete pv1: %+v", string(body))
		if !framework.IsSuccess(resp.StatusCode) {
			clog.Warn("res code %d", resp.StatusCode)
			return framework.NewTestResp(fmt.Errorf("fail to delete sts pv1 %s", stsNameWithUser), resp.StatusCode)
		}
		pv2Url := BuildK8sProxyUrl(framework.KubecubeHost, framework.TargetClusterName, "api/v1", framework.NamespaceName, "persistentvolumeclaims", "pv1-"+stsNameWithUser+"-1")
		resp, err = httpHelper.RequestByUser(http.MethodDelete, pv2Url, "", user, nil)
		framework.ExpectNoError(err)
		defer resp.Body.Close()
		body, err = io.ReadAll(resp.Body)
		clog.Debug("delete pv2: %+v", string(body))
		if !framework.IsSuccess(resp.StatusCode) {
			clog.Warn("res code %d", resp.StatusCode)
			return framework.NewTestResp(fmt.Errorf("fail to delete sts pv2 %s", stsNameWithUser), resp.StatusCode)
		}
	}

	time.Sleep(time.Minute)
	return framework.SucceedResp
}

var multiUserStsTest = framework.MultiUserTest{
	TestName:        "[工作负载][9478763]创建StatefulSet工作负载挂载卷",
	ContinueIfError: false,
	Skipfunc: func() bool {
		return !framework.StatefulSetEnable
	},
	ErrorFunc:  framework.PermissionErrorFunc,
	AfterEach:  nil,
	BeforeEach: nil,
	InitStep:   nil,
	FinalStep:  nil,
	Steps: []framework.MultiUserTestStep{
		{
			Name:        "创建Statefulset",
			Description: "",
			StepFunc:    createStatefulset,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
		{
			Name:        "创建StatefulSet工作负载挂载卷",
			Description: "检查容器挂载pv1到/mnt1/成功",
			StepFunc:    checkStatefulsetVolume,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
		{
			Name: "StatefulSet健康检查部署策略",
			Description: "1.负载正常运行，检查负载副本为sts1-0、sts1-1" +
				"2.查看副本sts1-0、sts1-1均调度到此节点上",
			StepFunc: checkStatefulsetRunning,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
		{
			Name:        "都可以查看到准确的对应信息",
			Description: "查看容器详情",
			StepFunc:    checkStatefulsetDetail,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
		{
			Name:        "与配置一致，信息准确",
			Description: "",
			StepFunc:    checkStatefulsetInfo,
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
			StepFunc:    checkStatefulsetPerformance,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
		{
			Name:        "Statefulset和副本的conditions与k8s查询一致",
			Description: "查看Statefulset和副本的condition详情与k8s的是否一致",
			StepFunc:    checkStatefulsetCondition,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
		{
			Name:        "Statefulset事件前端页面和后台k8s命令显示一致",
			Description: "查看Statefulset事件",
			StepFunc:    checkStatefulsetEvents,
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
			StepFunc:    checkStatefulsetPodEvents,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
		{
			Name:        "调整负载副本数",
			Description: "更新副本个数为1",
			StepFunc:    restStatefulsetReplica,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
		{
			Name:        "自动伸缩设置",
			Description: "创建hap，设置触发条件memory为1024，保证一定会达到条件",
			StepFunc:    createStatefulsetHpa,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
		{
			Name:        "容器副本数扩容到2",
			Description: "",
			StepFunc:    checkStatefulsetHpa,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
		{
			Name:        "clean statefulSet hpa",
			Description: "clean statefulSet",
			StepFunc:    deleteStatefulsetHpa,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
		{
			Name:        "clean statefulSet",
			Description: "clean statefulSet",
			StepFunc:    deleteStatefulset,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
	},
}

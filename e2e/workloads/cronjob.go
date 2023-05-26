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
	v1 "k8s.io/api/batch/v1"
	"k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubecube-io/kubecube-e2e/e2e/framework"
)

func createCronjob(user string) framework.TestResp {
	initParam()
	cronJobNameWithUser = framework.NameWithUser(cronJobName, user)
	cronJobJson := `{"apiVersion":"batch/v1beta1","kind":"CronJob","metadata":{"name":"%s","annotations":{},"labels":{"kubecube.io/app":"%s"}},"spec":{"selector":{"matchLabels":{"kubecube.io/app":"%s"}},"concurrencyPolicy":null,"schedule":"*/1 * * * *","successfulJobsHistoryLimit":null,"failedJobsHistoryLimit":null,"jobTemplate":{"spec":{"completions":null,"parallelism":null,"backoffLimit":null,"template":{"metadata":{"annotations":{},"labels":{"kubecube.io/app":"%s"}},"spec":{"containers":[{"name":"%s","args":["Hello from the Kubernetes cluste"],"command":["echo"],"env":[],"image":"%s","imagePullPolicy":"IfNotPresent","lifecycle":{"postStart":null,"preStop":null},"livenessProbe":null,"readinessProbe":null,"ports":null,"resources":{"limits":{"cpu":"100m","memory":"128Mi"},"requests":{"cpu":"100m","memory":"128Mi"}},"volumeMounts":[]}],"initContainers":[],"volumes":[],"affinity":{},"restartPolicy":"OnFailure","imagePullSecrets":[{"name":"%s"}]}}}}}}`
	cronJobJson = fmt.Sprintf(cronJobJson, cronJobNameWithUser, cronJobNameWithUser, cronJobNameWithUser, cronJobNameWithUser, cronJobNameWithUser, framework.TestImage, framework.ImagePullSecret)
	url := BuildK8sProxyUrl(framework.KubecubeHost, framework.TargetClusterName, "apis/batch/v1beta1", framework.NamespaceName, "cronjobs", "")
	cronJobResp, err := httpHelper.RequestByUser(http.MethodPost, url, cronJobJson, user, nil)
	framework.ExpectNoError(err)
	defer cronJobResp.Body.Close()
	body, err := io.ReadAll(cronJobResp.Body)
	framework.ExpectNoError(err)
	clog.Info("create cronJob %v, %v", cronJobNameWithUser, string(body))

	if !framework.IsSuccess(cronJobResp.StatusCode) {
		clog.Warn("res code %d", cronJobResp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to create cronjob %s", cronJobNameWithUser), cronJobResp.StatusCode)
	}

	cronJob := v1beta1.CronJob{}
	err = wait.Poll(framework.WaitInterval, framework.WaitTimeout,
		func() (bool, error) {
			err := targetClient.Cache().Get(context.TODO(), types.NamespacedName{
				Name:      cronJobNameWithUser,
				Namespace: framework.NamespaceName,
			}, &cronJob)
			if err != nil {
				return false, nil
			} else {
				return true, nil
			}
		})
	framework.ExpectNoError(err)
	return framework.SucceedResp
}

func checkCronjobCreate(user string) framework.TestResp {
	cronJob := v1beta1.CronJob{}
	err := wait.Poll(framework.WaitInterval, framework.WaitTimeout,
		func() (bool, error) {
			err := targetClient.Cache().Get(context.TODO(), types.NamespacedName{
				Name:      cronJobNameWithUser,
				Namespace: framework.NamespaceName,
			}, &cronJob)
			if err != nil {
				return false, nil
			} else {
				return true, nil
			}
		})
	framework.ExpectNoError(err)
	clog.Info("cronjob status: %v", cronJob.Status)
	return framework.SucceedResp
}

func checkCronjobInfo(user string) framework.TestResp {
	cronJob := v1beta1.CronJob{}
	err := wait.Poll(framework.WaitInterval, framework.WaitTimeout,
		func() (bool, error) {
			err := targetClient.Cache().Get(context.TODO(), types.NamespacedName{
				Name:      cronJobNameWithUser,
				Namespace: framework.NamespaceName,
			}, &cronJob)
			if err != nil {
				return false, nil
			} else {
				return true, nil
			}
		})
	framework.ExpectNoError(err)
	clog.Info("查看CronJob列表信息")
	framework.ExpectEqual(cronJob.Name, cronJobNameWithUser)
	framework.ExpectEqual(cronJob.Namespace, framework.NamespaceName)
	framework.ExpectEqual(cronJob.Spec.Schedule, "*/1 * * * *")
	return framework.SucceedResp
}

func checkCronjobStatus(user string) framework.TestResp {
	jobList := v1.JobList{}
	err := wait.Poll(framework.WaitInterval, framework.WaitTimeout,
		func() (bool, error) {
			err := targetClient.Cache().List(context.TODO(), &jobList, &client.ListOptions{
				Namespace:     framework.NamespaceName,
				LabelSelector: labels.Set{"kubecube.io/app": cronJobNameWithUser}.AsSelector(),
			})
			if err != nil || len(jobList.Items) == 0 || jobList.Items[0].Status.Succeeded != 1 {
				return false, err
			} else {
				return true, nil
			}
		})
	framework.ExpectNoError(err)
	framework.ExpectNotEqual(len(jobList.Items), 0)
	framework.ExpectEqual(jobList.Items[0].Status.Conditions[0].Type, v1.JobComplete)
	return framework.SucceedResp
}

func updateCronjob(user string) framework.TestResp {
	updateJson := `{"apiVersion":"batch/v1beta1","kind":"CronJob","metadata":{"labels":{"kubecube.io/app":"%s"},"name":"%s","namespace":"%s"},"spec":{"failedJobsHistoryLimit":1,"jobTemplate":{"metadata":{"creationTimestamp":null},"spec":{"template":{"metadata":{"creationTimestamp":null,"labels":{"kubecube.io/app":"%s"},"annotations":{}},"spec":{"affinity":{},"containers":[{"name":"%s","args":["-c","date;echo  Hello from the Kubernetes cluste"],"command":["/bin/bash"],"env":[],"image":"%s","imagePullPolicy":"IfNotPresent","lifecycle":{"postStart":null,"preStop":null},"livenessProbe":null,"readinessProbe":null,"ports":null,"resources":{"limits":{"cpu":"100m","memory":"128Mi"},"requests":{"cpu":"100m","memory":"128Mi"}},"volumeMounts":[]}],"dnsPolicy":"ClusterFirst","restartPolicy":"OnFailure","schedulerName":"default-scheduler","securityContext":{},"terminationGracePeriodSeconds":30,"initContainers":[],"imagePullSecrets":[{"name":"%s"}],"volumes":[]}}}},"schedule":"0 0 */1 * *","successfulJobsHistoryLimit":3,"suspend":false}}`
	updateJson = fmt.Sprintf(updateJson, cronJobNameWithUser, cronJobNameWithUser, framework.NamespaceName, cronJobNameWithUser, cronJobNameWithUser, framework.TestImage, framework.ImagePullSecret)
	url := BuildK8sProxyUrl(framework.KubecubeHost, framework.TargetClusterName, "apis/batch/v1beta1", framework.NamespaceName, "cronjobs", cronJobNameWithUser)
	body, err := httpHelper.RequestByUser(http.MethodPut, url, updateJson, user, nil)
	framework.ExpectNoError(err)
	defer body.Body.Close()

	if !framework.IsSuccess(body.StatusCode) {
		clog.Warn("res code %d", body.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to update cronjob %s", cronJobNameWithUser), body.StatusCode)
	}

	cronJob := v1beta1.CronJob{}
	time.Sleep(time.Second * 20)
	err = targetClient.Cache().Get(context.TODO(), types.NamespacedName{
		Name:      cronJobNameWithUser,
		Namespace: framework.NamespaceName,
	}, &cronJob)
	framework.ExpectNoError(err)
	framework.ExpectEqual(cronJob.Spec.Schedule, "0 0 */1 * *")
	return framework.SucceedResp
}

func checkUpdatedCronjob(user string) framework.TestResp {
	cronJob := v1beta1.CronJob{}
	err := wait.Poll(framework.WaitInterval, framework.WaitTimeout,
		func() (bool, error) {
			err := targetClient.Cache().Get(context.TODO(), types.NamespacedName{
				Name:      cronJobNameWithUser,
				Namespace: framework.NamespaceName,
			}, &cronJob)
			if err != nil {
				return false, nil
			} else {
				return true, nil
			}
		})
	framework.ExpectNoError(err)
	framework.ExpectEqual(len(cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers), 1)
	container := cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0]
	framework.ExpectEqual(container.Image, framework.TestImage)
	framework.ExpectEqual(container.Command[0], "/bin/bash")
	framework.ExpectEqual(container.Args[0], "-c")
	framework.ExpectEqual(container.Args[1], "date;echo  Hello from the Kubernetes cluste")
	return framework.SucceedResp
}

func checkUpdatedCronjobStatus(user string) framework.TestResp {
	cronJob := v1beta1.CronJob{}
	err := wait.Poll(framework.WaitInterval, framework.WaitTimeout,
		func() (bool, error) {
			err := targetClient.Cache().Get(context.TODO(), types.NamespacedName{
				Name:      cronJobNameWithUser,
				Namespace: framework.NamespaceName,
			}, &cronJob)
			if err != nil {
				return false, nil
			} else {
				return true, nil
			}
		})
	framework.ExpectNoError(err)
	url := BuildEventUrl(framework.KubecubeHost, framework.TargetClusterName, framework.NamespaceName, string(cronJob.UID))
	resp, err := httpHelper.RequestByUser(http.MethodGet, url, "", user, nil)
	framework.ExpectNoError(err)
	body, err := io.ReadAll(resp.Body)
	framework.ExpectNoError(err)

	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to get cronjob %s", cronJobNameWithUser), resp.StatusCode)
	}

	eventList := corev1.EventList{}
	err = json.Unmarshal(body, &eventList)
	framework.ExpectNoError(err)
	framework.ExpectNotEqual(len(eventList.Items), 0)
	return framework.SucceedResp
}

func deleteCronjob(user string) framework.TestResp {
	url := BuildK8sProxyUrl(framework.KubecubeHost, framework.TargetClusterName, "apis/batch/v1beta1", framework.NamespaceName, "cronjobs", cronJobNameWithUser)
	resp, err := httpHelper.RequestByUser(http.MethodDelete, url, "", user, nil)
	framework.ExpectNoError(err)
	defer resp.Body.Close()

	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to delete cronjob %s", cronJobNameWithUser), resp.StatusCode)
	}
	time.Sleep(time.Minute)
	return framework.SucceedResp
}

var multiUserCronjobTest = framework.MultiUserTest{
	TestName:        "[工作负载][9478777]CronJob检查",
	ContinueIfError: false,
	Skipfunc: func() bool {
		return !framework.CronJobEnable
	},
	ErrorFunc:  framework.PermissionErrorFunc,
	AfterEach:  nil,
	BeforeEach: nil,
	InitStep:   nil,
	FinalStep:  nil,
	Steps: []framework.MultiUserTestStep{
		{
			Name: "创建实例hellocronjob",
			Description: "进入容器云》工作负载》CronJob创建实例hellocronjob" +
				"填入以下信息：" +
				"镜像：选择library镜像如tomcat" +
				"高级》启动命令：" +
				"- /bin/bash" +
				"- '-c'" +
				"- 'date;echo  Hello from the Kubernetes cluste'" +
				"定时规则》定时调度设置：*/1 * * * *" +
				"其他配置任选后提交",
			StepFunc: createCronjob,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
		{
			Name:        "CronJob创建成功",
			Description: "CronJob创建成功",
			StepFunc:    checkCronjobCreate,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
		{
			Name:        "CronJob列表信息准确，任务列表、事件信息准确",
			Description: "查看CronJob列表信息",
			StepFunc:    checkCronjobInfo,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
		{
			Name:        "Job列表中有新增Job且状态为执行完成",
			Description: "Job列表中有新增Job且状态为执行完成",
			StepFunc:    checkCronjobStatus,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
		{
			Name:        "设置CronJob配置生效",
			Description: "设置CronJob配置生效",
			StepFunc:    updateCronjob,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
		{
			Name:        "工作负载详情检查",
			Description: "与配置一致，信息准确",
			StepFunc:    checkUpdatedCronjob,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
		{
			Name:        "CronJob和副本事件前端页面和后台k8s命令显示一致",
			Description: "通过前台界面和后台k8s命令查看CronJob和副本事件展示是否一致",
			StepFunc:    checkUpdatedCronjobStatus,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
		{
			Name:        "clear cornjob",
			Description: "clear cornjob",
			StepFunc:    deleteCronjob,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
	},
}

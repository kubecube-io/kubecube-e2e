package workloads_new

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kubecube-io/kubecube/pkg/clog"
	"github.com/onsi/ginkgo"
	v1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubecube-io/kubecube-e2e/e2e/framework"
)

func createJob(user string) framework.TestResp {
	initParam()
	jobNameWithUser = framework.NameWithUser(jobName, user)
	jobJson := `{"apiVersion":"batch/v1","kind":"Job","metadata":{"name":"%s","annotations":{},"labels":{"kubecube.io/app":"%s"}},"spec":{"selector":{},"template":{"metadata":{"annotations":{},"labels":{"kubecube.io/app":"%s"}},"spec":{"containers":[{"name":"%s","args":["Hello from the Kubernetes cluste"],"command":["echo"],"env":[],"image":"%s","imagePullPolicy":"IfNotPresent","lifecycle":{"postStart":null,"preStop":null},"livenessProbe":null,"readinessProbe":null,"ports":null,"resources":{"limits":{"cpu":"100m","memory":"128Mi"},"requests":{"cpu":"100m","memory":"128Mi"}},"volumeMounts":[]}],"initContainers":[],"imagePullSecrets":[{"name":"%s"}],"volumes":[],"affinity":{},"restartPolicy":"OnFailure"}},"completions":1,"parallelism":1,"backoffLimit":6}}`
	jobJson = fmt.Sprintf(jobJson, jobNameWithUser, jobNameWithUser, jobNameWithUser, jobNameWithUser, framework.TestImage, framework.ImagePullSecret)
	url := BuildK8sProxyUrl(framework.KubecubeHost, framework.TargetClusterName, "apis/batch/v1", framework.NamespaceName, "jobs", "")
	jobResp, err := httpHelper.RequestByUser(http.MethodPost, url, jobJson, user, nil)
	framework.ExpectNoError(err)
	defer jobResp.Body.Close()
	body, err := io.ReadAll(jobResp.Body)
	framework.ExpectNoError(err)
	clog.Debug("create job %v, %v", jobNameWithUser, string(body))

	if !framework.IsSuccess(jobResp.StatusCode) {
		clog.Warn("res code %d", jobResp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to create job %s", jobNameWithUser), jobResp.StatusCode)
	}

	time.Sleep(time.Second * 10)
	return framework.SucceedResp
}

func checkJob(user string) framework.TestResp {
	job := v1.Job{}
	err := targetClient.Cache().Get(context.TODO(), types.NamespacedName{
		Name:      jobNameWithUser,
		Namespace: framework.NamespaceName,
	}, &job)
	framework.ExpectNoError(err)
	clog.Debug("create job status: %v", job.Status)
	return framework.SucceedResp
}

func checkJobList(user string) framework.TestResp {
	jobList := v1.JobList{}
	err := targetClient.Cache().List(context.TODO(), &jobList, &client.ListOptions{
		Namespace:     framework.NamespaceName,
		LabelSelector: labels.Set{"kubecube.io/app": jobNameWithUser}.AsSelector(),
	})
	framework.ExpectNoError(err)
	framework.ExpectEqual(len(jobList.Items), 1)
	job := jobList.Items[0]
	framework.ExpectEqual(job.Name, jobNameWithUser)
	framework.ExpectEqual(job.Status.Conditions[0].Type, v1.JobComplete)
	return framework.SucceedResp
}

func checkJobInfo(user string) framework.TestResp {
	podList := corev1.PodList{}
	err := targetClient.Cache().List(context.TODO(), &podList, &client.ListOptions{
		Namespace:     framework.NamespaceName,
		LabelSelector: labels.Set{"kubecube.io/app": jobNameWithUser}.AsSelector(),
	})
	framework.ExpectNoError(err)
	framework.ExpectEqual(len(podList.Items), 1)
	pod := podList.Items[0]
	framework.ExpectEqual(len(pod.Spec.Containers), 1)
	container := pod.Spec.Containers[0]
	framework.ExpectEqual(container.Image, framework.TestImage)
	framework.ExpectEqual(container.Command[0], "echo")
	framework.ExpectEqual(container.Args[0], "Hello from the Kubernetes cluste")
	return framework.SucceedResp
}

func checkJobDetail(user string) framework.TestResp {
	podList := corev1.PodList{}
	err := targetClient.Cache().List(context.TODO(), &podList, &client.ListOptions{
		Namespace:     framework.NamespaceName,
		LabelSelector: labels.Set{"kubecube.io/app": jobNameWithUser}.AsSelector(),
	})
	framework.ExpectNoError(err)
	framework.ExpectEqual(len(podList.Items), 1)
	pod := podList.Items[0]
	ginkgo.By("查看容器详情")
	framework.ExpectEqual(len(pod.Spec.Containers), 1)
	container := pod.Spec.Containers[0]
	framework.ExpectEqual(container.Image, framework.TestImage)

	ginkgo.By("查看容器日志")
	url := BuildLogUrl(framework.KubecubeHost, framework.TargetClusterName, framework.NamespaceName, pod.Name, container.Name)
	logResp, err := httpHelper.RequestByUser(http.MethodGet, url, "", user, nil)
	framework.ExpectNoError(err)
	defer logResp.Body.Close()

	if !framework.IsSuccess(logResp.StatusCode) {
		clog.Warn("res code %d", logResp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to get pod %s logs", pod.Name), logResp.StatusCode)
	}

	framework.ExpectEqual(logResp.StatusCode, 200)
	return framework.SucceedResp
}

func checkJobPerformance(user string) framework.TestResp {
	// TODO
	return framework.SucceedResp
}

func checkJobCondition(user string) framework.TestResp {
	job := v1.Job{}
	err := targetClient.Cache().Get(context.TODO(), types.NamespacedName{
		Name:      jobNameWithUser,
		Namespace: framework.NamespaceName,
	}, &job)
	framework.ExpectNoError(err)
	framework.ExpectNotEqual(len(job.Status.Conditions), 0)
	podList := corev1.PodList{}
	err = targetClient.Cache().List(context.TODO(), &podList, &client.ListOptions{
		Namespace:     framework.NamespaceName,
		LabelSelector: labels.Set{"kubecube.io/app": jobNameWithUser}.AsSelector(),
	})
	framework.ExpectNoError(err)
	framework.ExpectEqual(len(podList.Items), 1)
	pod := podList.Items[0]
	framework.ExpectNotEqual(len(pod.Status.Conditions), 0)
	return framework.SucceedResp
}

func checkJobEvents(user string) framework.TestResp {
	job := v1.Job{}
	err := targetClient.Cache().Get(context.TODO(), types.NamespacedName{
		Name:      jobNameWithUser,
		Namespace: framework.NamespaceName,
	}, &job)
	framework.ExpectNoError(err)
	url := BuildEventUrl(framework.KubecubeHost, framework.TargetClusterName, framework.NamespaceName, string(job.UID))
	resp, err := httpHelper.RequestByUser(http.MethodGet, url, "", user, nil)
	framework.ExpectNoError(err)
	body, err := io.ReadAll(resp.Body)
	framework.ExpectNoError(err)
	defer resp.Body.Close()

	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to get job %s events", jobNameWithUser), resp.StatusCode)
	}

	eventList := corev1.EventList{}
	err = json.Unmarshal(body, &eventList)
	framework.ExpectNoError(err)
	framework.ExpectNotEqual(len(eventList.Items), 0)
	return framework.SucceedResp
}

func checkJobPodEvents(user string) framework.TestResp {
	podList := corev1.PodList{}
	err := targetClient.Cache().List(context.TODO(), &podList, &client.ListOptions{
		Namespace:     framework.NamespaceName,
		LabelSelector: labels.Set{"kubecube.io/app": jobNameWithUser}.AsSelector(),
	})
	framework.ExpectNoError(err)
	framework.ExpectEqual(len(podList.Items), 1)
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

func deleteJob(user string) framework.TestResp {
	url := BuildK8sProxyUrl(framework.KubecubeHost, framework.TargetClusterName, "apis/batch/v1", framework.NamespaceName, "jobs", jobNameWithUser)
	resp, err := httpHelper.Delete(url)
	framework.ExpectNoError(err)
	defer resp.Body.Close()

	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to delete job %s", jobNameWithUser), resp.StatusCode)
	}

	time.Sleep(time.Minute)
	return framework.SucceedResp
}

var multiUserJobTest = framework.MultiUserTest{
	TestName:        "[工作负载][9478778]Job检查",
	ContinueIfError: false,
	Skipfunc: func() bool {
		return !framework.JobEnable
	},
	ErrorFunc:  framework.PermissionErrorFunc,
	AfterEach:  nil,
	BeforeEach: nil,
	InitStep:   nil,
	FinalStep:  nil,
	Steps: []framework.MultiUserTestStep{
		{
			Name: "创建实例hellojob",
			Description: "进入容器云》工作负载》Job创建实例hellojob" +
				"填入以下信息：" +
				"镜像：选择library镜像如tomcat" +
				"高级》启动命令：" +
				"- /bin/bash" +
				"- '-c'" +
				"- 'date;echo  Hello from the Kubernetes cluste'",
			StepFunc: createJob,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
		{
			Name:        "Job创建成功",
			Description: "Job创建成功",
			StepFunc:    checkJob,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
		{
			Name:        "Job列表中有新增Job且状态为执行完成，其他信息准确",
			Description: "查看Job列表信息",
			StepFunc:    checkJobList,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
		{
			Name:        "与配置一致，信息准确",
			Description: "检查副本详情页信息",
			StepFunc:    checkJobInfo,
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
			StepFunc:    checkJobDetail,
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
			StepFunc:    checkJobPerformance,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
		{
			Name:        "Job和副本的conditions与k8s查询一致",
			Description: "查看Job和副本的condition详情与k8s的是否一致",
			StepFunc:    checkJobCondition,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
		{
			Name:        "Job事件前端页面和后台k8s命令显示一致",
			Description: "查看Job事件",
			StepFunc:    checkJobEvents,
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
			StepFunc:    checkJobPodEvents,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
		{
			Name:        "clean job",
			Description: "clean job",
			StepFunc:    deleteJob,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
	},
}

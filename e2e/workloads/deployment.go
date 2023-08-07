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

	"github.com/onsi/ginkgo"
	"github.com/tidwall/gjson"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubecube-io/kubecube-e2e/e2e/framework"
	"github.com/kubecube-io/kubecube/pkg/clog"
)

func createDeployWithPvc(user string) framework.TestResp {
	ginkgo.By("创建存储声明pv1，容量100Mi、创建方式动态持久化存储、独占读写模式")
	pv1Json := "{\"apiVersion\":\"v1\",\"kind\":\"PersistentVolumeClaim\",\"metadata\":{\"name\":\"" + pv1NameWithUser + "\",\"annotations\":{},\"labels\":{}},\"spec\":{\"storageClassName\":\"" + framework.StorageClass + "\",\"accessModes\":[\"ReadWriteOnce\"],\"resources\":{\"requests\":{\"storage\":\"100Mi\"}}}}"
	url := BuildK8sProxyUrl(framework.KubecubeHost, framework.TargetClusterName, "api/v1", framework.NamespaceName, "persistentvolumeclaims", "")
	pv1Response, err := httpHelper.RequestByUser(http.MethodPost, url, pv1Json, user, nil)
	framework.ExpectNoError(err)
	defer pv1Response.Body.Close()
	body, err := io.ReadAll(pv1Response.Body)
	framework.ExpectNoError(err)
	clog.Info("get pvc pv1, %v", string(body))

	if !framework.IsSuccess(pv1Response.StatusCode) {
		clog.Warn("res code %d", pv1Response.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to create pvc %s", pv1NameWithUser), pv1Response.StatusCode)
	}

	ginkgo.By("创建存储声明pv2，容量200Mi、创建方式动态持久化存储、只读共享")
	pv2Json := "{\"apiVersion\":\"v1\",\"kind\":\"PersistentVolumeClaim\",\"metadata\":{\"name\":\"" + pv2NameWithUser + "\",\"annotations\":{},\"labels\":{}},\"spec\":{\"storageClassName\":\"" + framework.StorageClass + "\",\"accessModes\":[\"ReadOnlyMany\"],\"resources\":{\"requests\":{\"storage\":\"200Mi\"}}}}"
	pv2Response, err := httpHelper.RequestByUser(http.MethodPost, url, pv2Json, user, nil)
	framework.ExpectNoError(err)
	defer pv2Response.Body.Close()
	body, err = io.ReadAll(pv2Response.Body)
	framework.ExpectNoError(err)
	clog.Info("get pvc pv2, %v", string(body))

	if !framework.IsSuccess(pv2Response.StatusCode) {
		clog.Warn("res code %d", pv2Response.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to create pvc %s", pv2NameWithUser), pv2Response.StatusCode)
	}

	ginkgo.By("1.在空间内创建工作负载使用demo镜像：hub.c.163.com/qingzhou/nsf-demo-stock-viewer:online-20180921-181512-b09dbe78,填写副本数1")
	ginkgo.By("2. 环境变量NCE_PORT=18080 NCE_JAVA_OPTS=-Dstock_provider_url=http://demo-data.ns2:8088（供空间内服务发现case使用）")
	ginkgo.By("3. 规格选低性能基础配置*5 （50M Cores/50 MiB）")
	ginkgo.By("4. 选择高级模式，pvc1挂载/mnt1/目录；挂载pvc2到/mnt2/目录")
	ginkgo.By("5. 添加启动命令 sh\n启动命令参数\n-c\nwhile true;do echo hello;sleep 1;done")
	ginkgo.By("6. 设置标签label1=label1")
	deployJson := `{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"%s","annotations":{},"labels":{"kubecube.io/app":"%s"}},"spec":{"selector":{"matchLabels":{"kubecube.io/app":"%s"}},"template":{"metadata":{"annotations":{},"labels":{"label1":"label1","kubecube.io/app":"%s"}},"spec":{"containers":[{"name":"%s","args":["-c","while true;do echo hello;sleep 1;done"],"command":["sh"],"env":[{"name":"NCE_PORT","value":"18080"},{"name":"NCE_JAVA_OPTS","value":"-Dstock_provider_url=http://demo-data.ns2:8088"}],"image":"%s","imagePullPolicy":"IfNotPresent","lifecycle":{"postStart":null,"preStop":null},"livenessProbe":null,"readinessProbe":null,"ports":null,"resources":{"limits":{"cpu":"50m","memory":"50Mi"},"requests":{"cpu":"50m","memory":"50Mi"}},"volumeMounts":[{"name":"data-volume-0-0","readOnly":false,"mountPath":"/mnt1/","subPath":""},{"name":"data-volume-0-1","readOnly":false,"mountPath":"/mnt2/","subPath":""}]}],"initContainers":[],"imagePullSecrets":[{"name":"%s"}],"volumes":[{"name":"data-volume-0-0","persistentVolumeClaim":{"claimName":"%s","readOnly":false}},{"name":"data-volume-0-1","persistentVolumeClaim":{"claimName":"%s","readOnly":false}}],"affinity":{},"restartPolicy":"Always"}},"replicas":1}}`
	deployJson = fmt.Sprintf(deployJson, deployNameWithUser, deployNameWithUser, deployNameWithUser, deployNameWithUser, deployNameWithUser, framework.TestImage, framework.ImagePullSecret, pv1NameWithUser, pv2NameWithUser)
	deployUrl := BuildK8sProxyUrl(framework.KubecubeHost, framework.TargetClusterName, "apis/apps/v1", framework.NamespaceName, "deployments", "")
	deployResponse, err := httpHelper.RequestByUser(http.MethodPost, deployUrl, deployJson, user, nil)
	framework.ExpectNoError(err)
	defer deployResponse.Body.Close()
	body, err = io.ReadAll(deployResponse.Body)
	framework.ExpectNoError(err)
	clog.Info("create deploy %v, %v", deployNameWithUser, string(body))

	if !framework.IsSuccess(deployResponse.StatusCode) {
		clog.Warn("res code %d", deployResponse.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to create deploy %s", deployNameWithUser), deployResponse.StatusCode)
	}

	deploy := v1.Deployment{}
	err = wait.Poll(framework.WaitInterval, framework.WaitTimeout,
		func() (bool, error) {
			err := targetClient.Cache().Get(context.TODO(), types.NamespacedName{
				Name:      deployNameWithUser,
				Namespace: framework.NamespaceName,
			}, &deploy)
			if err != nil {
				return false, nil
			} else {
				return true, nil
			}
		})
	framework.ExpectNoError(err)
	return framework.SucceedResp
}

func createDeployWithoutPvc(user string) framework.TestResp {
	ginkgo.By("1.在空间内创建工作负载使用demo镜像：hub.c.163.com/qingzhou/nsf-demo-stock-viewer:online-20180921-181512-b09dbe78,填写副本数1")
	ginkgo.By("2. 环境变量NCE_PORT=18080 NCE_JAVA_OPTS=-Dstock_provider_url=http://demo-data.ns2:8088（供空间内服务发现case使用）")
	ginkgo.By("3. 规格选低性能基础配置*5 （50M Cores/50MiB）")
	ginkgo.By("5. 添加启动命令 sh\n启动命令参数\n-c\nwhile true;do echo hello;sleep 1;done")
	ginkgo.By("6. 设置标签label1=label1")
	deployJson := `{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"%s","annotations":{},"labels":{"kubecube.io/app":"%s"}},"spec":{"selector":{"matchLabels":{"kubecube.io/app":"%s"}},"template":{"metadata":{"annotations":{},"labels":{"label1":"label1","kubecube.io/app":"%s"}},"spec":{"containers":[{"name":"%s","args":["-c","while true;do echo hello;sleep 1;done"],"command":["sh"],"env":[{"name":"NCE_PORT","value":"18080"},{"name":"NCE_JAVA_OPTS","value":"-Dstock_provider_url=http://demo-data.ns2:8088"}],"image":"%s","imagePullPolicy":"IfNotPresent","lifecycle":{"postStart":null,"preStop":null},"livenessProbe":null,"readinessProbe":null,"ports":null,"resources":{"limits":{"cpu":"50m","memory":"50Mi"},"requests":{"cpu":"50m","memory":"50Mi"}}}],"initContainers":[],"imagePullSecrets":[{"name":"%s"}],"affinity":{},"restartPolicy":"Always"}},"replicas":1}}`
	deployJson = fmt.Sprintf(deployJson, deployNameWithUser, deployNameWithUser, deployNameWithUser, deployNameWithUser, deployNameWithUser, framework.TestImage, framework.ImagePullSecret)
	deployUrl := BuildK8sProxyUrl(framework.KubecubeHost, framework.TargetClusterName, "apis/apps/v1", framework.NamespaceName, "deployments", "")
	deployResponse, err := httpHelper.RequestByUser(http.MethodPost, deployUrl, deployJson, user, nil)
	framework.ExpectNoError(err)
	defer deployResponse.Body.Close()
	body, err := io.ReadAll(deployResponse.Body)
	framework.ExpectNoError(err)
	clog.Info("create deploy %v, %v", deployNameWithUser, string(body))

	if !framework.IsSuccess(deployResponse.StatusCode) {
		clog.Warn("res code %d", deployResponse.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to create deploy %s", deployNameWithUser), deployResponse.StatusCode)
	}

	deploy := v1.Deployment{}
	err = wait.Poll(framework.WaitInterval, framework.WaitTimeout,
		func() (bool, error) {
			err := targetClient.Cache().Get(context.TODO(), types.NamespacedName{
				Name:      deployNameWithUser,
				Namespace: framework.NamespaceName,
			}, &deploy)
			if err != nil {
				return false, nil
			} else {
				return true, nil
			}
		})
	framework.ExpectNoError(err)
	return framework.SucceedResp
}

func createDeploy(user string) framework.TestResp {
	initParam()
	pv1NameWithUser = framework.NameWithUser(pv1Name, user)
	pv2NameWithUser = framework.NameWithUser(pv2Name, user)
	deployNameWithUser = framework.NameWithUser(deployName, user)
	if framework.PVEnabled {
		clog.Info("createDeployWithPv")
		return createDeployWithPvc(user)
	} else {
		clog.Info("createDeployWithoutPv")
		return createDeployWithoutPvc(user)
	}
}

func checkDeploy(user string) framework.TestResp {
	err := wait.Poll(framework.WaitInterval, framework.WaitTimeout, func() (done bool, err error) {
		deploy := v1.Deployment{}
		err = targetClient.Direct().Get(context.TODO(), types.NamespacedName{
			Name:      deployNameWithUser,
			Namespace: framework.NamespaceName,
		}, &deploy)
		if err != nil {
			return false, err
		}
		framework.ExpectEqual(deploy.Name, deployNameWithUser)
		var i int32
		i = 1
		if deploy.Status.AvailableReplicas != i {
			return false, nil
		}
		return true, nil
	})

	framework.ExpectNoError(err)

	return framework.SucceedResp
}

func checkDeployLog(user string) framework.TestResp {
	podList := corev1.PodList{}
	err := wait.Poll(framework.WaitInterval, framework.WaitTimeout, func() (done bool, err error) {
		err = targetClient.Direct().List(context.TODO(), &podList, &client.ListOptions{
			Namespace:     framework.NamespaceName,
			LabelSelector: labels.Set{"kubecube.io/app": deployNameWithUser}.AsSelector(),
		})
		if err != nil {
			return false, err
		}
		if len(podList.Items) != 1 {
			return false, nil
		}
		return true, nil
	})

	framework.ExpectNoError(err)
	pod := podList.Items[0]
	framework.ExpectEqual(len(pod.Spec.Containers), 1)

	url := BuildLogUrl(framework.KubecubeHost, framework.TargetClusterName, framework.NamespaceName, pod.Name, pod.Spec.Containers[0].Name)
	logResp, err := httpHelper.RequestByUser(http.MethodGet, url, "", user, nil)
	framework.ExpectNoError(err)
	defer logResp.Body.Close()
	body, err := io.ReadAll(logResp.Body)
	framework.ExpectNoError(err)

	if !framework.IsSuccess(logResp.StatusCode) {
		clog.Warn("res code %d", logResp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to get pod %s log", pod.Name), logResp.StatusCode)
	}

	if gjson.Get(string(body), "logs.0.content").Str != "hello" {
		err = wait.Poll(framework.WaitInterval, framework.WaitTimeout, func() (done bool, err error) {
			logResp, err := httpHelper.RequestByUser(http.MethodGet, url, "", user, nil)
			framework.ExpectNoError(err)
			defer logResp.Body.Close()
			body, err := io.ReadAll(logResp.Body)
			framework.ExpectNoError(err)
			framework.ExpectEqual(framework.IsSuccess(logResp.StatusCode), true)
			if gjson.Get(string(body), "logs.0.content").Str != "hello" {
				return false, nil
			}
			return true, nil
		})
		framework.ExpectNoError(err)
	}

	return framework.SucceedResp
}

func checkDeployPv(user string) framework.TestResp {
	if !framework.PVEnabled {
		return framework.SucceedResp
	}

	podList := corev1.PodList{}
	err := targetClient.Cache().List(context.TODO(), &podList, &client.ListOptions{
		Namespace:     framework.NamespaceName,
		LabelSelector: labels.Set{"kubecube.io/app": deployNameWithUser}.AsSelector(),
	})
	framework.ExpectNoError(err)
	framework.ExpectEqual(len(podList.Items), 1)
	pod := podList.Items[0]
	framework.ExpectEqual(len(pod.Spec.Containers), 1)
	checkPv1Name := ""
	checkPv2Name := ""
	for _, volume := range pod.Spec.Volumes {
		if volume.PersistentVolumeClaim != nil && volume.PersistentVolumeClaim.ClaimName == pv1NameWithUser {
			checkPv1Name = volume.Name
			continue
		}
		if volume.PersistentVolumeClaim != nil && volume.PersistentVolumeClaim.ClaimName == pv2NameWithUser {
			checkPv2Name = volume.Name
			continue
		}
	}
	framework.ExpectNotEqual(checkPv1Name, "")
	framework.ExpectNotEqual(checkPv2Name, "")
	pv1Check := false
	pv2Check := false
	for _, volumeMount := range pod.Spec.Containers[0].VolumeMounts {
		if volumeMount.Name == checkPv1Name {
			framework.ExpectEqual(volumeMount.MountPath, "/mnt1/")
			pv1Check = true
			continue
		}
		if volumeMount.Name == checkPv2Name {
			framework.ExpectEqual(volumeMount.MountPath, "/mnt2/")
			pv2Check = true
			continue
		}
	}
	framework.ExpectEqual(pv1Check, true)
	framework.ExpectEqual(pv2Check, true)
	return framework.SucceedResp
}

func checkDeployInfo(user string) framework.TestResp {
	deploy := v1.Deployment{}
	err := targetClient.Cache().Get(context.TODO(), types.NamespacedName{
		Name:      deployNameWithUser,
		Namespace: framework.NamespaceName,
	}, &deploy)
	framework.ExpectNoError(err)
	framework.ExpectEqual(deploy.Name, deployNameWithUser)
	var i int32
	i = 1
	framework.ExpectEqual(*deploy.Spec.Replicas, i)
	return framework.SucceedResp
}

func checkDeployDetails(user string) framework.TestResp {
	deploy := v1.Deployment{}
	err := targetClient.Cache().Get(context.TODO(), types.NamespacedName{
		Name:      deployNameWithUser,
		Namespace: framework.NamespaceName,
	}, &deploy)
	framework.ExpectNoError(err)
	framework.ExpectEqual(deploy.Name, deployNameWithUser)
	var i int32
	i = 1
	framework.ExpectEqual(*deploy.Spec.Replicas, i)
	ginkgo.By("查看容器详情")
	podList := corev1.PodList{}
	err = targetClient.Cache().List(context.TODO(), &podList, &client.ListOptions{
		Namespace:     framework.NamespaceName,
		LabelSelector: labels.Set{"kubecube.io/app": deployNameWithUser}.AsSelector(),
	})
	framework.ExpectNoError(err)
	framework.ExpectEqual(len(podList.Items), 1)
	pod := podList.Items[0]
	framework.ExpectEqual(len(pod.Spec.Containers), 1)
	container := pod.Spec.Containers[0]
	framework.ExpectEqual(container.Image, framework.TestImage)
	ncePortCheck := false
	nceOptsCheck := false
	for _, env := range container.Env {
		if env.Name == "NCE_PORT" {
			framework.ExpectEqual(env.Value, "18080")
			ncePortCheck = true
			continue
		}
		if env.Name == "NCE_JAVA_OPTS" {
			framework.ExpectEqual(env.Value, "-Dstock_provider_url=http://demo-data.ns2:8088")
			nceOptsCheck = true
			continue
		}
	}
	framework.ExpectEqual(ncePortCheck, true)
	framework.ExpectEqual(nceOptsCheck, true)
	framework.ExpectEqual(container.Resources.Requests.Cpu().String(), "50m")
	framework.ExpectEqual(container.Resources.Requests.Memory().String(), "50Mi")
	framework.ExpectEqual(container.Command[0], "sh")
	framework.ExpectEqual(container.Args[0], "-c")
	framework.ExpectEqual(container.Args[1], "while true;do echo hello;sleep 1;done")
	framework.ExpectHaveKey(pod.Labels, "label1")
	return framework.SucceedResp
}

func checkDeployPerformance(user string) framework.TestResp {
	// TODO
	return framework.SucceedResp
}

func checkDeployConditions(user string) framework.TestResp {
	deploy := v1.Deployment{}
	err := targetClient.Cache().Get(context.TODO(), types.NamespacedName{
		Name:      deployNameWithUser,
		Namespace: framework.NamespaceName,
	}, &deploy)
	framework.ExpectNoError(err)
	framework.ExpectNotEqual(len(deploy.Status.Conditions), 0)
	podList := corev1.PodList{}
	err = targetClient.Cache().List(context.TODO(), &podList, &client.ListOptions{
		Namespace:     framework.NamespaceName,
		LabelSelector: labels.Set{"kubecube.io/app": deployNameWithUser}.AsSelector(),
	})
	framework.ExpectNoError(err)
	framework.ExpectEqual(len(podList.Items), 1)
	pod := podList.Items[0]
	framework.ExpectNotEqual(len(pod.Status.Conditions), 0)
	return framework.SucceedResp
}

func checkDeployEvents(user string) framework.TestResp {
	deploy := v1.Deployment{}
	err := targetClient.Cache().Get(context.TODO(), types.NamespacedName{
		Name:      deployNameWithUser,
		Namespace: framework.NamespaceName,
	}, &deploy)
	framework.ExpectNoError(err)
	url := BuildEventUrl(framework.KubecubeHost, framework.TargetClusterName, framework.NamespaceName, string(deploy.UID))
	resp, err := httpHelper.RequestByUser(http.MethodGet, url, "", user, nil)
	framework.ExpectNoError(err)
	body, err := io.ReadAll(resp.Body)
	framework.ExpectNoError(err)

	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to get deploy %s event", deployNameWithUser), resp.StatusCode)
	}

	eventList := corev1.EventList{}
	err = json.Unmarshal(body, &eventList)
	framework.ExpectNoError(err)
	framework.ExpectNotEqual(len(eventList.Items), 0)
	return framework.SucceedResp
}

func checkDeployPodEvents(user string) framework.TestResp {
	podList := corev1.PodList{}
	err := targetClient.Direct().List(context.TODO(), &podList, &client.ListOptions{
		Namespace:     framework.NamespaceName,
		LabelSelector: labels.Set{"kubecube.io/app": deployNameWithUser}.AsSelector(),
	})
	framework.ExpectNoError(err)
	framework.ExpectEqual(len(podList.Items), 1)
	pod := podList.Items[0]
	url := BuildEventUrl(framework.KubecubeHost, framework.TargetClusterName, framework.NamespaceName, string(pod.UID))
	resp, err := httpHelper.RequestByUser(http.MethodGet, url, "", user, nil)
	framework.ExpectNoError(err)
	body, err := io.ReadAll(resp.Body)
	framework.ExpectNoError(err)

	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to get pod %s event", pod.Name), resp.StatusCode)
	}

	eventList := corev1.EventList{}
	err = json.Unmarshal(body, &eventList)
	framework.ExpectNotEqual(len(eventList.Items), 0)
	return framework.SucceedResp
}

func updateDeployConfig(user string) framework.TestResp {
	if !framework.PVEnabled {
		return framework.SucceedResp
	}

	updateDeployJson := `{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"labels":{"kubecube.io/app":"%s"},"name":"%s","namespace":"%s"},"spec":{"progressDeadlineSeconds":600,"replicas":1,"revisionHistoryLimit":10,"selector":{"matchLabels":{"kubecube.io/app":"%s"}},"strategy":{"rollingUpdate":{},"type":"RollingUpdate"},"template":{"metadata":{"creationTimestamp":null,"labels":{"kubecube.io/app":"%s","label1":"label1"},"annotations":{}},"spec":{"affinity":{"nodeAffinity":{"requiredDuringSchedulingIgnoredDuringExecution":{"nodeSelectorTerms":[{"matchExpressions":[{"key":"node.kubecube.io/tenant","operator":"In","values":["share"]}]}]}}},"containers":[{"name":"%s","args":["-c","while true;do echo hello;sleep 1;done"],"command":["sh"],"env":[{"name":"NCE_PORT","value":"18080"},{"name":"NCE_JAVA_OPTS","value":"-Dstock_provider_url=http://demo-data.ns2:8088"}],"image":"%s","imagePullPolicy":"IfNotPresent","lifecycle":{"postStart":null,"preStop":null},"livenessProbe":null,"readinessProbe":null,"ports":null,"resources":{"limits":{"cpu":"50m","memory":"50Mi"},"requests":{"cpu":"50m","memory":"50Mi"}},"volumeMounts":[{"name":"data-volume-0-0","readOnly":false,"mountPath":"/mnt1"}]}],"dnsPolicy":"ClusterFirst","restartPolicy":"Always","schedulerName":"default-scheduler","securityContext":{},"terminationGracePeriodSeconds":30,"tolerations":[{"key":"node.kubecube.io","operator":"Exists","effect":"NoSchedule"}],"volumes":[{"name":"data-volume-0-0","persistentVolumeClaim":{"claimName":"%s","readOnly":false}}],"initContainers":[],"imagePullSecrets":[{"name":"%s"}]}}}}`
	updateDeployJson = fmt.Sprintf(updateDeployJson, deployNameWithUser, deployNameWithUser, framework.NamespaceName, deployNameWithUser, deployNameWithUser, deployNameWithUser, framework.TestImage, pv1NameWithUser, framework.ImagePullSecret)
	deployUrl := BuildK8sProxyUrl(framework.KubecubeHost, framework.TargetClusterName, "apis/apps/v1", framework.NamespaceName, "deployments", deployNameWithUser)
	resp, err := httpHelper.RequestByUser(http.MethodPut, deployUrl, updateDeployJson, user, nil)
	framework.ExpectNoError(err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	framework.ExpectNoError(err)
	clog.Info("update deploy resp: %s", string(body))

	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to update deploy %s", deployNameWithUser), resp.StatusCode)
	}

	podList := corev1.PodList{}
	err = wait.Poll(framework.WaitInterval, framework.WaitTimeout,
		func() (bool, error) {
			err = targetClient.Direct().List(context.TODO(), &podList, &client.ListOptions{
				Namespace:     framework.NamespaceName,
				LabelSelector: labels.Set{"kubecube.io/app": deployNameWithUser}.AsSelector(),
			})
			if err != nil {
				return false, err
			} else {
				return true, nil
			}
		})
	framework.ExpectNoError(err)
	framework.ExpectEqual(len(podList.Items), 1)
	pod := podList.Items[0]
	framework.ExpectEqual(len(pod.Spec.Containers), 1)
	checkPv1Name := ""
	checkPv2Name := ""
	for _, volume := range pod.Spec.Volumes {
		if volume.PersistentVolumeClaim != nil && volume.PersistentVolumeClaim.ClaimName == pv1NameWithUser {
			checkPv1Name = volume.Name
			continue
		}
		if volume.PersistentVolumeClaim != nil && volume.PersistentVolumeClaim.ClaimName == pv2NameWithUser {
			checkPv2Name = volume.Name
			continue
		}
	}
	framework.ExpectNotEqual(checkPv1Name, "")
	framework.ExpectEqual(checkPv2Name, "")
	pv1Check := false
	pv2Check := false
	for _, volumeMount := range pod.Spec.Containers[0].VolumeMounts {
		if volumeMount.Name == checkPv1Name {
			framework.ExpectEqual(volumeMount.MountPath, "/mnt1")
			pv1Check = true
			continue
		}
		if volumeMount.Name == checkPv2Name {
			framework.ExpectEqual(volumeMount.MountPath, "/mnt2")
			pv2Check = true
			continue
		}
	}
	framework.ExpectEqual(pv1Check, true)
	framework.ExpectEqual(pv2Check, false)
	return framework.SucceedResp
}

func updateDeployReplica(user string) framework.TestResp {
	patchJson := "{\"spec\":{\"replicas\":4}}"
	header := map[string]string{
		"Content-Type": "application/strategic-merge-patch+json",
	}
	deployUrl := BuildK8sProxyUrl(framework.KubecubeHost, framework.TargetClusterName, "apis/apps/v1", framework.NamespaceName, "deployments", deployNameWithUser)
	resp, err := httpHelper.RequestByUser(http.MethodPatch, deployUrl, patchJson, user, header)
	framework.ExpectNoError(err)
	defer resp.Body.Close()

	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to update deploy %s replica", deployNameWithUser), resp.StatusCode)
	}

	ginkgo.By("检查副本详情页信息")
	deploy := v1.Deployment{}
	err = wait.Poll(framework.WaitInterval, framework.WaitTimeout,
		func() (bool, error) {
			err = targetClient.Cache().Get(context.TODO(), types.NamespacedName{
				Name:      deployNameWithUser,
				Namespace: framework.NamespaceName,
			}, &deploy)
			if err != nil || *deploy.Spec.Replicas != int32(4) {
				return false, err
			} else {
				return true, nil
			}
		})
	framework.ExpectNoError(err)
	framework.ExpectEqual(*deploy.Spec.Replicas, int32(4))
	return framework.SucceedResp
}

func resetDeployReplica(user string) framework.TestResp {
	patchJson := "{\"spec\":{\"replicas\":1}}"
	header := map[string]string{
		"Content-Type": "application/strategic-merge-patch+json",
	}
	deployUrl := BuildK8sProxyUrl(framework.KubecubeHost, framework.TargetClusterName, "apis/apps/v1", framework.NamespaceName, "deployments", deployNameWithUser)
	resp, err := httpHelper.RequestByUser(http.MethodPatch, deployUrl, patchJson, user, header)
	framework.ExpectNoError(err)
	defer resp.Body.Close()

	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to reset deploy %s replica", deployNameWithUser), resp.StatusCode)
	}

	deploy := v1.Deployment{}
	err = wait.Poll(framework.WaitInterval, framework.WaitTimeout,
		func() (bool, error) {
			err := targetClient.Cache().Get(context.TODO(), types.NamespacedName{
				Name:      deployNameWithUser,
				Namespace: framework.NamespaceName,
			}, &deploy)
			if err != nil || deploy.Status.Replicas != int32(1) {
				return false, err
			} else {
				return true, nil
			}
		})
	framework.ExpectNoError(err)
	clog.Info("update deployment status: %v", deploy.Status)
	framework.ExpectEqual(deploy.Status.Replicas, int32(1))
	return framework.SucceedResp
}

func setDeployHpa(user string) framework.TestResp {
	hpaJson := `{"apiVersion":"autoscaling/v2beta1","kind":"HorizontalPodAutoscaler","metadata":{"annotations":{},"labels":{},"name":"%s"},"spec":{"maxReplicas":2,"minReplicas":1,"metrics":[{"type":"Resource","resource":{"name":"memory","targetAverageValue":"1024"}}],"scaleTargetRef":{"apiVersion":"apps/v1","kind":"Deployment","name":"%s"}}}`
	postJson := fmt.Sprintf(hpaJson, deployNameWithUser, deployNameWithUser)
	hpaUrl := BuildK8sProxyUrl(framework.KubecubeHost, framework.TargetClusterName, "apis/autoscaling/v2beta1", framework.NamespaceName, "horizontalpodautoscalers", "")
	resp, err := httpHelper.RequestByUser(http.MethodPost, hpaUrl, postJson, user, nil)
	framework.ExpectNoError(err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	framework.ExpectNoError(err)
	clog.Info("get hap response, %v", string(body))

	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to create hpa for %s", deployNameWithUser), resp.StatusCode)
	}

	return framework.SucceedResp
}

func checkDeployHpa(user string) framework.TestResp {
	deploy := v1.Deployment{}
	err := wait.Poll(framework.WaitInterval, framework.WaitTimeout*2,
		func() (bool, error) {
			err := targetClient.Cache().Get(context.TODO(), types.NamespacedName{
				Name:      deployNameWithUser,
				Namespace: framework.NamespaceName,
			}, &deploy)
			if err != nil || deploy.Status.Replicas != int32(2) {
				return false, err
			} else {
				return true, nil
			}
		})
	framework.ExpectNoError(err)
	clog.Info("hpa deployment status: %v", deploy.Status)
	framework.ExpectEqual(deploy.Status.Replicas, int32(2))
	return framework.SucceedResp
}

func deleteDeployHpa(user string) framework.TestResp {
	hpaUrl := BuildK8sProxyUrl(framework.KubecubeHost, framework.TargetClusterName, "apis/autoscaling/v2beta1", framework.NamespaceName, "horizontalpodautoscalers", deployNameWithUser)
	resp, err := httpHelper.RequestByUser(http.MethodDelete, hpaUrl, "", user, nil)
	framework.ExpectNoError(err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	clog.Info("delete hpa: %+v", string(body))

	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to delete hpa for %s", deployNameWithUser), resp.StatusCode)
	}
	return framework.SucceedResp
}

func deleteDeploy(user string) framework.TestResp {
	deployUrl := BuildK8sProxyUrl(framework.KubecubeHost, framework.TargetClusterName, "apis/apps/v1", framework.NamespaceName, "deployments", deployNameWithUser)
	resp, err := httpHelper.RequestByUser(http.MethodDelete, deployUrl, "", user, nil)
	framework.ExpectNoError(err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	clog.Info("delete deploy: %+v", string(body))

	if !framework.IsSuccess(resp.StatusCode) {
		clog.Warn("res code %d", resp.StatusCode)
		return framework.NewTestResp(fmt.Errorf("fail to delete deploy %s", deployNameWithUser), resp.StatusCode)
	}

	if framework.PVEnabled {
		pv1Url := BuildK8sProxyUrl(framework.KubecubeHost, framework.TargetClusterName, "api/v1", framework.NamespaceName, "persistentvolumeclaims", pv1NameWithUser)
		resp, err = httpHelper.RequestByUser(http.MethodDelete, pv1Url, "", user, nil)
		framework.ExpectNoError(err)
		defer resp.Body.Close()
		body, err = io.ReadAll(resp.Body)
		clog.Info("delete pv1: %+v", string(body))

		if !framework.IsSuccess(resp.StatusCode) {
			clog.Warn("res code %d", resp.StatusCode)
			return framework.NewTestResp(fmt.Errorf("fail to delete pv1 %s", pv1NameWithUser), resp.StatusCode)
		}

		pv2Url := BuildK8sProxyUrl(framework.KubecubeHost, framework.TargetClusterName, "api/v1", framework.NamespaceName, "persistentvolumeclaims", pv2NameWithUser)
		resp, err = httpHelper.RequestByUser(http.MethodDelete, pv2Url, "", user, nil)
		framework.ExpectNoError(err)
		defer resp.Body.Close()
		body, err = io.ReadAll(resp.Body)
		clog.Info("delete pv2: %+v", string(body))

		if !framework.IsSuccess(resp.StatusCode) {
			clog.Warn("res code %d", resp.StatusCode)
			return framework.NewTestResp(fmt.Errorf("fail to delete pv2 %s", pv2NameWithUser), resp.StatusCode)
		}
	}

	return framework.SucceedResp
}

var multiUserDeployTest = framework.MultiUserTest{
	TestName:        "[工作负载][9386601]创建Deployment工作负载挂载卷",
	ContinueIfError: false,
	Skipfunc: func() bool {
		return !framework.DeploymentEnable
	},
	ErrorFunc:  framework.PermissionErrorFunc,
	AfterEach:  nil,
	BeforeEach: nil,
	InitStep:   nil,
	FinalStep:  nil,
	Steps: []framework.MultiUserTestStep{
		{
			Name:        "创建工作负载",
			Description: "创建工作负载",
			StepFunc:    createDeploy,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
		{
			Name:        "工作负载最终创建成功",
			Description: "工作负载最终创建成功",
			StepFunc:    checkDeploy,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
		{
			Name:        "查看工作负载》日志一直输出 hello",
			Description: "查看工作负载》日志一直输出 hello",
			StepFunc:    checkDeployLog,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
		{
			Name:        "容器内查看pvc1 挂载到了/mnt1/目录；/mnt2/目录无法进行写入（只读共享类型）",
			Description: "容器内查看pvc1 挂载到了/mnt1/目录；/mnt2/目录无法进行写入（只读共享类型）",
			StepFunc:    checkDeployPv,
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
			StepFunc:    checkDeployInfo,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
		{
			Name:        "都可以查看到准确的对应信息",
			Description: "查看副本基本信息\n3、查看副本事件\n4、查看容器日志",
			StepFunc:    checkDeployDetails,
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
			StepFunc:    checkDeployPerformance,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
		{
			Name:        "Deployment和副本的conditions与k8s查询一致",
			Description: "查看Deployment和副本的condition详情与k8s的是否一致",
			StepFunc:    checkDeployConditions,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
		{
			Name:        "Deployment事件前端页面和后台k8s命令显示一致件",
			Description: "查看Deployment事件",
			StepFunc:    checkDeployEvents,
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
			StepFunc:    checkDeployPodEvents,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
		{
			Name:        "查看工作负载更新成功，配置生效",
			Description: "更新负载配置",
			StepFunc:    updateDeployConfig,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
		{
			Name:        "查看最终副本详情生成4个副本，副本运行正常",
			Description: "更新副本个数为4",
			StepFunc:    updateDeployReplica,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
		{
			Name:        "调整负载副本数",
			Description: "更新副本个数为1",
			StepFunc:    resetDeployReplica,
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
			StepFunc:    setDeployHpa,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
		{
			Name:     "容器副本数扩容到2",
			StepFunc: checkDeployHpa,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
		{
			Name:     "clean hpa",
			StepFunc: deleteDeployHpa,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
		{
			Name:     "clean deploy",
			StepFunc: deleteDeploy,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
	},
}

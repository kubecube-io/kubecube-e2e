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

package e2e

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	userv1 "github.com/kubecube-io/kubecube/pkg/apis/user/v1"
	"github.com/kubecube-io/kubecube/pkg/clients"
	"github.com/kubecube-io/kubecube/pkg/clog"
	"github.com/kubecube-io/kubecube/pkg/utils/constants"
	"github.com/kubecube-io/kubecube/pkg/utils/md5util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubecube-io/kubecube-e2e/e2e/framework"
)

var (
	TenantId      int64
	ProjectId     int64
	scheme        = runtime.NewScheme()
	tenantHeader  = make(map[string][]string)
	projectHeader = make(map[string][]string)
)

var isMaster bool

func RunE2ETests(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "E2e Suite")
}

// InitAll 初始化参数
func InitAll() error {
	// init client-go client
	clients.InitCubeClientSetWithOpts(nil)
	// Read config and init global v
	err := framework.InitGlobalV()
	if err != nil {
		clog.Debug(err.Error())
		return err
	}

	err = framework.InitMultiConfig()
	if err != nil {
		clog.Debug(err.Error())
		return err
	}
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	return nil
}

// Start 执行 e2e 测试的前置步骤
func Start() error {
	if !isMaster {
		return waitUntilResourceInited()
	}

	clearResources()
	err := initializeResources()
	if err != nil {
		markAllResourceInitFailed()
		return err
	}

	markAllResourceInited()
	return nil
}

// End 清理测试数据
func End() error {
	if !isMaster {
		markAllTestInThisWorkerFinished()
		return nil
	} else {
		waitUntilTestsInAllWorkersFinished()
	}

	err := clearResources()
	if err != nil {
		return err
	}

	markAllResourceCleared()
	return nil
}

func Clear() error {
	// init client-go client
	clients.InitCubeClientSetWithOpts(nil)

	err := loadConfigFromCm()
	if err != nil {
		clog.Error(err.Error())
		return err
	}

	// Read config and init global v
	err = framework.InitGlobalV()
	if err != nil {
		clog.Error(err.Error())
		return err
	}
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	err = clearResources()
	if err != nil {
		return err
	}

	return clearTempResources()
}

func deleteUserInKubecube(ctx context.Context, cli client.Client, namespace string, username string) error {
	// 解除公共权限绑定
	clusterRoleBinding := rbacv1.ClusterRoleBinding{}
	clusterRoleBinding.Name = framework.TenantAdmin + "-in-cluster"
	clog.Info("[clusterRoleBinding] delete clusterRoleBinding %v", clusterRoleBinding.Name)
	err := cli.Delete(ctx, &clusterRoleBinding)
	if err != nil && !kerrors.IsNotFound(err) {
		clog.Debug("delete clusterRoleBinding fail, %v", err)
		return err
	}

	// 解除专用权限绑定
	roleBinding := rbacv1.RoleBinding{}
	roleBinding.Name = framework.TenantAdmin + "-in-" + namespace
	roleBinding.Namespace = namespace
	clog.Info("[roleBinding] delete roleBinding %v, namespace %v", roleBinding.Name, roleBinding.Namespace)
	err = cli.Delete(ctx, &roleBinding)
	if err != nil && !kerrors.IsNotFound(err) {
		clog.Debug("delete roleBinding fail, %v", err)
		return err
	}

	// 删除用户
	user := userv1.User{}
	user.Name = username
	err = cli.Delete(ctx, &user)
	if err != nil && !kerrors.IsNotFound(err) {
		clog.Debug("delete user fail, %v", err)
		return err
	}
	return nil
}

func waitUntilResourceInited() error {
	cm := &v1.ConfigMap{}
	err := wait.Poll(framework.WaitInterval, time.Minute*10, func() (done bool, err error) {
		err = framework.PivotClusterClient.Direct().Get(context.Background(), types.NamespacedName{
			Namespace: framework.KubeCubeSystem,
			Name:      framework.KubeCubeE2ECM,
		}, cm)
		if err != nil {
			clog.Error("fail to test resource by cm due to %s", err.Error())
			return false, err
		}

		if cm.Labels[framework.ResourceFailed] == "true" {
			clog.Warn("resource failed")
			return false, errors.New("resourceFailed")
		}

		if cm.Labels[framework.ResourceReady] == "true" {
			clog.Info("resource ready")
			return true, nil
		}

		clog.Info("resource not ready, waiting")
		return false, nil
	})
	if err != nil {
		return err
	}

	cm = &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      getCmName(),
			Namespace: framework.KubeCubeSystem,
			Labels: map[string]string{
				framework.KubeCubeE2ECM: framework.KubeCubeE2ECM,
			},
		},
	}
	err = framework.PivotClusterClient.Direct().Delete(context.Background(), cm)
	if err != nil && !kerrors.IsNotFound(err) {
		clog.Error("fail to clear cm with kubecube-e2e-config label when worker is going to test due to %s", err.Error())
		return err
	}
	err = framework.PivotClusterClient.Direct().Create(context.Background(), cm)
	if err != nil {
		clog.Error("fail to create cm with kubecube-e2e-config label when worker is going to test due to %s", err.Error())
		return err
	}

	clog.Info("resource initialized, worker is going to test")

	return nil
}

func markAllResourceInited() {
	cm := &v1.ConfigMap{}
	err := framework.PivotClusterClient.Direct().Get(context.Background(), types.NamespacedName{
		Namespace: framework.KubeCubeSystem,
		Name:      framework.KubeCubeE2ECM,
	}, cm)
	if err != nil {
		clog.Error("fail to get kubecube-e2e-config label when markAllResourceInited due to %s", err.Error())
		return
	}

	if len(cm.Labels) == 0 {
		cm.Labels = make(map[string]string)
	}
	cm.Labels[framework.ResourceReady] = "true"
	cm.Labels[framework.ResourceFailed] = "false"
	err = framework.PivotClusterClient.Direct().Update(context.Background(), cm)
	if err != nil {
		clog.Error("fail to update kubecube-e2e-config label when markAllResourceInited due to %s", err.Error())
		return
	}

	clog.Info("master initialized all resources, master is going to test")
}

func markAllResourceInitFailed() {
	cm := &v1.ConfigMap{}
	err := framework.PivotClusterClient.Direct().Get(context.Background(), types.NamespacedName{
		Namespace: framework.KubeCubeSystem,
		Name:      framework.KubeCubeE2ECM,
	}, cm)
	if err != nil {
		clog.Error("fail to get kubecube-e2e-config label when markAllResourceInitFailed due to %s", err.Error())
		return
	}

	if len(cm.Labels) == 0 {
		cm.Labels = make(map[string]string)
	}
	cm.Labels[framework.ResourceFailed] = "true"
	err = framework.PivotClusterClient.Direct().Update(context.Background(), cm)
	if err != nil {
		clog.Error("fail to update kubecube-e2e-config label when markAllResourceInitFailed due to %s", err.Error())
		return
	}
}

func waitUntilTestsInAllWorkersFinished() {
	cmList := &v1.ConfigMapList{}
	err := wait.Poll(framework.WaitInterval, time.Minute*20, func() (done bool, err error) {
		err = framework.PivotClusterClient.Direct().List(context.Background(), cmList,
			client.InNamespace(framework.KubeCubeSystem),
			client.MatchingLabels{
				framework.KubeCubeE2ECM: framework.KubeCubeE2ECM,
			})

		if err != nil {
			clog.Warn("fail to watch cm list by labels due to %s", err.Error())
			return false, nil
		}

		if len(cmList.Items) == 0 {
			clog.Info("cm list with labels is empty")
			return true, nil
		}

		clog.Info("%d workers running, waiting", len(cmList.Items))
		return false, nil
	})
	if err != nil {
		clog.Error("error when waitUntilTestsInAllWorkersFinished, %s", err.Error())
	}

	clog.Info("all tests of workers finished, master is going to clean resources")
}

func markAllTestInThisWorkerFinished() {
	cm := &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      getCmName(),
			Namespace: framework.KubeCubeSystem,
			Labels: map[string]string{
				framework.KubeCubeE2ECM: framework.KubeCubeE2ECM,
			},
		},
	}

	err := framework.PivotClusterClient.Direct().Delete(context.Background(), cm)
	if err != nil && !kerrors.IsNotFound(err) {
		clog.Error("fail to delete cm with label when markAllTestInThisWorkerFinished")
		return
	}

	clog.Info("worker finished test and marked")
}

func markAllResourceCleared() {
	cm := &v1.ConfigMap{}
	err := framework.PivotClusterClient.Direct().Get(context.Background(), types.NamespacedName{
		Namespace: framework.KubeCubeSystem,
		Name:      framework.KubeCubeE2ECM,
	}, cm)
	if err != nil {
		clog.Error("fail to get kubecube-e2e-config label when markAllResourceCleared due to %s", err.Error())
		return
	}

	if len(cm.Labels) == 0 {
		cm.Labels = make(map[string]string)
	}
	cm.Labels[framework.ResourceReady] = "false"
	cm.Labels[framework.ResourceFailed] = "false"
	err = framework.PivotClusterClient.Direct().Update(context.Background(), cm)
	if err != nil {
		clog.Error("fail to update kubecube-e2e-config label when markAllResourceCleared due to %s", err.Error())
		return
	}

	clog.Info("master cleared all resources")
}

func getCmName() string {
	s := framework.KubeCubeE2ECM + "-" + strings.Join(framework.TestUser, "-")
	return strings.ToLower(s)
}

func loadConfigFromCm() error {
	localCli := clients.Interface().Kubernetes(constants.LocalCluster)
	if localCli == nil {
		return fmt.Errorf("get local client %v failed", constants.LocalCluster)
	}

	cm := &v1.ConfigMap{}
	err := localCli.Direct().Get(context.Background(), types.NamespacedName{
		Namespace: "kubecube-system",
		Name:      "kubecube-e2e-config",
	}, cm)
	if err != nil {
		clog.Error("fail to get cm kubecube-e2e-config due to %s", err.Error())
		return err
	}

	config := cm.Data["config"]

	current, err := os.Getwd()
	if err != nil {
		return err
	}
	path := current + "/config.yaml"
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0o666)
	if err != nil {
		clog.Error("fail to output config helper due to %s \n", err.Error())
		return err
	}

	defer file.Close()
	_, err = file.Write([]byte(config))
	if err != nil {
		clog.Error("fail to output config helper due to %s \n", err.Error())
		return err
	}
	return nil
}

func clearTempResources() error {
	current, err := os.Getwd()
	if err != nil {
		clog.Error("fail to get os wd due to %s", err.Error())
		return err
	}
	path := current + "/config.yaml"
	err = os.Remove(path)
	if err != nil {
		clog.Error("fail to delete file due to %s", err.Error())
		return err
	}
	clog.Info("%s deleted", path)

	cli := clients.Interface().Kubernetes(constants.LocalCluster)
	if cli == nil {
		clog.Error("get local client %v failed", constants.LocalCluster)
		return err
	}

	err = cli.Direct().DeleteAllOf(context.Background(), &v1.ConfigMap{}, client.InNamespace("kubecube-system"), client.MatchingLabels{"kubecube-e2e-config": "kubecube-e2e-config"})
	if err != nil && !kerrors.IsNotFound(err) {
		clog.Error("fail to delete worker cm due to %s", err.Error())
		return err
	}
	clog.Info("worker cm deleted")
	return nil
}

func createUser(accountId string, accountPassword string) error {
	user := &userv1.User{}
	err := framework.PivotClusterClient.Direct().Get(context.Background(), types.NamespacedName{Name: accountId}, user)
	if err != nil {
		if kerrors.IsNotFound(err) {
			user.Name = accountId
			user.Spec.Password = md5util.GetMD5Salt(accountPassword)
			user.Spec.LoginType = userv1.NormalLogin
			user.Spec.State = userv1.NormalState
			annotations := make(map[string]string)
			annotations["kubecube.io/sync"] = "true"
			user.Annotations = annotations
			user.Spec.DisplayName = accountId
			err = framework.PivotClusterClient.Direct().Create(context.Background(), user)
			return err
		} else {
			return err
		}
	}
	return nil
}

func createTenantAdminRoleBindings(username string, tenant string) error {
	ctx := context.Background()
	rolebindng := &rbacv1.RoleBinding{}
	rolebindng.Name = username + "-in-kubecube-tenant-" + tenant
	rolebindng.Namespace = "kubecube-tenant-" + tenant
	clusterrole := rbacv1.RoleRef{
		Kind: "ClusterRole",
		Name: constants.TenantAdmin,
	}
	user := rbacv1.Subject{
		Kind: "User",
		Name: username,
	}
	rolebindng.Subjects = []rbacv1.Subject{user}
	rolebindng.RoleRef = clusterrole
	annotations := make(map[string]string)
	annotations[constants.SyncAnnotation] = "true"
	rolebindng.Annotations = annotations
	labels := make(map[string]string)
	labels[constants.RbacLabel] = "true"
	labels[constants.TenantLabel] = tenant
	rolebindng.Labels = labels
	return framework.PivotClusterClient.Direct().Create(ctx, rolebindng)
}

func createProjectAdminRoleBindings(username string, tenant string, project string) error {
	ctx := context.Background()
	rolebindng := &rbacv1.RoleBinding{}
	rolebindng.Name = username + "-in-kubecube-project-" + project
	rolebindng.Namespace = "kubecube-project-" + project
	clusterrole := rbacv1.RoleRef{
		Kind: "ClusterRole",
		Name: constants.ProjectAdmin,
	}
	user := rbacv1.Subject{
		Kind: "User",
		Name: username,
	}
	rolebindng.Subjects = []rbacv1.Subject{user}
	rolebindng.RoleRef = clusterrole
	annotations := make(map[string]string)
	annotations[constants.SyncAnnotation] = "true"
	rolebindng.Annotations = annotations
	labels := make(map[string]string)
	labels[constants.RbacLabel] = "true"
	labels[constants.TenantLabel] = tenant
	labels[constants.ProjectLabel] = project
	rolebindng.Labels = labels
	return framework.PivotClusterClient.Direct().Create(ctx, rolebindng)
}

func createProjectViewerRoleBindings(username string, tenant string, project string) error {
	ctx := context.Background()
	rolebindng := &rbacv1.RoleBinding{}
	rolebindng.Name = username + "-in-kubecube-project-" + project
	rolebindng.Namespace = "kubecube-project-" + project
	clusterrole := rbacv1.RoleRef{
		Kind: "ClusterRole",
		Name: constants.Reviewer,
	}
	user := rbacv1.Subject{
		Kind: "User",
		Name: username,
	}
	rolebindng.Subjects = []rbacv1.Subject{user}
	rolebindng.RoleRef = clusterrole
	annotations := make(map[string]string)
	annotations[constants.SyncAnnotation] = "true"
	rolebindng.Annotations = annotations
	labels := make(map[string]string)
	labels[constants.RbacLabel] = "true"
	labels[constants.TenantLabel] = tenant
	labels[constants.ProjectLabel] = project
	rolebindng.Labels = labels
	return framework.PivotClusterClient.Direct().Create(ctx, rolebindng)
}

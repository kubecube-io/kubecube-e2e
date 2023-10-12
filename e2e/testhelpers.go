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
	"fmt"
	"strconv"

	"github.com/kubecube-io/kubecube/pkg/utils/constants"

	"github.com/kubecube-io/kubecube-e2e/e2e/framework"
	quotav1 "github.com/kubecube-io/kubecube/pkg/apis/quota/v1"
	v1 "github.com/kubecube-io/kubecube/pkg/apis/quota/v1"
	tenantv1 "github.com/kubecube-io/kubecube/pkg/apis/tenant/v1"
	"github.com/kubecube-io/kubecube/pkg/clog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

// initializeResources 执行 e2e 测试的前置资源创建
func initializeResources() error {
	clog.Info("Before Testing...")
	ctx := context.Background()
	// 1.创建租户: cube-e2e-tenant-1
	clog.Info("[Before] create test tenant %v", framework.TenantName)
	tenant := &tenantv1.Tenant{}
	err := framework.PivotClusterClient.Direct().Get(ctx, types.NamespacedName{Name: framework.TenantName}, tenant)
	if err != nil {
		if errors.IsNotFound(err) {
			tenant.Name = framework.TenantName
			tenant.Spec.DisplayName = framework.TenantName
			tenant.Spec.Description = framework.TenantName
			err = framework.PivotClusterClient.Direct().Create(ctx, tenant)
			if err != nil {
				clog.Error("[Before] create tenant err: %v", err.Error())
				return err
			}
		} else {
			clog.Error("[Before] create tenant err: %v", err.Error())
			return err
		}
	}
	// 2.创建项目: cube-e2e-project-1
	clog.Info("[Before] create test project %v", framework.ProjectName)
	project := &tenantv1.Project{}
	err = framework.PivotClusterClient.Direct().Get(ctx, types.NamespacedName{Name: framework.ProjectName}, project)
	if err != nil {
		if errors.IsNotFound(err) {
			project.Name = framework.ProjectName
			project.Spec.DisplayName = framework.ProjectName
			project.Spec.Description = framework.ProjectName
			labels := make(map[string]string)
			labels[constants.TenantLabel] = framework.TenantName
			project.Labels = labels
			err = framework.PivotClusterClient.Direct().Create(ctx, project)
			if err != nil {
				clog.Error("[Before] create project err: %v", err.Error())
				return err
			}
		} else {
			clog.Error("[Before] create project err: %v", err.Error())
			return err
		}
	}

	// 3.创建用户
	clog.Info("[Before] create user: %v, tenant user: %v, project user: %v", framework.User, framework.TenantAdmin, framework.ProjectAdmin)
	err = createUser(framework.TenantAdmin, framework.TenantAdminPassword)
	if err != nil && !errors.IsAlreadyExists(err) {
		clog.Info("[Before] create user %s in platform err: %v", framework.TenantAdmin, err)
		return err
	}
	err = createUser(framework.ProjectAdmin, framework.ProjectAdminPassword)
	if err != nil && !errors.IsAlreadyExists(err) {
		clog.Info("[Before] create user %s in platform err: %v", framework.ProjectAdmin, err)
		return err
	}
	err = createUser(framework.User, framework.UserPassword)
	if err != nil && !errors.IsAlreadyExists(err) {
		clog.Info("[Before] create user %s in platform err: %v", framework.User, err)
		return err
	}

	waitInterval := framework.WaitInterval
	waitTimeout := framework.WaitTimeout
	cli := framework.TargetClusterClient

	// 4.绑定角色
	// user1-租户管理员
	err = createTenantAdminRoleBindings(framework.TenantAdmin, framework.TenantName)
	if err != nil {
		clog.Info("[Before] bind tenantAdmin role err: %v", err)
		return err
	}
	// user2-项目管理员
	err = createProjectAdminRoleBindings(framework.ProjectAdmin, framework.TenantName, framework.ProjectName)
	if err != nil {
		clog.Info("[Before] bind projectAdmin role err: %v", err)
		return err
	}
	// user3-项目普通成员
	err = createProjectViewerRoleBindings(framework.ProjectAdmin, framework.TenantName, framework.ProjectName)
	if err != nil {
		clog.Info("[Before] bind user role err: %v", err)
		return err
	}

	// 6.创建租户配额
	clog.Info("[Before] create cube resource quota")
	tenantQuota := v1.CubeResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name: framework.CubeResourceQuota,
			Annotations: map[string]string{
				"kubecube.io/sync": "true",
			},
			Labels: map[string]string{
				constants.ClusterLabel:   framework.TargetClusterName,
				constants.CubeQuotaLabel: framework.TenantName,
			},
		},
		Spec: v1.CubeResourceQuotaSpec{
			Hard: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceRequestsCPU:     resource.MustParse(strconv.Itoa(10)),
				corev1.ResourceLimitsCPU:       resource.MustParse(strconv.Itoa(10)),
				corev1.ResourceRequestsMemory:  resource.MustParse(strconv.Itoa(10) + "Gi"),
				corev1.ResourceLimitsMemory:    resource.MustParse(strconv.Itoa(10) + "Gi"),
				corev1.ResourceRequestsStorage: resource.MustParse(strconv.Itoa(30) + "Gi"),
			},
			Target: v1.TargetObj{
				Name: framework.TenantName,
				Kind: "Tenant",
			},
		},
	}
	err = framework.PivotClusterClient.Direct().Create(ctx, &tenantQuota)
	if err != nil && !errors.IsAlreadyExists(err) {
		clog.Info("[Before] can not create cube resource quota: %v", err)
		return err
	}

	// 7.目标集群会创建一个测试用命名空间
	clog.Info("[Before] create namespace %v in cluster %v", framework.NamespaceName, framework.TargetClusterName)
	subNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: framework.NamespaceName,
			Labels: map[string]string{
				constants.TenantLabel:        framework.TenantName,
				constants.ProjectLabel:       framework.ProjectName,
				constants.HncIncludedNsLabel: "true",
				fmt.Sprintf("%v%v.tree.hnc.x-k8s.io/depth", constants.ProjectNsPrefix, framework.ProjectName): "1",
				fmt.Sprintf("%v%v.tree.hnc.x-k8s.io/depth", constants.TenantNsPrefix, framework.TenantName):   "2",
				fmt.Sprintf("%v.tree.hnc.x-k8s.io/depth", framework.NamespaceName):                            "0",
				"node.kubecube.io/ns": "share",
				"system/namespace":    "netease.share",
				fmt.Sprintf("system/project-%v", framework.ProjectName): "true",
				"system/tenant": framework.TenantName,
			},
			Annotations: map[string]string{
				constants.HncAnnotation:      constants.ProjectNsPrefix + framework.ProjectName,
				constants.HncIncludedNsLabel: "true",
			},
		},
	}
	if err := cli.Direct().Create(ctx, subNs); err != nil && !errors.IsAlreadyExists(err) {
		clog.Debug("[Before] e2e init fail, can not create e2e namespace in %s, %v", framework.TargetClusterName, err)
		return err
	}

	// 8.创建空间 ResourceQuota
	if err := wait.Poll(waitInterval, waitTimeout,
		func() (bool, error) {
			var tenantQuota v1.CubeResourceQuota
			errInfo := cli.Cache().Get(ctx, types.NamespacedName{
				Name: framework.CubeResourceQuota,
			}, &tenantQuota)
			if errInfo != nil {
				return false, errInfo
			} else {
				return true, errInfo
			}
		}); err != nil {
		clog.Info("[Before] e2e init fail, can not find tenant resource quota in %s, %v", framework.TargetClusterName, err)
		return err
	}
	nsQuota := corev1.ResourceQuota{}
	err = framework.TargetClusterClient.Cache().Get(context.TODO(), types.NamespacedName{Namespace: framework.NamespaceName, Name: framework.TargetClusterName + "." + framework.TenantName + "." + framework.ProjectName + "." + framework.NamespaceName}, &nsQuota)
	if errors.IsNotFound(err) {
		nsQuota = corev1.ResourceQuota{
			ObjectMeta: metav1.ObjectMeta{
				Name:      framework.TargetClusterName + "." + framework.TenantName + "." + framework.ProjectName + "." + framework.NamespaceName,
				Namespace: framework.NamespaceName,
				Labels: map[string]string{
					constants.ClusterLabel:   framework.TargetClusterName,
					constants.ProjectLabel:   framework.ProjectName,
					constants.CubeQuotaLabel: framework.CubeResourceQuota,
					constants.TenantLabel:    framework.TenantName,
				},
			},
			Spec: corev1.ResourceQuotaSpec{
				Hard: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceRequestsCPU:     resource.MustParse(strconv.Itoa(2)),
					corev1.ResourceLimitsCPU:       resource.MustParse(strconv.Itoa(2)),
					corev1.ResourceRequestsMemory:  resource.MustParse(strconv.Itoa(4) + "Gi"),
					corev1.ResourceLimitsMemory:    resource.MustParse(strconv.Itoa(4) + "Gi"),
					corev1.ResourceRequestsStorage: resource.MustParse(strconv.Itoa(30) + "Gi"),
				},
			},
		}
		err = framework.TargetClusterClient.Direct().Create(context.TODO(), &nsQuota)
		if err != nil && !errors.IsAlreadyExists(err) {
			clog.Info("[Before] e2e init fail, can not create namespace resource quota in %s, %v", framework.TargetClusterName, err)
			return err
		}
	}
	// 9.create ns in pivot cluster
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: framework.NamespaceName,
		},
	}
	err = framework.PivotClusterClient.Direct().Create(ctx, ns)
	if err != nil && !errors.IsAlreadyExists(err) {
		clog.Info("create ns in pivot cluster error: %s", err.Error())
		return err
	}
	// 10.create image pull secret
	err = framework.CreateSecret()
	if err != nil {
		return err
	}

	return nil
}

// clearResources 清理测试数据
func clearResources() error {
	ctx := context.Background()
	waitInterval := framework.WaitInterval
	waitTimeout := framework.WaitTimeout
	tenant := framework.TenantName
	project := framework.ProjectName
	tenantNamespace := "kubecube-tenant-" + tenant
	projectNamespace := "kubecube-project-" + project
	pivotCli := framework.PivotClusterClient
	targetCli := framework.TargetClusterClient
	clog.Info("After testing...")

	// 1.删除空间
	clog.Info("[After] delete e2e namespace %v", framework.NamespaceName)
	sns := corev1.Namespace{}
	sns.Name = framework.NamespaceName
	err := targetCli.Direct().Delete(ctx, &sns)
	if err != nil {
		if !errors.IsNotFound(err) {
			clog.Debug("[After] delete e2e namespace fail, %v", err)
			return err
		}
	}

	// 2.删除用户
	clog.Info("[After] delete user")
	err = deleteUserInKubecube(ctx, pivotCli.Direct(), tenantNamespace, framework.TenantAdmin)
	if err != nil {
		return err
	}
	err = deleteUserInKubecube(ctx, pivotCli.Direct(), projectNamespace, framework.ProjectAdmin)
	if err != nil {
		return err
	}
	err = deleteUserInKubecube(ctx, pivotCli.Direct(), projectNamespace, framework.User)
	if err != nil {
		return err
	}

	// 3.删除项目
	clog.Info("[After] delete project %v", framework.ProjectName)
	p := tenantv1.Project{}
	p.Name = framework.ProjectName
	err = pivotCli.Direct().Delete(ctx, &p)
	if err != nil && !errors.IsNotFound(err) {
		clog.Info("[After] delete project fail, %v", err)
		return err
	}

	// 4.删除租户配额
	clog.Info("[After] delete tenant resource quota")
	cubeQuota := quotav1.CubeResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name: framework.CubeResourceQuota,
		},
	}
	err = pivotCli.Direct().Delete(ctx, &cubeQuota)
	if err != nil {
		if !errors.IsNotFound(err) {
			clog.Info("[After] delete tenant resource quota fail: %v", err)
			return err
		}
	}

	// 5.删除租户
	clog.Info("[After] delete tenant %v", framework.TenantName)
	t := tenantv1.Tenant{}
	t.Name = framework.TenantName
	err = pivotCli.Direct().Delete(ctx, &t)
	if err != nil && !errors.IsNotFound(err) {
		clog.Info("[After] delete tenant fail, %v", err)
		return err
	}

	// 6. 删除 namespace resource quota
	clog.Info("[After] delete namespace resource quota %v", framework.TargetClusterName+"."+framework.TenantName+"."+framework.ProjectName+"."+framework.NamespaceName)
	nsQuota := corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      framework.TargetClusterName + "." + framework.TenantName + "." + framework.ProjectName + "." + framework.NamespaceName,
			Namespace: framework.NamespaceName,
		},
	}
	err = targetCli.Direct().Delete(ctx, &nsQuota)
	if err != nil {
		if !errors.IsNotFound(err) {
			clog.Info("[After] delete namespace resource quota fail, %v", err)
			return err
		}
	} else {
		if err = wait.Poll(waitInterval, waitTimeout,
			func() (bool, error) {
				var nsQuota corev1.Namespace
				err := pivotCli.Direct().Get(ctx, types.NamespacedName{
					Name:      framework.TargetClusterName + "." + framework.TenantName + "." + framework.ProjectName + "." + framework.NamespaceName,
					Namespace: framework.NamespaceName,
				}, &nsQuota)
				if err != nil {
					if errors.IsNotFound(err) {
						return true, nil
					}
					return false, nil
				}
				return false, nil
			}); err != nil {
			clog.Info("[After] delete namespace resource quota timeout, %v", err)
			return err
		}
	}
	// 7.delete pivot ns
	var pivotNamespace corev1.Namespace
	pivotNamespace.Name = framework.NamespaceName
	err = framework.PivotClusterClient.Direct().Delete(ctx, &pivotNamespace)
	if err != nil {
		return err
	}
	err = wait.Poll(waitInterval, waitTimeout,
		func() (bool, error) {
			var namespace corev1.Namespace
			errInfo := framework.PivotClusterClient.Direct().Get(ctx, types.NamespacedName{Name: framework.NamespaceName}, &namespace)
			if errors.IsNotFound(errInfo) {
				return true, nil
			} else {
				return false, nil
			}
		})
	if err != nil {
		return err
	}
	return nil
}

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

package framework

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	e2econstants "github.com/kubecube-io/kubecube-e2e/util/constants"
	v1 "github.com/kubecube-io/kubecube/pkg/apis/cluster/v1"
	"github.com/kubecube-io/kubecube/pkg/clients"
	"github.com/kubecube-io/kubecube/pkg/clog"
	"github.com/kubecube-io/kubecube/pkg/multicluster"
	"github.com/kubecube-io/kubecube/pkg/multicluster/client"
	"github.com/kubecube-io/kubecube/pkg/utils/constants"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// cluster
	TargetClusterName string
	PivotClusterName  string
	// quota
	CubeResourceQuota string
	// host
	KubecubeHost string
	ConsoleHost  string
	// tenant
	TenantName    string
	ProjectName   string
	NamespaceName string
	// user
	Admin                string
	AdminPassword        string
	TenantAdmin          string
	TenantAdminPassword  string
	ProjectAdmin         string
	ProjectAdminPassword string
	User                 string
	UserPassword         string
	// timeout
	WaitInterval       time.Duration
	WaitTimeout        time.Duration
	HttpRequestTimeout time.Duration
	// pv
	PVEnabled bool
	// CloudShellEnabled cloudShell
	CloudShellEnabled bool
	// workload
	CronJobEnable     bool
	DaemonSetEnable   bool
	DeploymentEnable  bool
	JobEnable         bool
	StatefulSetEnable bool
	NodeHostName      string
	NodeHostIp        string
	TestImage         string
	ImagePullSecret   string
	StorageClass      string
	// hub
	Registry string
	Username string
	Password string
	Email    string
	// PivotClusterClient communicate with pivot cluster
	PivotClusterClient client.Client

	// TargetClusterClient communicate with target cluster
	TargetClusterClient client.Client

	KubeCubeSystem string
	KubeCubeE2ECM  string
	LoginType      string
)

// InitGlobalV 初始化全局变量
func InitGlobalV() error {
	err := readEnvConfig()
	if err != nil {
		return err
	}
	clogConfig := clog.Config{
		LogFile:         "/etc/logs/cube.log",
		MaxSize:         1000,
		MaxBackups:      7,
		MaxAge:          1,
		Compress:        true,
		LogLevel:        "debug",
		JsonEncode:      false,
		StacktraceLevel: "error",
	}
	v := viper.Sub("Log")
	if v != nil {
		err = v.Unmarshal(&clogConfig)
		if err != nil {
			return err
		}
	}
	clog.InitCubeLoggerWithOpts(&clogConfig)

	// host
	KubecubeHost = viper.GetString("host.kubecubeHost")
	ConsoleHost = viper.GetString("host.consoleHost")
	// cluster
	TargetClusterName = viper.GetString("e2eInit.targetCluster")
	PivotClusterName = viper.GetString("e2eInit.pivotCluster")
	// tenant
	TenantName = viper.GetString("e2eInit.tenant")
	ProjectName = viper.GetString("e2eInit.project")
	NamespaceName = viper.GetString("e2eInit.namespace")
	// user
	Admin = viper.GetString("e2eInit.multiuser.admin")
	AdminPassword = viper.GetString("e2eInit.multiuser.adminPassword")
	TenantAdmin = viper.GetString("e2eInit.multiuser.tenantAdmin")
	TenantAdminPassword = viper.GetString("e2eInit.multiuser.tenantAdminPassword")
	ProjectAdmin = viper.GetString("e2eInit.multiuser.projectAdmin")
	ProjectAdminPassword = viper.GetString("e2eInit.multiuser.projectAdminPassword")
	User = viper.GetString("e2eInit.multiuser.user")
	UserPassword = viper.GetString("e2eInit.multiuser.userPassword")
	// timeout
	WaitInterval = time.Duration(viper.GetInt("timeout.waitInterval")) * time.Second
	WaitTimeout = time.Duration(viper.GetInt("timeout.waitTimeout")) * time.Second
	HttpRequestTimeout = time.Duration(viper.GetInt("timeout.httpRequestTimeout")) * time.Second
	// pv
	PVEnabled = viper.GetBool("pv.enabled")

	CubeResourceQuota = TargetClusterName + "." + TenantName

	CloudShellEnabled = viper.GetBool("cloudshell.enabled")
	// workload
	CronJobEnable = viper.GetBool("workload.cronjob")
	DaemonSetEnable = viper.GetBool("workload.daemonSet")
	DeploymentEnable = viper.GetBool("workload.deployment")
	JobEnable = viper.GetBool("workload.job")
	StatefulSetEnable = viper.GetBool("workload.statefulSet")
	NodeHostName = viper.GetString("workload.nodeHostName")
	NodeHostIp = viper.GetString("workload.nodeHostIp")
	TestImage = viper.GetString("image.testImage")
	StorageClass = viper.GetString("workload.storageClass")
	if TestImage == "" {
		return fmt.Errorf("test image value can not be empty")
	}
	if StorageClass == "" {
		StorageClass = "localstorage-class"
	}
	ImagePullSecret = "harbor-qingzhou"
	Registry = viper.GetString("hub.registry")
	Username = viper.GetString("hub.username")
	Password = viper.GetString("hub.password")
	Email = viper.GetString("hub.email")

	KubeCubeSystem = viper.GetString("sys.namespace")
	KubeCubeE2ECM = viper.GetString("sys.cm-name")
	LoginType = viper.GetString("sys.login-type")
	if len(LoginType) == 0 {
		LoginType = e2econstants.GeneralLoginType
	}

	localCli := clients.Interface().Kubernetes(constants.LocalCluster)
	if localCli == nil {
		return fmt.Errorf("get local client %v failed", constants.LocalCluster)
	}

	clusters := v1.ClusterList{}

	err = localCli.Direct().List(context.Background(), &clusters)
	if err != nil {
		return fmt.Errorf("list clusters failed %v", err)
	}

	for _, v := range clusters.Items {
		err = multicluster.AddInternalCluster(v)
		if err != nil {
			return fmt.Errorf("add cluster %v failed", v.Name)
		}
	}

	cli := clients.Interface().Kubernetes(PivotClusterName)
	if cli == nil {
		return fmt.Errorf("get pivot client %v failed", PivotClusterName)
	}
	PivotClusterClient = cli

	cli2 := clients.Interface().Kubernetes(TargetClusterName)
	if cli2 == nil {
		return fmt.Errorf("get tatget client %v failed", TargetClusterClient)
	}
	TargetClusterClient = cli2

	return nil
}

// readEnvConfig read params from config
func readEnvConfig() error {
	// todo commond line
	cfgFile := ""
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		current, err := os.Getwd()
		if err != nil {
			return err
		}
		viper.AddConfigPath(current)
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}
	viper.SetEnvPrefix("kubecube")
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	return nil
}

func CreateSecret() error {
	auth := Username + ":" + Password
	auth = base64.StdEncoding.EncodeToString([]byte(auth))
	data := `{"auths":{"%s":{"username":"%s","password":"%s","email":"%s","auth":"%s"}}}`
	data = fmt.Sprintf(data, Registry, Username, Password, Email, auth)
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ImagePullSecret,
			Namespace: NamespaceName,
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: []byte(data),
		},
	}
	err := TargetClusterClient.Direct().Create(context.TODO(), &secret)
	if err != nil && !errors.IsAlreadyExists(err) {
		clog.Error("create secret in target cluster fail, secret: %+v, err: %s", secret, err.Error())
		return err
	}
	secret = corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ImagePullSecret,
			Namespace: NamespaceName,
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: []byte(data),
		},
	}
	err = PivotClusterClient.Direct().Create(context.TODO(), &secret)
	if err != nil && !errors.IsAlreadyExists(err) {
		clog.Error("create secret in pivot cluster fail, secret: %+v, err: %s", secret, err.Error())
		return err
	}
	return nil
}

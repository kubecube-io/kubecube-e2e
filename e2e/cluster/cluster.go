package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/kubecube-io/kubecube-e2e/e2e/framework"
	v1 "github.com/kubecube-io/kubecube/pkg/apis/cluster/v1"
	"github.com/kubecube-io/kubecube/pkg/clog"
	"github.com/kubecube-io/kubecube/pkg/multicluster/client"
)

var (
	pivotClient client.Client
	httpHelper  *framework.HttpHelper
)

func init() {
	framework.RegisterByDefault(multiUserTest)
}

func initParam() {
	pivotClient = framework.PivotClusterClient
	httpHelper = framework.NewSingleHttpHelper()
}

var multiUserTest = framework.MultiUserTest{
	TestName:        "[集群信息]集群列表检查检查",
	ContinueIfError: false,
	SkipUsers:       []string{},
	Skipfunc:        nil,
	ErrorFunc:       framework.PermissionErrorFunc,
	AfterEach:       nil,
	BeforeEach:      nil,
	InitStep:        nil,
	FinalStep:       nil,
	Steps: []framework.MultiUserTestStep{
		{
			Name:        "集群列表检查",
			Description: "1. 通过接口获取集群列表, 并和集群信息进行比较，查看数量和名称是否相符",
			StepFunc:    listCluster,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
	},
}

func listCluster(user string) framework.TestResp {
	initParam()
	urlOfListCluster := "%s/api/v1/cube/clusters/info"
	urlOfListCluster = fmt.Sprintf(urlOfListCluster, framework.KubecubeHost)
	resp, err := httpHelper.RequestByUser(http.MethodGet, urlOfListCluster, "", user, nil)
	framework.ExpectNoError(err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	framework.ExpectNoError(err)
	clog.Debug("get cluster list resp, data: %s", string(body))
	var clusterResMap map[string]interface{}
	err = json.Unmarshal(body, &clusterResMap)
	framework.ExpectNoError(err)
	clusterList := v1.ClusterList{}
	err = pivotClient.Direct().List(context.Background(), &clusterList)
	framework.ExpectNoError(err)
	total, ok := clusterResMap["total"]
	framework.ExpectEqual(ok, true)
	framework.ExpectEqual(float64(len(clusterList.Items)), total)
	items, ok := clusterResMap["items"].([]interface{})
	framework.ExpectEqual(ok, true)
	for _, item := range items {
		item, ok := item.(map[string]interface{})
		framework.ExpectEqual(ok, true)
		name, ok := item["clusterName"].(string)
		framework.ExpectEqual(ok, true)
		nameCheck := false
		for _, cluster := range clusterList.Items {
			if name == cluster.Name {
				nameCheck = true
				break
			}
		}
		framework.ExpectEqual(nameCheck, true)
	}
	return framework.SucceedResp
}

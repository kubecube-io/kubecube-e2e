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

package node

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/kubecube-io/kubecube-e2e/e2e/framework"
	"github.com/kubecube-io/kubecube/pkg/clog"
	"github.com/kubecube-io/kubecube/pkg/multicluster/client"

	v1 "k8s.io/api/core/v1"
)

var (
	pivotClient  client.Client
	pivotCluster string
	httpHelper   *framework.HttpHelper
)

func init() {
	framework.RegisterByDefault(multiUserTest)
}

func initParam() {
	pivotClient = framework.PivotClusterClient
	pivotCluster = framework.PivotClusterName
	httpHelper = framework.NewSingleHttpHelper()
}

var multiUserTest = framework.MultiUserTest{
	TestName:        "[节点信息]集群节点列表检查检查",
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
			Name:        "管控集群节点检查",
			Description: "1. 通过接口获取管控集群节点列表, 并和节点信息进行比较，查看数量和名称是否相符",
			StepFunc:    listNode,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       true,
			},
		},
	},
}

func listNode(user string) framework.TestResp {
	initParam()
	urlOfListNode := "%s/api/v1/cube/extend/clusters/%s/resources/nodes"
	urlOfListNode = fmt.Sprintf(urlOfListNode, framework.KubecubeHost, pivotCluster)
	resp, err := httpHelper.RequestByUser(http.MethodGet, urlOfListNode, "", user, nil)
	framework.ExpectNoError(err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	framework.ExpectNoError(err)
	clog.Info("get node list resp, data: %s", string(body))
	var nodeResMap map[string]interface{}
	err = json.Unmarshal(body, &nodeResMap)
	framework.ExpectNoError(err)
	nodeList := v1.NodeList{}
	err = pivotClient.Direct().List(context.Background(), &nodeList)
	framework.ExpectNoError(err)
	total, ok := nodeResMap["total"]
	framework.ExpectEqual(ok, true)
	framework.ExpectEqual(float64(len(nodeList.Items)), total)
	items, ok := nodeResMap["items"].([]interface{})
	framework.ExpectEqual(ok, true)
	for _, item := range items {
		item, ok := item.(map[string]interface{})
		framework.ExpectEqual(ok, true)
		metadata, ok := item["metadata"].(map[string]interface{})
		framework.ExpectEqual(ok, true)
		name, ok := metadata["name"].(string)
		framework.ExpectEqual(ok, true)
		nameCheck := false
		for _, node := range nodeList.Items {
			if name == node.Name {
				nameCheck = true
				break
			}
		}
		framework.ExpectEqual(nameCheck, true)
	}
	return framework.SucceedResp
}

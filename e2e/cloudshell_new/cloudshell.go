package cloudshell_new

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	backoff2 "github.com/kubecube-io/kubecube-e2e/util/retry"
	websocket2 "github.com/kubecube-io/kubecube-e2e/util/websocket"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kubecube-io/kubecube/pkg/clog"
	"github.com/onsi/ginkgo"
	v1 "k8s.io/api/core/v1"

	"github.com/kubecube-io/kubecube-e2e/e2e/framework"
)

func checkCloudShell(user string) framework.TestResp {
	httpHelper := framework.NewSingleHttpHelper()
	nodeList := &v1.NodeList{}
	err := framework.TargetClusterClient.Cache().List(context.Background(), nodeList)
	framework.ExpectNoError(err)
	framework.ExpectNotEqual(len(nodeList.Items), 0)
	var nodename string
	for _, node := range nodeList.Items {
		for _, condition := range node.Status.Conditions {
			if condition.Type == v1.NodeReady && condition.Status == "True" {
				nodename = node.Name
				break
			}
		}
	}
	stop := make(chan struct{}, 1)
	ginkgo.By("请求接口api/v1/extends/cloudShell/clusters/{clusterName}获取sessionId")
	sessionUrl := fmt.Sprintf("%s/api/v1/extends/cloudShell/clusters/%s", framework.ConsoleHost, framework.TargetClusterName)
	response, err := httpHelper.RequestByUser(http.MethodGet, sessionUrl, "", user, nil)
	framework.ExpectNoError(err)
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	clog.Info("session msg: %s", body)
	framework.ExpectNoError(err)

	if !framework.IsSuccess(response.StatusCode) {
		clog.Warn("res code %d", response.StatusCode)
		return framework.NewTestResp(errors.New("fail to get cloudshell sessionId"), response.StatusCode)
	}

	var sessionInfo map[string]interface{}
	err = json.Unmarshal(body, &sessionInfo)
	framework.ExpectNoError(err)
	sessionId, ok := sessionInfo["id"].(string)
	framework.ExpectEqual(ok, true)
	client, err := websocket2.NewClient(framework.ConsoleHost+"/api/sockjs", sessionId, stop)
	framework.ExpectNoError(err)
	message, err := websocket2.GetOpData("kubectl get node " + nodename + " \r").GetWriteMessage()
	framework.ExpectNoError(err)
	err = client.WriteMessage([]string{message})
	framework.ExpectNoError(err)
	b := &backoff2.ExponentialBackOff{
		InitialInterval:     framework.WaitInterval,
		RandomizationFactor: backoff2.DefaultRandomizationFactor,
		Multiplier:          backoff2.DefaultMultiplier,
		MaxInterval:         backoff2.DefaultMaxInterval,
		MaxElapsedTime:      framework.WaitTimeout,
		Clock:               backoff2.SystemClock,
	}
	b.Reset()
	ctx := context.Background()
	timeoutCtx, cancelFunc := context.WithTimeout(ctx, time.Second*framework.WaitTimeout)
	defer cancelFunc()
	err = backoff2.Retry(func() error {
		var res []string
		err = client.ReadMessage(&res)
		if err != nil {
			return err
		}
		if len(res) == 1 {
			re := &websocket2.Data{}
			err = json.Unmarshal([]byte(res[0]), re)
			if err != nil {
				return err
			}
			if strings.Contains(re.Data, string(v1.NodeReady)) {
				return nil
			}
		}
		return err
	}, b, timeoutCtx)
	framework.ExpectNoError(err)
	return framework.SucceedResp
}

var multiUserTest = framework.MultiUserTest{
	TestName:        "CloudShell检查",
	ContinueIfError: false,
	Skipfunc: func() bool {
		return !framework.CloudShellEnabled
	},
	ErrorFunc:  framework.PermissionErrorFunc,
	AfterEach:  nil,
	BeforeEach: nil,
	InitStep:   nil,
	FinalStep:  nil,
	Steps: []framework.MultiUserTestStep{
		{
			Name:        "CloudShell检查",
			Description: "CloudShell检查",
			StepFunc:    checkCloudShell,
			ExpectPass: map[string]bool{
				framework.UserAdmin:        true,
				framework.UserTenantAdmin:  true,
				framework.UserProjectAdmin: true,
				framework.UserNormal:       false,
			},
		},
	},
}

func init() {
	framework.RegisterByDefault(multiUserTest)
}

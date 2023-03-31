package http

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kubecube-io/kubecube-e2e/e2e/framework"
	"github.com/kubecube-io/kubecube-e2e/util/constants"
	"github.com/kubecube-io/kubecube/pkg/clog"
)

var client = http.DefaultClient

func init() {
	Register(constants.GeneralLoginType, GeneralLogin)
}

func GeneralLogin(user *AuthUser) error {
	postBody := map[string]string{
		"name":      user.Username,
		"password":  user.Password,
		"loginType": "normal",
	}
	postBodyJson, err := json.Marshal(postBody)
	if err != nil {
		clog.Error("login fail, marshal post body fail, %v", err)
		return err
	}
	url := fmt.Sprintf("%s/%s", framework.KubecubeHost, "/api/v1/cube/login")
	req, err := BuildRequest(http.MethodPost, url, string(postBodyJson), nil)
	if err != nil {
		clog.Error("login fail, error: %v", err)
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		clog.Error("login fail, error: %v", err)
		return err
	}
	cookies := resp.Cookies()
	if len(cookies) < 1 {
		clog.Error("get cookie error")
		return fmt.Errorf("get cookie error")
	}
	user.Cookie = cookies[0]
	return nil
}

package framework

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kubecube-io/kubecube-e2e/util/constants"
	"github.com/kubecube-io/kubecube/pkg/clog"
)

var (
	httpclient   = http.DefaultClient
	generalLogin = &GeneralLogin{}
)

type GeneralLogin struct {
}

func init() {
	Register(constants.GeneralLoginType, generalLogin)
}

func (g *GeneralLogin) AuthHeader() string {
	return constants.AuthorizationHeader
}
func (g *GeneralLogin) LoginByUser(user *AuthUser) error {
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
	url := fmt.Sprintf("%s/%s", KubecubeHost, "/api/v1/cube/login")
	req, err := BuildRequest(http.MethodPost, url, string(postBodyJson), nil)
	if err != nil {
		clog.Error("login fail, error: %v", err)
		return err
	}
	resp, err := httpclient.Do(req)
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

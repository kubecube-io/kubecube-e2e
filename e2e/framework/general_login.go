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

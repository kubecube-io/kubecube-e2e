package framework

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kubecube-io/kubecube/pkg/clog"
)

type AuthUser struct {
	Username string
	Password string
	Cookie   *http.Cookie
}

type HttpHelper struct {
	Admin        AuthUser
	TenantAdmin  AuthUser
	ProjectAdmin AuthUser
	User         AuthUser
	Client       http.Client
}

var httphelper *HttpHelper

// single mode
func NewSingleHttpHelper() *HttpHelper {
	if httphelper != nil {
		return httphelper
	}

	httphelper = NewHttpHelper().Login()
	return httphelper
}

func NewHttpHelper() *HttpHelper {

	h := &HttpHelper{
		Admin:        AuthUser{Username: Admin, Password: AdminPassword},
		TenantAdmin:  AuthUser{Username: TenantAdmin, Password: TenantAdminPassword},
		ProjectAdmin: AuthUser{Username: ProjectAdmin, Password: ProjectAdminPassword},
		User:         AuthUser{Username: User, Password: UserPassword},
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	h.Client = http.Client{
		Transport: tr,
		Timeout:   30 * time.Second,
	}
	return h
}

func (h *HttpHelper) Login() *HttpHelper {
	h.LoginByUser(&h.Admin)
	h.LoginByUser(&h.TenantAdmin)
	h.LoginByUser(&h.ProjectAdmin)
	h.LoginByUser(&h.User)
	return h
}

func (h *HttpHelper) LoginByUser(user *AuthUser) error {
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
	resp, err := h.Request("POST", url, string(postBodyJson), nil)
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

// get
func (h *HttpHelper) Get(urlVal string, header map[string]string) (*http.Response, error) {
	return h.RequestByUser(http.MethodGet, urlVal, "", Admin, header)
}

// post
func (h *HttpHelper) Post(urlVal, body string, header map[string]string) (*http.Response, error) {
	return h.RequestByUser(http.MethodPost, urlVal, body, Admin, header)
}

// delete
func (h *HttpHelper) Delete(urlVal string) (*http.Response, error) {
	return h.RequestByUser(http.MethodDelete, urlVal, "", Admin, nil)
}

// put
func (h *HttpHelper) Put(urlVal, body string, header map[string]string) (*http.Response, error) {
	return h.RequestByUser(http.MethodPut, urlVal, body, Admin, header)
}

func (h *HttpHelper) Patch(urlVal, body string, header map[string]string) (*http.Response, error) {
	return h.RequestByUser(http.MethodPatch, urlVal, body, Admin, header)
}

// default request by admin
func (h *HttpHelper) Request(method, urlVal, data string, header map[string]string) (*http.Response, error) {
	return h.RequestByUser(method, urlVal, data, Admin, header)
}

// request by user
func (h *HttpHelper) RequestByUser(method, urlVal, data, user string, header map[string]string) (*http.Response, error) {

	req, err := h.BuildRequest(method, urlVal, data, user, header)
	if err != nil {
		return nil, err
	}
	return h.Client.Do(req)
}

// build request
func (h *HttpHelper) BuildRequest(method, urlVal, data, user string, header map[string]string) (*http.Request, error) {
	var req *http.Request
	var err error

	urlArr := strings.Split(urlVal, "?")
	if len(urlArr) == 2 {
		urlVal = urlArr[0] + "?" + url.PathEscape(urlArr[1])
	}
	if data == "" {
		req, err = http.NewRequest(method, urlVal, nil)
	} else {
		req, err = http.NewRequest(method, urlVal, strings.NewReader(data))
	}
	if err != nil {
		return nil, err
	}

	switch user {
	case Admin:
		req.AddCookie(h.Admin.Cookie)
	case TenantAdmin:
		req.AddCookie(h.TenantAdmin.Cookie)
	case ProjectAdmin:
		req.AddCookie(h.ProjectAdmin.Cookie)
	case User:
		req.AddCookie(h.User.Cookie)
	}

	if _, ok := header["Content-Type"]; !ok {
		req.Header.Add("Content-Type", "application/json")
	}
	for k, v := range header {
		req.Header.Add(k, v)
	}

	return req, nil
}

func IsSuccess(code int) bool {
	return code >= 200 && code <= 299
}

type MultiRequestResponse struct {
	Resp *http.Response
	Err  error
}

// multi user request test
func (h *HttpHelper) MultiUserRequest(method, url, body string, header map[string]string) map[string]MultiRequestResponse {
	ret := make(map[string]MultiRequestResponse)

	resp1, err1 := h.RequestByUser(method, url, body, Admin, header)
	ret["admin"] = MultiRequestResponse{resp1, err1}

	resp2, err2 := h.RequestByUser(method, url, body, TenantAdmin, header)
	ret["tenantAdmin"] = MultiRequestResponse{resp2, err2}

	resp3, err3 := h.RequestByUser(method, url, body, ProjectAdmin, header)
	ret["projectAdmin"] = MultiRequestResponse{resp3, err3}

	resp4, err4 := h.RequestByUser(method, url, body, User, header)
	ret["user"] = MultiRequestResponse{resp4, err4}

	return ret
}

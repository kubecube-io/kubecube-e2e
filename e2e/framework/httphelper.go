package framework

import (
	"crypto/tls"
	"net/http"
	"sync"
	"time"

	"github.com/kubecube-io/kubecube/pkg/clog"
)

type HttpHelper struct {
	Admin        AuthUser
	TenantAdmin  AuthUser
	ProjectAdmin AuthUser
	User         AuthUser
	Client       http.Client
}

var httphelper *HttpHelper
var once sync.Once

// single mode
func NewSingleHttpHelper() *HttpHelper {
	if httphelper != nil {
		return httphelper
	}
	once.Do(func() {
		httphelper = NewHttpHelper().Login(LoginType)
	})
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

func (h *HttpHelper) Login(login string) *HttpHelper {
	loginFunc := GetLoginMap(login)
	_ = loginFunc(&h.Admin)
	_ = loginFunc(&h.TenantAdmin)
	_ = loginFunc(&h.ProjectAdmin)
	_ = loginFunc(&h.User)
	return h
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
	req, err := BuildRequest(method, urlVal, data, header)
	if err != nil {
		clog.Warn("build request error: %v", err.Error())
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

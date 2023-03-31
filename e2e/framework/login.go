package framework

import (
	"net/http"
	"net/url"
	"strings"
)

type AuthUser struct {
	Username string
	Password string
	Token    string
	Cookie   *http.Cookie
}

type LoginByUser func(user *AuthUser) error

var loginMap = make(map[string]LoginByUser)

func Register(key string, loginFunc LoginByUser) {
	loginMap[key] = loginFunc
}

func GetLoginMap(key string) LoginByUser {
	return loginMap[key]
}

func BuildRequest(method, urlVal, data string, header map[string]string) (*http.Request, error) {
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

	if _, ok := header["Content-Type"]; !ok {
		req.Header.Add("Content-Type", "application/json")
	}
	for k, v := range header {
		req.Header.Add(k, v)
	}

	return req, nil
}

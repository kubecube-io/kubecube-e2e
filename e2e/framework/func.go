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
	"github.com/kubecube-io/kubecube/pkg/clog"
)

type TestResp struct {
	Err  error
	Data interface{}
}

type TestFunc func(user string) TestResp

var (
	SucceedResp = TestResp{
		Err:  nil,
		Data: "",
	}
)

func NewTestResp(err error, data interface{}) TestResp {
	return TestResp{
		Err:  err,
		Data: data,
	}
}

func NewTestRespWithErr(err error) TestResp {
	if err == nil {
		return SucceedResp
	}

	return TestResp{
		Err:  err,
		Data: err.Error(),
	}
}

func DefaultErrorFunc(resp TestResp) {
	if resp.Err != nil {
		clog.Debug(resp.Err.Error())
	}
	ExpectError(resp.Err, "should be error")
}

func DefaultSkipFunc() bool {
	return false
}

func PermissionErrorFunc(resp TestResp) {
	clog.Debug("res code %v", resp)
	ExpectEqual(resp.Data, 403)
}

func NameWithUser(name, user string) string {
	return name + "-" + user
}

func RegisterByDefault(test MultiUserTest) {
	err := RegisterTestAndSteps(test)
	if err != nil {
		clog.Error("fail to register test steps with err %s \n", err.Error())
		return
	}

	err = CreateTestExample(test)
	if err != nil {
		clog.Error("fail to create tests with err %s \n", err.Error())
		return
	}
}

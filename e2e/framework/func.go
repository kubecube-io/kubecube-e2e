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

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
	"errors"
	"os"

	e2econstants "github.com/kubecube-io/kubecube-e2e/util/constants"
	"github.com/kubecube-io/kubecube/pkg/clog"
	"github.com/onsi/ginkgo"
	"gopkg.in/yaml.v3"
)

var AllTestMap = make(map[string]struct{})

var ConfigHelper = make([]MultiUserTest, 0)

func RegisterTestAndSteps(test MultiUserTest) error {
	_, ok := AllTestMap[test.TestName]
	if ok {
		clog.Warn("test %s exists", test.TestName)
		return errors.New("test already exists")
	}

	ConfigHelper = append(ConfigHelper, test)

	funcMap := make(map[string]struct{})
	for _, step := range test.Steps {
		if step.StepFunc == nil {
			return errors.New("test func is nil")
		}
		if _, ok := funcMap[step.Name]; ok {
			clog.Warn("test %s has duplicated step %s", test.TestName, step.Name)
			return errors.New("step duplicated")
		}
		funcMap[step.Name] = struct{}{}
	}

	AllTestMap[test.TestName] = struct{}{}

	return nil
}

func CreateTestExample(test MultiUserTest) error {
	_, ok := AllTestMap[test.TestName]
	if !ok {
		return errors.New("test not exist")
	}

	if test.ErrorFunc == nil {
		test.ErrorFunc = DefaultErrorFunc
	}

	if test.Skipfunc == nil {
		test.Skipfunc = DefaultSkipFunc
	}

	for _, user := range GetAllUsersAvailable() {
		generateSingleUserTestExample(test, test.ErrorFunc, user, test.BeforeEach, test.AfterEach, test.Skipfunc)
	}

	return nil
}

func OutputMultiUserTestConfig(path string) error {
	clog.Info("out put config helper")
	m := MultiUserTestConfig{
		TestMap:  ConfigHelper,
		AllUsers: []string{UserAdmin, UserProjectAdmin, UserTenantAdmin, UserNormal},
	}
	out, err := yaml.Marshal(m)
	if err != nil {
		clog.Error("fail to output config helper due to %s \n", err.Error())
		return err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0o666)
	if err != nil {
		clog.Error("fail to output config helper due to %s \n", err.Error())
		return err
	}

	defer file.Close()
	_, err = file.Write(out)
	if err != nil {
		clog.Error("fail to output config helper due to %s \n", err.Error())
		return err
	}
	return nil
}

func generateSingleUserTestExample(test MultiUserTest, errorFunc func(resp TestResp), user string, beforeEach, afterEach func(), skipFunc func() bool) {
	_ = ginkgo.Describe(test.TestName, func() {
		key := user + "-" + test.TestName
		testExampleConfiguredByUser, ok := ToTestMap[test.TestName]
		if !ok {
			clog.Info("no test named detected %s", test.TestName)
			return
		}

		if !UserContains(user) {
			clog.Info("user %s is not in runAs list", user)
			return
		}

		// get configmap
		// get run result: jobUid+user+funcName
		// if exist and result is pass, skip it
		// if not exist or result is not pass, run it
		// defer func() {
		// if err != nil , write to configmap pass, else write to configmap fail
		//}
		val, ok := TestResultInstance.Get(key)
		if ok && val == e2econstants.ConfigMapTestPassValue {
			clog.Info("test %s is passed, skip it", test.TestName)
			return
		}
		TestResultInstance.Remove(key)
		if skipFunc() {
			clog.Info("test is skipped by skip func %s", test.TestName)
			return
		}

		userTestFromConfig := TestConfig[test.TestName]
		if contains(userTestFromConfig.SkipUsers, user) {
			clog.Info("user %s is skipped in test %s", user, test.TestName)
			return
		}

		flag := false

		ginkgo.BeforeEach(func() {
			if !flag && beforeEach != nil {
				beforeEach()
			}
		})

		ginkgo.AfterEach(func() {
			if !flag && afterEach != nil {
				afterEach()
			}
		})

		ginkgo.Context("测试用例", func() {
			getUser := GetUser(user)
			setTestResult := func(pass bool) {
				if pass {
					// get test result
					// if test result not exist, write pass
					// if test result exist, do not write result, avoid overwrite previous pass or fail result
					_, ok := TestResultInstance.Get(key)
					if !ok {
						TestResultInstance.Set(key, e2econstants.ConfigMapTestPassValue)
					}
				} else {
					TestResultInstance.Set(key, e2econstants.ConfigMapTestFailValue)
				}
			}
			exec := func(step MultiUserTestStep, testFunc TestFunc) TestResp {
				finish := false
				var resp TestResp
				defer func() {
					expect := false
					if val, ok := testExampleConfiguredByUser[step.Name]; ok {
						expect = val.ExpectPass[user]
					} else {
						// it is init step or final step
						expect = true
					}
					if ginkgo.CurrentGinkgoTestDescription().Failed {
						clog.Info("test failed, test is %s" + user + "-" + test.TestName)
						setTestResult(!expect)
						return
					}
					if !finish {
						clog.Info("test is not finish, that means it failed, test is %s" + user + "-" + test.TestName)
						setTestResult(!expect)
						return
					}
					if resp.Err != nil {
						clog.Info("test res has error, that means it failed, test is %s" + user + "-" + test.TestName)
						setTestResult(!expect)
						return
					}
					clog.Info("test %s is passed", user+"-"+test.TestName)
					setTestResult(expect)
				}()
				resp = testFunc(getUser)
				// if test use Expect and fail, it will be panic, that means finish wili be false, and test is failed
				finish = true
				return resp
			}
			if test.InitStep != nil {
				step := test.InitStep
				ginkgo.It(user+" : "+step.Name, func() {
					if len(step.Description) > 0 {
						ginkgo.By(step.Description)
					}
					clog.Info("running init step as %s \n", getUser)
					exec(*step, step.StepFunc)
				})
			}

			for _, s := range test.Steps {
				step := s
				ginkgo.It(user+" : "+step.Name, func() {
					if flag {
						clog.Info("step before failed, skipping remaining steps")
						ginkgo.Skip("ignore")
					}

					if len(step.Description) > 0 {
						ginkgo.By(step.Description)
					}

					testFunc := step.StepFunc
					clog.Info("running %s step %s as %s", test.TestName, step.Name, getUser)
					resp := exec(step, testFunc)
					if !userTestFromConfig.ContinueIfError && resp.Err != nil {
						flag = true
					}

					if !testExampleConfiguredByUser[step.Name].ExpectPass[user] {
						flag = true
						errorFunc(resp)
					} else {
						ExpectNoError(resp.Err)
					}
				})

			}

			if test.FinalStep != nil {
				step := test.FinalStep
				ginkgo.It(user+" : "+step.Name, func() {
					if len(step.Description) > 0 {
						ginkgo.By(step.Description)
					}
					clog.Info("running final step as %s \n", getUser)
					exec(*step, step.StepFunc)
				})
			}
		})
	})
}

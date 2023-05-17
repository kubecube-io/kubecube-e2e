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
	"os"

	"github.com/kubecube-io/kubecube/pkg/clog"
	"gopkg.in/yaml.v3"
)

var (
	ToTestMap  = make(map[string]map[string]MultiUserTestStep)
	TestUser   []string
	TestConfig = make(map[string]MultiUserTest)
	config     MultiUserTestConfig
)

type MultiUserTest struct {
	TestName        string              `yaml:"testName"`
	ContinueIfError bool                `yaml:"continueIfError"`
	Steps           []MultiUserTestStep `yaml:"steps"`
	SkipUsers       []string            `yaml:"skipUsers"`
	BeforeEach      func()              `yaml:"-"`
	AfterEach       func()              `yaml:"-"`
	Skipfunc        func() bool         `yaml:"-"`
	ErrorFunc       func(resp TestResp) `yaml:"-"`
	InitStep        *MultiUserTestStep  `yaml:"-"`
	FinalStep       *MultiUserTestStep  `yaml:"-"`
}

type MultiUserTestStep struct {
	Name        string          `yaml:"name"`
	Description string          `yaml:"description"`
	StepFunc    TestFunc        `yaml:"-"`
	ExpectPass  map[string]bool `yaml:"expectPass,omitempty"`
}

type MultiUserTestConfig struct {
	AllUsers []string        `yaml:"allUsers"`
	TestMap  []MultiUserTest `yaml:"testMap"`
}

func InitMultiConfig() error {
	err := loadConfig()
	if err != nil {
		clog.Error(err.Error())
		return err
	}

	tests := config.TestMap
	if len(tests) > 0 {
		for _, test := range tests {
			m := make(map[string]MultiUserTestStep)
			for _, step := range test.Steps {
				m[step.Name] = step
			}
			ToTestMap[test.TestName] = m
			TestConfig[test.TestName] = test
		}
	}
	return nil
}

func readConfig() ([]byte, error) {
	current, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	filePath := current + "/" + MultiConfig
	_, err = os.Stat(filePath)
	if err != nil {
		clog.Info("config empty")
		return []byte{}, nil
	}

	clog.Info("load from file %s", filePath)
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func loadConfig() error {
	bytes, err := readConfig()
	if err != nil {
		clog.Error(err.Error())
		return err
	}

	err = yaml.Unmarshal(bytes, &config)
	if err != nil {
		clog.Error(err.Error())
		return err
	}

	return nil
}

func UserContains(user string) bool {
	return contains(TestUser, user)
}

func GetAllUsersAvailable() []string {
	if len(config.AllUsers) == 0 {
		err := loadConfig()
		if err != nil {
			clog.Error(err.Error())
		}
	}

	if len(config.AllUsers) == 0 {
		config.AllUsers = []string{UserAdmin, UserProjectAdmin, UserTenantAdmin, UserNormal}
	}

	return config.AllUsers
}

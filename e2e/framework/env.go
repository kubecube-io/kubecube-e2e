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
	"sync"

	"github.com/kubecube-io/kubecube/pkg/clog"
)

var controllerUid string
var controllerLock sync.Once
var jobName string
var jobLock sync.Once

func GetControllerUid() string {
	controllerLock.Do(func() {
		controllerUid = os.Getenv("controller-uid")
		if len(controllerUid) == 0 {
			clog.Fatal("controller-uid can not be empty")
		}
	})
	return controllerUid
}

func GetJobName() string {
	jobLock.Do(func() {
		jobName = os.Getenv("job-name")
		if len(jobName) == 0 {
			jobName = "kubecube-e2e-test"
		}
	})
	return jobName
}

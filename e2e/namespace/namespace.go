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

package namespace

import . "github.com/onsi/ginkgo"

var _ = Describe("[空间管理][9382715]空间管理", func() {
	// init v here

	// before condition
	BeforeEach(func() {})

	// after clean action
	AfterEach(func() {})

	Context("[创建空间-计算资源共享]", func() {
		It("应该能够创建成功", func() {
			By("1.创建共享空间并设置共享资源")
			// todo
			By("2.查看租户下空间信息")
		})
	})
})

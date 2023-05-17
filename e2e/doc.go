/*
// template 1
var _ = Describe("[module_name][module_id]module_info", func() {
	// init v here

	// before condition
	BeforeEach(func() {})

	// after clean action
	AfterEach(func() {})

	Context("[tc_name1]", func() {
		It("assertion", func() {
			By("1.step-one")

			By("2.step-two")
		})
	})

	Context("[tc_name2]", func() {
		It("assertion", func() {
			By("1.step-one")

			By("2.step-two")
		})
	})
})

// template 2
var _ = Describe("[module_name][tc_id]tc_name", func() {
	// init v here

	// before condition
	BeforeEach(func() {})

	// after clean action
	AfterEach(func() {})

	Context("[tc_name]context1", func() {
		It("assertion", func() {
			By("1.step-one")

			By("2.step-two")
		})
	})

	Context("[tc_name]context2", func() {
		It("assertion", func() {
			By("1.step-one")

			By("2.step-two")
		})
	})
})

*/

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

package e2e // Package e2e copy comment as template

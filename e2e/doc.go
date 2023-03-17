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

package e2e // Package e2e copy comment as template

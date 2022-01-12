package command_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	. "github.com/platform9/nodelet/pkg/utils/command"
)

func TestCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit.xml")
	//RunSpecs(t, "Command Suite")
	RunSpecsWithDefaultAndCustomReporters(t, "Command Suite", []Reporter{junitReporter})
}

var _ = Describe("Command", func() {
	var (
		cmdObj   CLI
		path     string
		args     []string
		cwd      string
		err      error
		env      map[string]string
		tmo      int
		exitCode int
		ctx      context.Context
	)

	Describe("Without Timeout", func() {
		Context("With Valid Input Parameters", func() {
			BeforeEach(func() {
				env = map[string]string{
					"TEST_ENV": "test",
				}
				cwd = "/go/src/github.com/platform9/nodelet/pkg/utils/command"
				path = "ls"
				args = []string{"-l", "-r", "-t", "-h"}
				cmdObj = New()
				tmo = -1
				ctx = context.TODO()
			})

			AfterEach(func() {
				ctx.Done()
			})

			It("Should run without error if path exists", func() {
				exitCode, err = cmdObj.RunCommand(ctx, env, tmo, cwd, path, args...)
				Expect(exitCode).To(Equal(0))
				Expect(err).To(BeNil())
			})
			It("Should run without error if no args provided", func() {
				exitCode, err = cmdObj.RunCommand(ctx, env, tmo, cwd, path)
				Expect(exitCode).To(Equal(0))
				Expect(err).To(BeNil())
			})
			It("Should fail with error if path doesn't exist", func() {
				exitCode, err = cmdObj.RunCommand(ctx, env, tmo, cwd, "/path/to/nowhere", args...)
				//fmt.Println(err.Error())
				Expect(exitCode).To(Equal(-1))
				Expect(err).NotTo(BeNil())
			})
			It("Should accept environment variables and use them", func() {
				exitCode, stdout, err := cmdObj.RunCommandWithStdOut(ctx, env, tmo, cwd, "sh", "-c", "env|grep TEST_ENV")
				Expect(err).To(BeNil())
				Expect(exitCode).To(Equal(0))
				Expect(stdout[0]).To(Equal("TEST_ENV=test"))
			})
			It("Should return exit status of the command", func() {
				exitCode, stdout, err := cmdObj.RunCommandWithStdOut(ctx, env, tmo, cwd, "sh", "-c", "env|grep TEST1_ENV")
				Expect(err).NotTo(BeNil())
				Expect(exitCode).To(Equal(1))
				Expect(len(stdout)).To(Equal(0))
			})
			It("Should not timeout if timeout is 0 seconds", func() {
				_, _, _, err := cmdObj.RunCommandWithStdOutStdErr(ctx, env, 0, cwd, "sleep", "5")
				Expect(err).To(BeNil())
			})
			It("Should read from STDOUT", func() {
				_, stdout, err := cmdObj.RunCommandWithStdOut(ctx, env, tmo, cwd, "sh", "-c", "echo OUT>&1")
				Expect(err).To(BeNil())
				Expect(stdout[0]).To(Equal("OUT"))
			})
			It("Should read from STDERR", func() {
				_, stderr, err := cmdObj.RunCommandWithStdErr(ctx, env, tmo, cwd, "sh", "-c", "echo ERR>&2")
				Expect(err).To(BeNil())
				Expect(stderr[0]).To(Equal("ERR"))
			})
			It("Should read from STDOUT and STDERR", func() {
				_, stdout, stderr, err := cmdObj.RunCommandWithStdOutStdErr(ctx, env, tmo, cwd, "sh", "-c", "echo OUT>&1; echo ERR>&2")
				Expect(err).To(BeNil())
				Expect(stdout[0]).To(Equal("OUT"))
				Expect(stderr[0]).To(Equal("ERR"))
			})
		})

		Context("With Invalid Input Parameters", func() {
			It("Should not create command object", func() {
			})
		})
	})

	Describe("With Timeout", func() {
		Context("With Valid Input Parameters", func() {
			BeforeEach(func() {
				env = map[string]string{
					"TEST_ENV": "test",
				}
				cwd = "/go/src/github.com/platform9/nodelet/pkg/utils/command"
				path = "ls"
				args = []string{"-l", "-r", "-t", "-h"}
				cmdObj = New()
				tmo = -1
				ctx = context.TODO()
			})

			AfterEach(func() {
				ctx.Done()
			})

			It("Should honour", func() {})
		})
	})
})

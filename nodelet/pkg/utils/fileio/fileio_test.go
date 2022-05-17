package fileio_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/reporters"
	. "github.com/onsi/gomega"
	. "github.com/platform9/nodelet/nodelet/pkg/utils/fileio"

	"github.com/platform9/nodelet/nodelet/pkg/utils/command"
)

func TestCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("junit.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Fileio Suite", []Reporter{junitReporter})
}

var _ = Describe("Fileio", func() {
	var (
		origFilename   string
		filename       string
		err            error
		content        string
		invalidContent string
		sliceContent   []string
		byteContent    []byte
		lines          []string
		byteData       []byte
		jsonData       map[string]interface{}
		fileInpOut     FileInterface
		cmdLine        command.CLI
		ctx            context.Context
	)

	Describe("File I/O", func() {
		Context("Creating a file", func() {
			BeforeEach(func() {
				origFilename = "/tmp/create_file/file1_orig"
				//dupFilename = "/tmp/create_file/file1_copy"
				//filename = "/tmp/create_file/test_file"
				fileInpOut = New()
				cmdLine = command.New()
				ctx = context.TODO()
				_, _ = cmdLine.RunCommand(ctx, nil, 0, "", "mkdir", "-p", "/tmp/create_file")
			})

			AfterEach(func() {
				_, _ = cmdLine.RunCommand(ctx, nil, 0, "", "rm", "-rf", "/tmp/create_file")
				ctx.Done()
			})

			It("Should create a file without error", func() {
				fmt.Printf("Creating file ...\n")
				err = fileInpOut.TouchFile(origFilename)
				Expect(err).To(BeNil())
				_, _, err = cmdLine.RunCommandWithStdOut(ctx, nil, 0, "/tmp/create_file", "sh", "-c", "ls -ltr | grep file1_orig")
				Expect(err).To(BeNil())
			})

			It("Should fail to create file within non existent directory", func() {
				fmt.Printf("Creating file in non existent directory\n")
				err = fileInpOut.TouchFile("/abcd" + origFilename)
				fmt.Printf("Error: %s\n", err.Error())
				Expect(err).NotTo(BeNil())
			})
		})

		Context("Deleting a file", func() {
			BeforeEach(func() {
				filename = "/tmp/delete_file/file1"
				fileInpOut = New()
				cmdLine = command.New()
				ctx = context.TODO()
				_, _ = cmdLine.RunCommand(ctx, nil, 0, "", "mkdir", "-p", "/tmp/delete_file")
			})

			AfterEach(func() {
				_, _ = cmdLine.RunCommand(ctx, nil, 0, "", "rm", "-rf", "/tmp/delete_file")
				ctx.Done()
			})

			It("Should delete a file", func() {
				fmt.Printf("Creating file using command line ...\n")
				_, _, _, err = cmdLine.RunCommandWithStdOutStdErr(ctx, nil, 0, "/tmp/delete_file", "sh", "-c", "touch file1")
				Expect(err).To(BeNil())
				fmt.Printf("Deleting file ...\n")
				err = fileInpOut.DeleteFile(filename)
				Expect(err).To(BeNil())
				_, _, _, err = cmdLine.RunCommandWithStdOutStdErr(ctx, nil, 0, "/tmp/delete_file", "sh", "-c", "ls -ltr | grep file1")
				fmt.Printf("Error: %s\n", err.Error())
				Expect(err).NotTo(BeNil())
			})

			It("Should fail to delete non-existent file", func() {
				fmt.Printf("Deleting non-existent file ...\n")
				err = fileInpOut.DeleteFile("/abcd" + filename)
				fmt.Printf("Error: %s\n", err.Error())
				Expect(err).NotTo(BeNil())
			})
		})

		Context("Fetching file details", func() {
			BeforeEach(func() {
				filename = "/tmp/file_details/file1"
				fileInpOut = New()
				cmdLine = command.New()
				ctx = context.TODO()
				_, _ = cmdLine.RunCommand(ctx, nil, 0, "", "mkdir", "-p", "/tmp/file_details")
			})

			AfterEach(func() {
				_, _ = cmdLine.RunCommand(ctx, nil, 0, "", "rm", "-rf", "/tmp/file_details")
				ctx.Done()
			})

			It("Should fetch file details", func() {
				fmt.Printf("Creating file ...\n")
				err = fileInpOut.TouchFile(filename)
				Expect(err).To(BeNil())
				fmt.Printf("Fetching file details ...\n")
				info, err := fileInpOut.GetFileInfo(filename)
				Expect(err).To(BeNil())
				fmt.Println("File Details:")
				fmt.Println("Name: ", info.Name())
				fmt.Println("Last Modified: ", info.ModTime())
				fmt.Println("IsDirectory: ", info.IsDir())
				fmt.Println("Permissions: ", info.Mode())
				fmt.Println("Size(in bytes): ", info.Size())
				fmt.Printf("System interface type: %T\n", info.Sys())
				fmt.Printf("System info: %+v\n\n", info.Sys())
				Expect(info).NotTo(BeNil())
			})

			It("Should fail to fetch file details of non existing directory", func() {
				fmt.Printf("Fetching file details ...\n")
				_, err = fileInpOut.GetFileInfo("/abcd" + filename)
				fmt.Printf("Error: %s\n", err.Error())
				Expect(err).NotTo(BeNil())
			})

		})

		Context("Reading a Plaintext file", func() {
			BeforeEach(func() {
				filename = "/tmp/read_file/file1"
				content = "This is not a test!"
				fileInpOut = New()
				cmdLine = command.New()
				ctx = context.TODO()
				_, _ = cmdLine.RunCommand(ctx, nil, 0, "", "mkdir", "-p", "/tmp/read_file")
				_, _, _, _ = cmdLine.RunCommandWithStdOutStdErr(ctx, nil, 0, "/tmp/read_file", "ls", "-ltr")
				err := ioutil.WriteFile(filename, []byte(content), 0644)
				Expect(err).To(BeNil())
			})

			AfterEach(func() {
				_, _ = cmdLine.RunCommand(ctx, nil, 0, "", "rm", "-rf", "/tmp/read_file")
				ctx.Done()
			})

			It("Should read file line by line", func() {
				lines, err = fileInpOut.ReadFileByLine(filename)
				Expect(err).To(BeNil())
				Expect(lines).To(Equal([]string{content}))
			})

			It("Should read file all at once", func() {
				byteData, err = fileInpOut.ReadFile(filename)
				Expect(err).To(BeNil())
				Expect(string(byteData)).To(Equal(content))
			})

			It("Should fail to read non-existent file line by line", func() {
				lines, err = fileInpOut.ReadFileByLine("/abcd" + filename)
				Expect(err).NotTo(BeNil())
			})

			It("Should fail to read non-existent file all at once", func() {
				byteData, err = fileInpOut.ReadFile("/abcd" + filename)
				Expect(err).NotTo(BeNil())
			})
		})

		Context("Reading from JSON file", func() {
			BeforeEach(func() {
				filename = "/tmp/read_json_file/file1.json"
				content = "{\"key1\": \"value1\",\"key2\":\"value2\",\"key3\":{\"key4\":\"value4\"}}"
				invalidContent = "\"key1\": value1\",\"key2\":value2\",\"key3\":{\"key4\":\"value4\"}}"
				fileInpOut = New()
				cmdLine = command.New()
				jsonData = map[string]interface{}{}
				ctx = context.TODO()
				fmt.Printf("Creating directory ...\n")
				_, _ = cmdLine.RunCommand(ctx, nil, 0, "", "mkdir", "-p", "/tmp/read_json_file")
				_, _, _, _ = cmdLine.RunCommandWithStdOutStdErr(ctx, nil, 0, "/tmp/read_json_file", "ls", "-ltr")
				err := ioutil.WriteFile(filename, []byte(content), 0644)
				Expect(err).To(BeNil())
			})

			AfterEach(func() {
				_, _ = cmdLine.RunCommand(ctx, nil, 0, "", "rm", "-rf", "/tmp/read_json_file")
				ctx.Done()
			})

			It("Should read a valid JSON file", func() {
				err = fileInpOut.ReadJSONFile(filename, &jsonData)
				Expect(err).To(BeNil())
				Expect(jsonData["key1"]).To(Equal("value1"))
				Expect(jsonData["key2"]).To(Equal("value2"))
			})

			It("Should fail to read non-existent JSON file", func() {
				err = fileInpOut.ReadJSONFile("/abcd"+filename, &jsonData)
				Expect(err).NotTo(BeNil())
			})

			It("Should fail to read invalid JSON file", func() {
				_, _ = cmdLine.RunCommand(ctx, nil, 0, "", "sh", "-c", "echo -n '"+invalidContent+"' > "+filename)
				err = fileInpOut.ReadJSONFile(filename, &jsonData)
				//fmt.Printf("Value: %+v\n\n", jsonData)
				Expect(err).NotTo(BeNil())
			})
		})

		Context("Writing to file", func() {
			BeforeEach(func() {
				filename = "/tmp/write_file/file1.json"
				content = "{\"key1\": \"value1\",\"key2\":\"value2\",\"key3\":{\"key4\":\"value4\"}}"

				//invalidContent = "\"key1\": value1\",\"key2\":value2\",\"key3\":{\"key4\":\"value4\"}}"
				fileInpOut = New()
				cmdLine = command.New()
				jsonData = map[string]interface{}{}
				fmt.Printf("Creating directory ...\n")
				ctx = context.TODO()
				_, _ = cmdLine.RunCommand(ctx, nil, 0, "", "mkdir", "-p", "/tmp/write_file")
				_, _, _, _ = cmdLine.RunCommandWithStdOutStdErr(ctx, nil, 0, "/tmp/write_file", "ls", "-ltr")
				//_ = cmdLine.RunCommand(nil, 0, "", "sh", "-c", "echo -n '" + content + "' > " +  filename)
			})

			AfterEach(func() {
				_, _ = cmdLine.RunCommand(ctx, nil, 0, "", "rm", "-rf", "/tmp/write_file")
				ctx.Done()
			})

			It("Should write a string to a file", func() {
				err = fileInpOut.WriteToFile(filename, content, false)
				Expect(err).To(BeNil())
				err = fileInpOut.ReadJSONFile(filename, &jsonData)
				Expect(err).To(BeNil())
				Expect(jsonData["key1"]).To(Equal("value1"))
				Expect(jsonData["key2"]).To(Equal("value2"))
			})

			It("Should write a slice of strings to a file", func() {
				sliceContent = []string{"{",
					"    \"key1\": \"value1\",",
					"    \"key2\": \"value2\",",
					"    \"key3\": {",
					"        \"key4\": \"value4\"",
					"    }",
					"}"}
				err = fileInpOut.WriteToFile(filename, sliceContent, false)
				Expect(err).To(BeNil())
				err = fileInpOut.ReadJSONFile(filename, &jsonData)
				Expect(err).To(BeNil())
				Expect(jsonData["key1"]).To(Equal("value1"))
				Expect(jsonData["key2"]).To(Equal("value2"))
			})

			It("Should write a slice of bytes to a file", func() {
				byteContent = []byte(`{"key1": "value1", "key2": "value2", "key3": {"key4": "value4"}}`)
				err = fileInpOut.WriteToFile(filename, byteContent, false)
				Expect(err).To(BeNil())
				err = fileInpOut.ReadJSONFile(filename, &jsonData)
				Expect(err).To(BeNil())
				Expect(jsonData["key1"]).To(Equal("value1"))
				Expect(jsonData["key2"]).To(Equal("value2"))
			})

			It("Should append a string to a file", func() {
				//_ = cmdLine.RunCommand(nil, 0, "", "sh", "-c", "echo -n '" + content + "' > " +  filename)
				content = "This is not a test!"
				err = fileInpOut.WriteToFile(filename, content+"\n", true)
				Expect(err).To(BeNil())
				err = fileInpOut.WriteToFile(filename, content, true)
				Expect(err).To(BeNil())
				_, stdout, err := cmdLine.RunCommandWithStdOut(ctx, nil, 0, "", "sh", "-c", "grep '"+content+"' "+filename+"| wc -l")
				Expect(err).To(BeNil())
				Expect(strings.TrimSpace(stdout[0])).To(Equal("2"))
			})

			It("Should append a slice of strings to a file", func() {
				content = "This is not a test!"
				err = fileInpOut.WriteToFile(filename, []string{content, content}, true)
				Expect(err).To(BeNil())
				_, stdout, err := cmdLine.RunCommandWithStdOut(ctx, nil, 0, "", "sh", "-c", "grep '"+content+"' "+filename+"| wc -l")
				Expect(err).To(BeNil())
				Expect(strings.TrimSpace(stdout[0])).To(Equal("2"))
			})

			It("Should append a slice of bytes to a file", func() {
				content = "This is not a test!"
				byteContent = []byte(content + "\n")
				byteContent = append(byteContent, byteContent...)
				err = fileInpOut.WriteToFile(filename, []string{content, content}, true)
				Expect(err).To(BeNil())
				_, stdout, err := cmdLine.RunCommandWithStdOut(ctx, nil, 0, "", "sh", "-c", "grep '"+content+"' "+filename+"| wc -l")
				Expect(err).To(BeNil())
				Expect(strings.TrimSpace(stdout[0])).To(Equal("2"))
			})
		})
		Context("Validating Sha256 Checksum", func() {
			var (
				dummy1       string
				dummy2       string
				content1     string
				content2     string
				checksumFile string
				expectedHash []string
				actualHash   []string
				hash2        []string
			)
			BeforeEach(func() {
				fileInpOut = New()
				cmdLine = command.New()
				ctx = context.TODO()
				_, _ = cmdLine.RunCommand(ctx, nil, 0, "", "mkdir", "testData")
				dummy1 = "testData/dummy1.txt"
				dummy2 = "testData/dummy2.txt"
				content1 = "this is dummy1 file"
				content2 = "this is dummy2 file"
				err = fileInpOut.WriteToFile(dummy1, content1, false)
				Expect(err).To(BeNil())
				err = fileInpOut.WriteToFile(dummy2, content2, false)
				Expect(err).To(BeNil())
				checksumFile = "testData/checksum/sha256sums.txt"
				_, expectedHash, _ = cmdLine.RunCommandWithStdOut(ctx, nil, 0, "", "/usr/bin/sha256sum", dummy1)
				_, hash2, _ = cmdLine.RunCommandWithStdOut(ctx, nil, 0, "", "/usr/bin/sha256sum", dummy2)
				expectedHash = append(expectedHash, hash2...)
			})

			AfterEach(func() {
				_, _ = cmdLine.RunCommand(ctx, nil, 0, "", "rm", "-rf", "testData")
				actualHash = nil
				expectedHash = nil
				ctx.Done()
			})
			It("Should generate hash for given file", func() {
				_, expectedHash, _ = cmdLine.RunCommandWithStdOut(ctx, nil, 0, "", "/usr/bin/sha256sum", dummy1)
				hash, err := fileInpOut.GenerateHashForFile(dummy1)
				actualHash = append(actualHash, hash)
				Expect(err).To(BeNil())
				Expect(actualHash).To(Equal(expectedHash))
			})
			It("Should fail if file absent", func() {
				fileToCheck := "testData/absent.txt"
				_, err := fileInpOut.GenerateHashForFile(fileToCheck)
				Expect(err).NotTo(BeNil())
			})
			It("Should generate hash for given directory", func() {
				dirToCheck := "testData"
				actualHash, err = fileInpOut.GenerateHashForDir(dirToCheck)
				Expect(err).To(BeNil())
				Expect(actualHash).To(Equal(expectedHash))
			})
			It("Should create checksum dir and checksum file for given directory ", func() {
				dirToCheck := "testData"
				err := fileInpOut.GenerateChecksum(dirToCheck)
				Expect(err).To(BeNil())
				actualHash, _ = fileInpOut.ReadFileByLine(checksumFile)
				Expect(actualHash).To(Equal(expectedHash))
			})
			It("Should create checksum file if checksum dir already present", func() {
				_, _ = cmdLine.RunCommand(ctx, nil, 0, "", "mkdir", "-p", "testData/checksum")
				dirToCheck := "testData"
				err := fileInpOut.GenerateChecksum(dirToCheck)
				Expect(err).To(BeNil())
				actualHash, _ = fileInpOut.ReadFileByLine(checksumFile)
				Expect(actualHash).To(Equal(expectedHash))
			})
			It("Should verify as true if files not changed", func() {
				dirToCheck := "testData"
				err := fileInpOut.GenerateChecksum(dirToCheck)
				Expect(err).To(BeNil())
				check, err := fileInpOut.VerifyChecksum(dirToCheck)
				Expect(err).To(BeNil())
				Expect(check).To(Equal(true))
			})
			It("Should verify as false if files are changed", func() {
				dirToCheck := "testData"
				err := fileInpOut.GenerateChecksum(dirToCheck)
				Expect(err).To(BeNil())
				content = "new data"
				err = fileInpOut.WriteToFile(dummy1, content, true)
				Expect(err).To(BeNil())
				check, err := fileInpOut.VerifyChecksum(dirToCheck)
				Expect(err).To(BeNil())
				Expect(check).To(Equal(false))
			})
		})

		Context("Writing Yaml", func() {
			type data struct {
				DnsIP string
			}
			var (
				templateYaml string
				newYaml      string
				newData      data
				content      string
			)
			BeforeEach(func() {
				fileInpOut = New()
				cmdLine = command.New()
				ctx = context.TODO()
				_, _ = cmdLine.RunCommand(ctx, nil, 0, "", "mkdir", "testData")
				templateYaml = "testData/tmplate.yaml"
				newYaml = "testData/new.yaml"
				newData = data{
					DnsIP: "10.20.30.40",
				}
			})
			AfterEach(func() {
				_, _ = cmdLine.RunCommand(ctx, nil, 0, "", "rm", "-rf", "testData")
				ctx.Done()
			})
			It("Should write replaced data to new yaml ", func() {
				content = "clusterIP: {{.DnsIP}}"
				err = fileInpOut.WriteToFile(templateYaml, content, false)
				Expect(err).To(BeNil())
				err = fileInpOut.NewYamlFromTemplateYaml(templateYaml, newYaml, newData)
				Expect(err).To(BeNil())
				actualData, err := fileInpOut.ReadFileByLine(newYaml)
				Expect(err).To(BeNil())
				expectedData := []string{"clusterIP: 10.20.30.40"}
				Expect(expectedData).To(Equal(actualData))
			})
		})
		Context("Listing files with pattern", func() {
			var (
				yaml1 string
				yaml2 string
				yml   string
				txt   string
			)
			BeforeEach(func() {
				fileInpOut = New()
				cmdLine = command.New()
				ctx = context.TODO()
				_, _ = cmdLine.RunCommand(ctx, nil, 0, "", "mkdir", "testData")
				yaml1 = "testData/a1.yaml"
				yaml2 = "testData/a2.yaml"
				yml = "testData/b.yml"
				txt = "testData/c.txt"
				content = "clusterIP: 123"
				err = fileInpOut.WriteToFile(yaml1, content, false)
				err = fileInpOut.WriteToFile(yaml2, content, false)
				err = fileInpOut.WriteToFile(yml, content, false)
				err = fileInpOut.WriteToFile(txt, content, false)
			})
			AfterEach(func() {
				_, _ = cmdLine.RunCommand(ctx, nil, 0, "", "rm", "-rf", "testData")
				ctx.Done()
			})
			It("Should list files with yaml extension ", func() {
				files, err := fileInpOut.ListFilesWithPatterns("testData", []string{"*.yaml"})
				Expect(err).To(BeNil())
				for _, f := range files {
					matched, err := filepath.Match("*.yaml", filepath.Base(f))
					Expect(err).To(BeNil())
					Expect(matched).To(Equal(true))
				}
			})
			It("Should list files with yml extension ", func() {
				files, err := fileInpOut.ListFilesWithPatterns("testData", []string{"*.yml"})
				Expect(err).To(BeNil())
				for _, f := range files {
					matched, err := filepath.Match("*.yml", filepath.Base(f))
					Expect(err).To(BeNil())
					Expect(matched).To(Equal(true))
				}
			})
			It("Should not list files with extension other than yml ", func() {
				files, err := fileInpOut.ListFilesWithPatterns("testData", []string{"*.yml"})
				Expect(err).To(BeNil())
				for _, f := range files {
					matched, err := filepath.Match("*.txt", filepath.Base(f))
					Expect(err).To(BeNil())
					Expect(matched).To(Equal(false))
				}
			})
		})
	})
})

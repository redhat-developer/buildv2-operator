// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package main_test

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/shipwright-io/build/cmd/git"
)

var _ = Describe("Git Resource", func() {
	var run = func(args ...string) error {
		log.SetOutput(ioutil.Discard)
		os.Args = append([]string{"tool", "--skip-validation"}, args...)
		return Execute(context.TODO())
	}

	var withTempDir = func(f func(target string)) {
		path, err := ioutil.TempDir(os.TempDir(), "git")
		Expect(err).ToNot(HaveOccurred())
		defer os.RemoveAll(path)

		f(path)
	}

	var withTempFile = func(pattern string, f func(filename string)) {
		file, err := ioutil.TempFile(os.TempDir(), pattern)
		Expect(err).ToNot(HaveOccurred())
		defer os.Remove(file.Name())

		f(file.Name())
	}

	var filecontent = func(path string) string {
		data, err := ioutil.ReadFile(path)
		Expect(err).ToNot(HaveOccurred())
		return string(data)
	}

	var file = func(path string, mode os.FileMode, data []byte) {
		Expect(ioutil.WriteFile(path, data, mode)).ToNot(HaveOccurred())
	}

	Context("validations and error cases", func() {
		It("should fail in case mandatory arguments are missing", func() {
			Expect(run()).To(HaveOccurred())
		})

		It("should fail in case --url is empty", func() {
			Expect(run(
				"--url", "",
				"--target", "/workspace/source",
			)).To(HaveOccurred())
		})

		It("should fail in case --target is empty", func() {
			Expect(run(
				"--url", "https://github.com/foo/bar",
				"--target", "",
			)).To(HaveOccurred())
		})

		It("should fail in case url does not exist", func() {
			withTempDir(func(target string) {
				Expect(run(
					"--url", "http://github.com/feqlQoDIHc/bcfHFHHXYF",
					"--target", target,
				)).To(HaveOccurred())
			})
		})

		It("should fail in case secret path content is not recognized", func() {
			withTempDir(func(secret string) {
				withTempDir(func(target string) {
					Expect(run(
						"--url", "https://github.com/foo/bar",
						"--target", target,
						"--secret-path", secret,
					)).To(HaveOccurred())
				})
			})
		})
	})

	Context("cloning publically available repositories", func() {
		const exampleRepo = "https://github.com/shipwright-io/sample-go"

		It("should Git clone a repository to the specified target directory", func() {
			withTempDir(func(target string) {
				Expect(run(
					"--url", exampleRepo,
					"--target", target,
				)).ToNot(HaveOccurred())

				Expect(filepath.Join(target, "README.md")).To(BeAnExistingFile())
			})
		})

		It("should Git clone a repository to the specified target directory using a specified branch", func() {
			withTempDir(func(target string) {
				Expect(run(
					"--url", exampleRepo,
					"--target", target,
					"--revision", "main",
				)).ToNot(HaveOccurred())

				Expect(filepath.Join(target, "README.md")).To(BeAnExistingFile())
			})
		})

		It("should Git clone a repository to the specified target directory using a specified tag", func() {
			withTempFile("commit-sha", func(filename string) {
				withTempDir(func(target string) {
					Expect(run(
						"--url", exampleRepo,
						"--target", target,
						"--revision", "v0.1.0",
						"--result-file-commit-sha", filename,
					)).ToNot(HaveOccurred())

					Expect(filecontent(filename)).To(Equal("8016b0437a7a09079f961e5003e81e5ad54e6c26"))
				})
			})
		})

		It("should Git clone a repository to the specified target directory using a specified commit-sha (long)", func() {
			withTempFile("commit-sha", func(filename string) {
				withTempDir(func(target string) {
					Expect(run(
						"--url", exampleRepo,
						"--target", target,
						"--revision", "0e0583421a5e4bf562ffe33f3651e16ba0c78591",
						"--result-file-commit-sha", filename,
					)).ToNot(HaveOccurred())

					Expect(filecontent(filename)).To(Equal("0e0583421a5e4bf562ffe33f3651e16ba0c78591"))
				})
			})
		})

		It("should Git clone a repository to the specified target directory using a specified commit-sha (short)", func() {
			withTempFile("commit-sha", func(filename string) {
				withTempDir(func(target string) {
					Expect(run(
						"--url", exampleRepo,
						"--target", target,
						"--revision", "0e05834",
						"--result-file-commit-sha", filename,
					)).ToNot(HaveOccurred())

					Expect(filecontent(filename)).To(Equal("0e0583421a5e4bf562ffe33f3651e16ba0c78591"))
				})
			})
		})
	})

	Context("cloning private repositories using SSH keys", func() {
		const exampleRepo = "git@github.com:shipwright-io/sample-nodejs-private.git"

		var sshPrivateKey string

		BeforeEach(func() {
			if sshPrivateKey = os.Getenv("TEST_GIT_PRIVATE_SSH_KEY"); sshPrivateKey == "" {
				Skip("Skipping private repository tests since TEST_GIT_PRIVATE_SSH_KEY environment variable is not set")
			}
		})

		It("should fail in case a private key is provided but no SSH Git URL", func() {
			withTempDir(func(secret string) {
				// Mock the filesystem state of `kubernetes.io/ssh-auth` type secret volume mount
				file(filepath.Join(secret, "ssh-privatekey"), 0400, []byte(sshPrivateKey))

				withTempDir(func(target string) {
					Expect(run(
						"--url", "https://github.com/foo/bar",
						"--secret-path", secret,
						"--target", target,
					)).To(HaveOccurred())
				})
			})
		})

		It("should fail in case a SSH Git URL is provided but no private key", func() {
			withTempDir(func(target string) {
				Expect(run(
					"--url", exampleRepo,
					"--secret-path", "/tmp/foobar",
					"--target", target,
				)).To(HaveOccurred())
			})
		})

		It("should Git clone a private repository using a SSH private key and known hosts file provided via a secret", func() {
			var knownHosts string
			if knownHosts = os.Getenv("TEST_GIT_KNOWN_HOSTS"); knownHosts == "" {
				Skip("Skipping private repository test since TEST_GIT_KNOWN_HOSTS environment variable is not set")
			}

			withTempDir(func(secret string) {
				// Mock the filesystem state of `kubernetes.io/ssh-auth` type secret volume mount
				file(filepath.Join(secret, "ssh-privatekey"), 0400, []byte(sshPrivateKey))
				file(filepath.Join(secret, "known_hosts"), 0600, []byte(knownHosts))

				withTempDir(func(target string) {
					Expect(run(
						"--url", exampleRepo,
						"--secret-path", secret,
						"--target", target,
					)).ToNot(HaveOccurred())

					Expect(filepath.Join(target, "README.md")).To(BeAnExistingFile())
				})
			})
		})

		It("should Git clone a private repository using a SSH private key provided via a secret", func() {
			withTempDir(func(secret string) {
				// Mock the filesystem state of `kubernetes.io/ssh-auth` type secret volume mount
				file(filepath.Join(secret, "ssh-privatekey"), 0400, []byte(sshPrivateKey))

				withTempDir(func(target string) {
					Expect(run(
						"--url", exampleRepo,
						"--secret-path", secret,
						"--target", target,
					)).ToNot(HaveOccurred())

					Expect(filepath.Join(target, "README.md")).To(BeAnExistingFile())
				})
			})
		})
	})

	Context("cloning private repositories using basic auth", func() {
		const exampleRepo = "https://github.com/shipwright-io/sample-nodejs-private"

		var username string
		var password string

		BeforeEach(func() {
			username = os.Getenv("TEST_GIT_PRIVATE_USERNAME")
			password = os.Getenv("TEST_GIT_PRIVATE_PASSWORD")
			if username == "" || password == "" {
				Skip("Skipping private repository tests since TEST_GIT_PRIVATE_USERNAME and/or TEST_GIT_PRIVATE_PASSWORD environment variables are not set")
			}
		})

		It("should fail in case only username or password is provided", func() {
			withTempDir(func(secret string) {
				// Mock the filesystem state of `kubernetes.io/basic-auth` type secret volume mount
				file(filepath.Join(secret, "password"), 0400, []byte(password))

				withTempDir(func(target string) {
					Expect(run(
						"--url", exampleRepo,
						"--secret-path", secret,
						"--target", target,
					)).To(HaveOccurred())
				})
			})
		})

		It("should Git clone a private repository using basic auth credentials provided via a secret", func() {
			withTempDir(func(secret string) {
				// Mock the filesystem state of `kubernetes.io/basic-auth` type secret volume mount
				file(filepath.Join(secret, "username"), 0400, []byte(username))
				file(filepath.Join(secret, "password"), 0400, []byte(password))

				withTempDir(func(target string) {
					Expect(run(
						"--url", exampleRepo,
						"--secret-path", secret,
						"--target", target,
					)).ToNot(HaveOccurred())

					Expect(filepath.Join(target, "README.md")).To(BeAnExistingFile())
				})
			})
		})
	})

	Context("cloning repositories with Git Large File Storage", func() {
		const exampleRepo = "https://github.com/shipwright-io/sample-lfs"

		BeforeEach(func() {
			if _, err := exec.LookPath("git-lfs"); err != nil {
				Skip("Skipping Git Large File Storage test as `git-lfs` binary is not in the PATH")
			}
		})

		It("should Git clone a repository to the specified target directory", func() {
			withTempDir(func(target string) {
				Expect(run(
					"--url", exampleRepo,
					"--target", target,
				)).ToNot(HaveOccurred())

				lfsFile := filepath.Join(target, "assets", "shipwright-logo-lightbg-512.png")
				Expect(lfsFile).To(BeAnExistingFile())

				data, err := ioutil.ReadFile(lfsFile)
				Expect(err).ToNot(HaveOccurred())
				Expect(http.DetectContentType(data)).To(Equal("image/png"))
			})
		})
	})

	Context("cloning repositories with submodules", func() {
		const exampleRepo = "https://github.com/shipwright-io/website"

		It("should Git clone a repository with a submodule", func() {
			withTempDir(func(target string) {
				Expect(run(
					"--url", exampleRepo,
					"--target", target,
				)).ToNot(HaveOccurred())

				Expect(filepath.Join(target, "README.md")).To(BeAnExistingFile())
				Expect(filepath.Join(target, "themes", "docsy", "README.md")).To(BeAnExistingFile())
			})
		})
	})

	Context("store details in result files", func() {
		const exampleRepo = "https://github.com/shipwright-io/sample-go"

		It("should store commit-sha into file specified in --result-file-commit-sha flag", func() {
			withTempFile("commit-sha", func(filename string) {
				withTempDir(func(target string) {
					Expect(run(
						"--url", exampleRepo,
						"--target", target,
						"--revision", "v0.1.0",
						"--result-file-commit-sha", filename,
					)).ToNot(HaveOccurred())

					Expect(filecontent(filename)).To(Equal("8016b0437a7a09079f961e5003e81e5ad54e6c26"))
				})
			})
		})

		It("should store commit-author into file specified in --result-file-commit-author flag", func() {
			withTempFile("commit-author", func(filename string) {

				withTempDir(func(target string) {
					Expect(run(
						"--url", exampleRepo,
						"--target", target,
						"--revision", "v0.1.0",
						"--result-file-commit-author", filename,
					)).ToNot(HaveOccurred())

					Expect(filecontent(filename)).To(Equal("Enrique Encalada"))
				})
			})
		})
	})
})

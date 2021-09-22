// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/pflag"
)

type credentialType int

const (
	typeUndef credentialType = iota
	typePrivateKey
	typeUsernamePassword
)

// ExitError is an error which has an exit code to be used in os.Exit() to
// return both an exit code and an error message
type ExitError struct {
	Code    int
	Message string
	Cause   error
}

func (e ExitError) Error() string {
	return fmt.Sprintf("%s (exit code %d)", e.Message, e.Code)
}

type settings struct {
	url                    string
	revision               string
	depth                  uint
	target                 string
	resultFileCommitSha    string
	resultFileCommitAuthor string
	secretPath             string
	skipValidation         bool
}

var flagValues settings

var (
	sshGitURLRegEx = regexp.MustCompile(`^(git@|ssh:\/\/).+$`)
	commitShaRegEx = regexp.MustCompile(`^[0-9a-f]{7,40}$`)
)

func init() {
	// Main flags for the Git step to define the configuration, for example
	// the flags for `url`, and `target` will always be used, but `revision`
	// depends on the respective use case.
	pflag.StringVar(&flagValues.url, "url", "", "The URL of the Git repository")
	pflag.StringVar(&flagValues.revision, "revision", "", "The revision of the Git repository to be cloned. Optional, defaults to the default branch.")
	pflag.StringVar(&flagValues.target, "target", "", "The target directory of the clone operation")
	pflag.StringVar(&flagValues.resultFileCommitSha, "result-file-commit-sha", "", "A file to write the commit sha to.")
	pflag.StringVar(&flagValues.resultFileCommitAuthor, "result-file-commit-author", "", "A file to write the commit author to.")
	pflag.StringVar(&flagValues.secretPath, "secret-path", "", "A directory that contains a secret. Either username and password for basic authentication. Or a SSH private key and optionally a known hosts file. Optional.")

	// Optional flag to be able to override the default shallow clone depth,
	// which should be fine for almost all use cases we use the Git source step
	// for (in the context of Shipwright build).
	pflag.UintVar(&flagValues.depth, "depth", 1, "Create a shallow clone based on the given depth")

	// Mostly internal flag
	pflag.BoolVar(&flagValues.skipValidation, "skip-validation", false, "skip pre-requisite validation")
}

func main() {
	if err := Execute(context.Background()); err != nil {
		var exitcode = 1
		switch err := err.(type) {
		case *ExitError:
			exitcode = err.Code
		}

		log.Print(err.Error())
		os.Exit(exitcode)
	}
}

// Execute performs flag parsing, input validation and the Git clone
func Execute(ctx context.Context) error {
	flagValues = settings{depth: 1}
	pflag.Parse()

	// pre-req checks
	if err := checkEnvironment(ctx); err != nil {
		return err
	}

	if err := runGitClone(ctx); err != nil {
		return err
	}

	return nil
}

func runGitClone(ctx context.Context) error {
	if flagValues.url == "" {
		return &ExitError{Code: 100, Message: "the 'url' argument must not be empty"}
	}

	if flagValues.target == "" {
		return &ExitError{Code: 101, Message: "the 'target' argument must not be empty"}
	}

	if err := clone(ctx); err != nil {
		return err
	}

	if flagValues.resultFileCommitSha != "" {
		output, err := git(ctx, "-C", flagValues.target, "rev-parse", "--verify", "HEAD")
		if err != nil {
			return err
		}

		if err := ioutil.WriteFile(flagValues.resultFileCommitSha, []byte(output), 0644); err != nil {
			return err
		}
	}

	if flagValues.resultFileCommitAuthor != "" {
		output, err := git(ctx, "-C", flagValues.target, "log", "-1", "--pretty=format:%an")
		if err != nil {
			return err
		}

		if err = ioutil.WriteFile(flagValues.resultFileCommitAuthor, []byte(output), 0644); err != nil {
			return err
		}
	}

	return nil
}

func checkEnvironment(ctx context.Context) error {
	if flagValues.skipValidation {
		return nil
	}

	var checks = []struct{ toolName, versionArg string }{
		{toolName: "ssh", versionArg: "-V"},
		{toolName: "git", versionArg: "version"},
		{toolName: "git-lfs", versionArg: "version"},
	}

	for _, check := range checks {
		path, err := exec.LookPath(check.toolName)
		if err != nil {
			return &ExitError{Code: 120, Message: err.Error(), Cause: err}
		}

		out, err := exec.CommandContext(ctx, path, check.versionArg).CombinedOutput()
		if err != nil {
			return err
		}

		log.Printf("Info: %s (%s): %s\n",
			check.toolName,
			path,
			strings.TrimRight(string(out), "\n"),
		)
	}

	return nil
}

func clone(ctx context.Context) error {
	cloneArgs := []string{
		"clone",
		"--quiet",
		"--no-tags",
	}

	var commitSha string
	switch {
	case commitShaRegEx.MatchString(flagValues.revision):
		commitSha = flagValues.revision
		cloneArgs = append(cloneArgs, "--no-checkout")

	default:
		cloneArgs = append(cloneArgs, "--single-branch")

		if flagValues.revision != "" {
			cloneArgs = append(cloneArgs, "--branch", flagValues.revision)
		}

		if flagValues.depth > 0 {
			cloneArgs = append(cloneArgs, "--depth", fmt.Sprintf("%d", flagValues.depth))
		}
	}

	var addtlCredArgs []string
	if flagValues.secretPath != "" {
		credType, err := checkCredentials()
		if err != nil {
			return err
		}

		switch credType {
		case typePrivateKey:
			// Since the key provided via a secret can have undesirable file
			// permissions, it will end up failing due to SSH sanity checks.
			// Therefore, create a temporary replacement with the right
			// file permissions.
			data, err := ioutil.ReadFile(filepath.Join(flagValues.secretPath, "ssh-privatekey"))
			if err != nil {
				return err
			}

			sshPrivateKeyFile, err := ioutil.TempFile(os.TempDir(), "ssh-private-key")
			if err != nil {
				return err
			}

			defer os.Remove(sshPrivateKeyFile.Name())

			if err := ioutil.WriteFile(sshPrivateKeyFile.Name(), data, 0400); err != nil {
				return err
			}

			var sshCmd = []string{"ssh",
				"-o", "LogLevel=ERROR",
				"-o", "BatchMode=yes",
				"-i", sshPrivateKeyFile.Name(),
			}

			var knownHostsFile = filepath.Join(flagValues.secretPath, "known_hosts")
			if hasFile(knownHostsFile) {
				sshCmd = append(sshCmd,
					"-o", "GlobalKnownHostsFile=/dev/null",
					"-o", fmt.Sprintf("UserKnownHostsFile=%s", knownHostsFile),
				)
			} else {
				sshCmd = append(sshCmd,
					"-o", "StrictHostKeyChecking=accept-new",
				)
			}

			addtlCredArgs = append(addtlCredArgs,
				"-c",
				fmt.Sprintf(`core.sshCommand=%s`, strings.Join(sshCmd, " ")),
			)

		case typeUsernamePassword:
			repoURL, err := url.Parse(flagValues.url)
			if err != nil {
				return err
			}

			username, err := ioutil.ReadFile(filepath.Join(flagValues.secretPath, "username"))
			if err != nil {
				return err
			}

			password, err := ioutil.ReadFile(filepath.Join(flagValues.secretPath, "password"))
			if err != nil {
				return err
			}

			repoURL.User = url.UserPassword(string(username), string(password))

			credHelperFile, err := ioutil.TempFile(os.TempDir(), "cred-helper-file")
			if err != nil {
				return err
			}

			defer os.Remove(credHelperFile.Name())

			if err := ioutil.WriteFile(credHelperFile.Name(), []byte(repoURL.String()), 0400); err != nil {
				return err
			}

			if _, err := git(ctx, "config", "--global", "credential.helper", fmt.Sprintf("store --file %s", credHelperFile.Name())); err != nil {
				return err
			}
		}
	}

	cloneArgs = append(cloneArgs, addtlCredArgs...)
	cloneArgs = append(cloneArgs, "--", flagValues.url, flagValues.target)
	if _, err := git(ctx, cloneArgs...); err != nil {
		return err
	}

	if commitSha != "" {
		if _, err := git(ctx, "-C", flagValues.target, "checkout", commitSha); err != nil {
			return err
		}
	}

	submoduleArgs := []string{"-C", flagValues.target}
	submoduleArgs = append(submoduleArgs, addtlCredArgs...)
	submoduleArgs = append(submoduleArgs, "submodule", "update", "--init", "--recursive")
	if flagValues.depth > 0 {
		submoduleArgs = append(submoduleArgs, "--depth", fmt.Sprintf("%d", flagValues.depth))
	}

	if _, err := git(ctx, submoduleArgs...); err != nil {
		return err
	}

	revision := flagValues.revision
	if revision == "" {
		// user requested to clone the default branch, determine the branch name
		refParse, err := git(ctx, "-C", flagValues.target, "rev-parse", "--abbrev-ref", "HEAD")
		if err != nil {
			return err
		}

		revision = strings.TrimRight(refParse, "\n")
	}

	log.Printf("Successfully loaded %s (%s) into %s\n",
		flagValues.url,
		revision,
		flagValues.target,
	)

	return nil
}

func git(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	log.Print(cmd.String())

	// Make sure that the spawned process does not try to prompt for infos
	os.Setenv("GIT_TERMINAL_PROMPT", "0")
	cmd.Stdin = nil

	out, err := cmd.CombinedOutput()

	var output string
	if out != nil {
		output = strings.TrimRight(string(out), "\n")
	}

	if err != nil {
		// In case the command fails, it is very likely to be an ExitCode error
		// which contains the exit code of the command. Create a custom exit
		// error where the command output is placed into the error to be more
		// readable to the end-user.
		switch terr := err.(type) {
		case *exec.ExitError:
			err = &ExitError{
				Code:    terr.ExitCode(),
				Message: output,
				Cause:   err,
			}
		}
	}

	return output, err
}

func hasFile(elem ...string) bool {
	_, err := os.Stat(filepath.Join(elem...))
	return !os.IsNotExist(err)
}

func checkCredentials() (credentialType, error) {
	// Checking whether mounted secret is of type `kubernetes.io/ssh-auth`
	// in which case there is a file called ssh-privatekey
	hasPrivateKey := hasFile(flagValues.secretPath, "ssh-privatekey")
	isSSHGitURL := sshGitURLRegEx.MatchString(flagValues.url)
	switch {
	case hasPrivateKey && isSSHGitURL:
		return typePrivateKey, nil

	case hasPrivateKey && !isSSHGitURL:
		return typeUndef, &ExitError{Code: 110, Message: "Credential/URL inconsistency: SSH credentials provided, but URL is not a SSH Git URL"}

	case !hasPrivateKey && isSSHGitURL:
		return typeUndef, &ExitError{Code: 110, Message: "Credential/URL inconsistency: No SSH credentials provided, but URL is a SSH Git URL"}
	}

	// Checking whether mounted secret is of type `kubernetes.io/basic-auth`
	// in which case there need to be the files username and password
	hasUsername := hasFile(flagValues.secretPath, "username")
	hasPassword := hasFile(flagValues.secretPath, "password")
	isHTTPSURL := strings.HasPrefix(flagValues.url, "https")
	switch {
	case hasUsername && hasPassword && isHTTPSURL:
		return typeUsernamePassword, nil

	case hasUsername && !hasPassword || !hasUsername && hasPassword:
		return typeUndef, &ExitError{Code: 110, Message: "Basic Auth incomplete: Both username and password need to be configured"}

	}

	return typeUndef, &ExitError{Code: 110, Message: "Unsupported type of credentials provided, either SSH private key or username/password is supported"}
}

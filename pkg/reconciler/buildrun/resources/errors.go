// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"fmt"
	"strings"
)

// ClientStatusUpdateError is an error that occurrs when trying to update a runtime object status
type ClientStatusUpdateError struct {
	reason error
}

func (e ClientStatusUpdateError) Error() string {
	return fmt.Sprintf("failed to update client status: %v", e.reason)
}

// IsClientStatusUpdateError checks whether the given error is of type ClientStatusUpdateError or
// in case this is a list of errors, that it contains at least one error of type ClientStatusUpdateError
func IsClientStatusUpdateError(err error) bool {
	if err == nil {
		return false
	}

	switch terr := err.(type) {
	case *ClientStatusUpdateError, ClientStatusUpdateError:
		return true
	case Errors:
		for _, e := range terr.errors {
			if IsClientStatusUpdateError(e) {
				return true
			}
		}
	}
	return false
}

// Errors allows you to wrap multiple errors
// in a single struct. Useful when wrapping multiple
// errors with a single message.
type Errors struct {
	message string
	errors  []error
}

func (e Errors) Error() string {
	strErrors := make([]string, len(e.errors))
	for i, err := range e.errors {
		if err != nil {
			strErrors[i] = err.Error()
		}
	}
	return fmt.Sprintf("errors: %s, msg: %s", strings.Join(strErrors, ", "), e.message)
}

// HandleError returns multiple errors if each error is not nil.
// And its error message.
func HandleError(message string, listOfErrors ...error) Errors {
	return Errors{message: message, errors: listOfErrors}
}

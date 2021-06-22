// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/util/errors"

	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
)

// Env implements the Env interface to add validations for the `build.spec.env` slice.
type Env struct {
	Build *build.Build
}

// ValidatePath executes the validation routine, inspecting the `build.spec.env` path, which
// contains a slice of corev1.EnvVar.
func (e *Env) ValidatePath(_ context.Context) error {
	if e.Build.Spec.Env == nil {
		return nil
	}

	var allErrs []error
	for _, envVar := range e.Build.Spec.Env {
		if errs := e.validate(envVar); len(errs) != 0 {
			allErrs = append(allErrs, errs...)
		}
	}

	if len(allErrs) != 0 {
		return fmt.Errorf("%s", kerrors.NewAggregate(allErrs).Error())
	}

	return nil
}

// validate inspects each environment variable and validates all required attributes.
func (e *Env) validate(envVar corev1.EnvVar) []error {
	var allErrs []error

	if envVar.Name == "" {
		e.Build.Status.Reason = build.SpecEnvNameCanNotBeBlank
		e.Build.Status.Message = "name for environment variable must not be blank"
		allErrs = append(allErrs, fmt.Errorf("%s", e.Build.Status.Message))
	}

	if envVar.Value == "" {
		e.Build.Status.Reason = build.SpecEnvValueCanNotBeBlank
		e.Build.Status.Message = fmt.Sprintf("value for environment variable %q must not be blank", envVar.Name)
		allErrs = append(allErrs, fmt.Errorf("%s", e.Build.Status.Message))
	}

	return allErrs
}

// NewEnv instantiates a new Env passing the build object pointer along.
func NewEnv(b *build.Build) *Env {
	return &Env{Build: b}
}

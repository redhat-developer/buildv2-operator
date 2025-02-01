// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"context"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/utils/ptr"

	build "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
)

// SchedulerNameRef contains all required fields
// to validate a Scheduler name
type SchedulerNameRef struct {
	Build *build.Build // build instance for analysis
}

func NewSchedulerName(build *build.Build) *SchedulerNameRef {
	return &SchedulerNameRef{build}
}

// ValidatePath implements BuildPath interface and validates
// that SchedulerName values are valid
func (b *SchedulerNameRef) ValidatePath(_ context.Context) error {
	if b.Build.Spec.SchedulerName != "" {
		if errs := validation.IsQualifiedName(b.Build.Spec.SchedulerName); len(errs) > 0 {
			b.Build.Status.Reason = ptr.To(build.SchedulerNameNotValid)
			b.Build.Status.Message = ptr.To(strings.Join(errs, ", "))
		}
	}
	return nil
}

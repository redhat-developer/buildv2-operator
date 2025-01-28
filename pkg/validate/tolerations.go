// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"context"
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/utils/ptr"

	build "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
)

// TolerationsRef contains all required fields
// to validate tolerations
type TolerationsRef struct {
	Build *build.Build // build instance for analysis
}

func NewTolerations(build *build.Build) *TolerationsRef {
	return &TolerationsRef{build}
}

// ValidatePath implements BuildPath interface and validates
// that tolerations key/operator/value are valid
func (b *TolerationsRef) ValidatePath(_ context.Context) error {
	for _, toleration := range b.Build.Spec.Tolerations {
		// validate Key
		if errs := validation.IsQualifiedName(toleration.Key); errs != nil {
			b.Build.Status.Reason = ptr.To(build.TolerationNotValid)
			b.Build.Status.Message = ptr.To(strings.Join(errs, ", "))
		}
		// validate Operator
		if !((toleration.Operator == v1.TolerationOpExists) || (toleration.Operator == v1.TolerationOpEqual)) {
			b.Build.Status.Reason = ptr.To(build.TolerationNotValid)
			b.Build.Status.Message = ptr.To(fmt.Sprintf("Toleration operator not valid. Must be one of: '%v', '%v'", v1.TolerationOpExists, v1.TolerationOpEqual))
		}
		// validate Value
		if errs := validation.IsValidLabelValue(toleration.Value); errs != nil {
			b.Build.Status.Reason = ptr.To(build.TolerationNotValid)
			b.Build.Status.Message = ptr.To(strings.Join(errs, ", "))
		}
		// validate Taint Effect, of which only "NoSchedule" is supported
		if !((toleration.Effect) == "" || (toleration.Effect == v1.TaintEffectNoSchedule)) {
			b.Build.Status.Reason = ptr.To(build.TolerationNotValid)
			b.Build.Status.Message = ptr.To(fmt.Sprintf("Only the '%v' toleration effect is supported.", v1.TaintEffectNoSchedule))
		}
		// validate TolerationSeconds, which should not be specified
		if toleration.TolerationSeconds != nil {
			b.Build.Status.Reason = ptr.To(build.TolerationNotValid)
			b.Build.Status.Message = ptr.To("Specifying TolerationSeconds is not supported.")
		}
	}

	return nil
}

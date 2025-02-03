// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/utils/ptr"

	build "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
)

// NodeSelectorRef contains all required fields
// to validate a node selector
type NodeSelectorRef struct {
	Build *build.Build // build instance for analysis
}

func NewNodeSelector(build *build.Build) *NodeSelectorRef {
	return &NodeSelectorRef{build}
}

// ValidatePath implements BuildPath interface and validates
// that NodeSelector keys/values are valid labels
func (b *NodeSelectorRef) ValidatePath(_ context.Context) error {
	for key, value := range b.Build.Spec.NodeSelector {
		if errs := validation.IsQualifiedName(key); len(errs) > 0 {
			b.Build.Status.Reason = ptr.To(build.NodeSelectorNotValid)
			b.Build.Status.Message = ptr.To(fmt.Sprintf("Node selector key not valid: %v", strings.Join(errs, ", ")))
		}
		if errs := validation.IsValidLabelValue(value); len(errs) > 0 {
			b.Build.Status.Reason = ptr.To(build.NodeSelectorNotValid)
			b.Build.Status.Message = ptr.To(fmt.Sprintf("Node selector value not valid: %v", strings.Join(errs, ", ")))
		}
	}

	return nil
}

// BuildRunNodeSelector is used to validate nodeSelectors in the BuildRun object
func BuildRunNodeSelector(nodeSelector map[string]string) (bool, string, string) {
	for key, value := range nodeSelector {
		if errs := validation.IsQualifiedName(key); len(errs) > 0 {
			return false, string(build.NodeSelectorNotValid), fmt.Sprintf("Node selector key not valid: %v", strings.Join(errs, ", "))
		}
		if errs := validation.IsValidLabelValue(value); len(errs) > 0 {
			return false, string(build.NodeSelectorNotValid), fmt.Sprintf("Node selector value not valid: %v", strings.Join(errs, ", "))
		}
	}
	return true, "", ""
}

// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
)

// GetBuildObject retrieves an existing Build based on a name and namespace
func GetBuildObject(ctx context.Context, client client.Client, buildRun *buildv1beta1.BuildRun, build *buildv1beta1.Build) error {
	// Option #1: BuildRef is specified
	// An actual Build resource is specified by name and needs to be looked up in the cluster.
	if buildRun.Spec.Build.Name != nil {
		err := client.Get(ctx, types.NamespacedName{Name: buildRun.Spec.BuildName(), Namespace: buildRun.Namespace}, build)
		if apierrors.IsNotFound(err) {
			// stop reconciling and mark the BuildRun as Failed
			// we only reconcile again if the status.Update call fails
			if updateErr := UpdateConditionWithFalseStatus(ctx, client, buildRun, fmt.Sprintf("build.shipwright.io %q not found", *buildRun.Spec.Build.Name), ConditionBuildNotFound); updateErr != nil {
				return HandleError("build object not found", err, updateErr)
			}
		}

		return err
	}

	// Option #2: BuildSpec is specified
	// The build specification is embedded in the BuildRun itself, create a transient Build resource.
	if buildRun.Spec.Build.Spec != nil {
		build.Name = ""
		build.Namespace = buildRun.Namespace
		build.Status = buildv1beta1.BuildStatus{}
		buildRun.Spec.Build.Spec.DeepCopyInto(&build.Spec)
		// In this case, fields set only on the BuildRun object do not get validated as they are not copied to the transient Build resource.
		// explicitly setting them here is required for validation to happen.
		build.Spec.NodeSelector = buildRun.Spec.NodeSelector
		build.Spec.Tolerations = buildRun.Spec.Tolerations
		return nil
	}

	// Bail out hard in case of an invalid state
	return fmt.Errorf("invalid BuildRun resource that neither has a BuildRef nor an embedded BuildSpec")
}

// IsOwnedByBuild checks if the controllerReferences contains a well known owner Kind
func IsOwnedByBuild(build *buildv1beta1.Build, controlledReferences []metav1.OwnerReference) bool {
	for _, ref := range controlledReferences {
		if ref.Kind == build.Kind && ref.Name == build.Name {
			return true
		}
	}

	return false
}

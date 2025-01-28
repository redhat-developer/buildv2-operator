// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	build "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources"
)

const (
	// Secrets for validating secret references in Build objects
	Secrets = "secrets"
	// Strategies for validating strategy references in Build objects
	Strategies = "strategy"
	// SourceURL for validating the source URL in Build objects
	SourceURL = "sourceurl"
	// Sources for validating `spec.sources` entries
	Source = "source"
	// Output for validating `spec.output` entry
	Output = "output"
	// BuildName for validating `metadata.name` entry
	BuildName = "buildname"
	// Envs for validating `spec.env` entries
	Envs = "env"
	// OwnerReferences for validating the ownerreferences between a Build
	// and BuildRun objects
	OwnerReferences = "ownerreferences"
	// Triggers for validating the `.spec.triggers` entries
	Triggers = "triggers"
	// NodeSelector for validating `spec.nodeSelector` entry
	NodeSelector = "nodeselector"
	// Tolerations for validating `spec.tolerations` entry
	Tolerations = "tolerations"
)

const (
	namespace = "namespace"
	name      = "name"
)

// BuildPath is an interface that holds a ValidatePath() function
// for validating different Build spec paths
type BuildPath interface {
	ValidatePath(ctx context.Context) error
}

// NewValidation returns a specific structure that implements
// BuildPath interface
func NewValidation(
	validationType string,
	build *build.Build,
	client client.Client,
	scheme *runtime.Scheme,
) (BuildPath, error) {
	switch validationType {
	case Secrets:
		return &Credentials{Build: build, Client: client}, nil
	case Strategies:
		return &Strategy{Build: build, Client: client}, nil
	case SourceURL:
		return &SourceURLRef{Build: build, Client: client}, nil
	case OwnerReferences:
		return &OwnerRef{Build: build, Client: client, Scheme: scheme}, nil
	case Source:
		return &SourceRef{Build: build}, nil
	case Output:
		return &BuildSpecOutputValidator{Build: build}, nil
	case BuildName:
		return &BuildNameRef{Build: build}, nil
	case Envs:
		return &Env{Build: build}, nil
	case Triggers:
		return &Trigger{build: build}, nil
	case NodeSelector:
		return &NodeSelectorRef{Build: build}, nil
	case Tolerations:
		return &TolerationsRef{Build: build}, nil
	default:
		return nil, fmt.Errorf("unknown validation type")
	}
}

// All runs all given validations and exists at the first technical error
func All(ctx context.Context, validations ...BuildPath) error {
	for _, validatation := range validations {
		if err := validatation.ValidatePath(ctx); err != nil {
			return err
		}
	}

	return nil
}

// BuildRunFields runs field validations against a BuildRun to detect
// disallowed field combinations and issues
func BuildRunFields(buildRun *build.BuildRun) (string, string) {
	if buildRun.Spec.Build.Spec == nil && buildRun.Spec.Build.Name == nil {
		return resources.BuildRunNoRefOrSpec,
			"no build referenced or specified, either 'buildRef' or 'buildSpec' has to be set"
	}

	if buildRun.Spec.Build.Spec != nil {
		if buildRun.Spec.Build.Name != nil {
			return resources.BuildRunAmbiguousBuild,
				"fields 'buildRef' and 'buildSpec' are mutually exclusive"
		}

		if buildRun.Spec.Output != nil {
			return resources.BuildRunBuildFieldOverrideForbidden,
				"cannot use 'output' override and 'buildSpec' simultaneously"
		}

		if len(buildRun.Spec.ParamValues) > 0 {
			return resources.BuildRunBuildFieldOverrideForbidden,
				"cannot use 'paramValues' override and 'buildSpec' simultaneously"
		}

		if len(buildRun.Spec.Env) > 0 {
			return resources.BuildRunBuildFieldOverrideForbidden,
				"cannot use 'env' override and 'buildSpec' simultaneously"
		}

		if buildRun.Spec.Timeout != nil {
			return resources.BuildRunBuildFieldOverrideForbidden,
				"cannot use 'timeout' override and 'buildSpec' simultaneously"
		}

		if buildRun.Spec.Build.Spec.Trigger != nil {
			return resources.BuildRunBuildFieldOverrideForbidden,
				"cannot use 'triggers' override in the 'BuildRun', only allowed in the 'Build'"
		}
	}

	return "", ""
}

// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
)

const (
	// Secrets for validating secret references in Build objects
	Secrets = "secrets"
	// Strategies for validating strategy references in Build objects
	Strategies = "strategy"
	// SourceURL for validating the source URL in Build objects
	SourceURL = "sourceurl"
	// Sources for validating `spec.sources` entries
	Sources = "sources"
	// BuildName for validating `metadata.name` entry
	BuildName = "buildname"
	// Envs for validating `spec.env` entries
	Envs = "env"
	// OwnerReferences for validating the ownerreferences between a Build
	// and BuildRun objects
	OwnerReferences = "ownerreferences"
	namespace       = "namespace"
	name            = "name"
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
	case Sources:
		return &SourcesRef{Build: build}, nil
	case BuildName:
		return &BuildNameRef{Build: build}, nil
	case Envs:
		return &Env{Build: build}, nil
	default:
		return nil, fmt.Errorf("unknown validation type")
	}
}

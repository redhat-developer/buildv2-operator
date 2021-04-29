// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources/sources"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

func AmendTaskSpecWithSources(
	cfg *config.Config,
	taskSpec *v1beta1.TaskSpec,
	build *buildv1alpha1.Build,
) {
	// create the step for spec.source, this is always Git
	sources.AppendGitStep(cfg, taskSpec, build.Spec.Source, "default")

	// create the step for spec.sources, this will eventually change into different steps depending on the type of the source
	if build.Spec.Sources != nil {
		for _, source := range *build.Spec.Sources {
			// today, we only have HTTP sources
			sources.AppendHttpStep(cfg, taskSpec, source)
		}
	}
}

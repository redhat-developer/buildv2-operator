// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package sources

import (
	"fmt"

	core "k8s.io/api/core/v1"

	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"

	pipeline "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

// AppendBundleStep appends the bundle step to the TaskSpec
func AppendBundleStep(
	cfg *config.Config,
	taskSpec *pipeline.TaskSpec,
	source build.Source,
	name string,
) {
	// append the result
	taskSpec.Results = append(taskSpec.Results, pipeline.TaskResult{
		Name:        fmt.Sprintf("%s-source-%s-bundle-image-digest", prefixParamsResultsVolumes, name),
		Description: "The digest of the bundle image.",
	})

	// initialize the step from the template
	bundleStep := pipeline.Step{
		Container: *cfg.BundleContainerTemplate.DeepCopy(),
	}

	// add the build-specific details
	bundleStep.Container.Name = fmt.Sprintf("source-%s", name)
	bundleStep.Container.Args = []string{
		"--image", source.BundleContainer.Image,
		"--target", fmt.Sprintf("$(params.%s-%s)", prefixParamsResultsVolumes, paramSourceRoot),
		"--result-file-image-digest", fmt.Sprintf("$(results.%s-source-%s-bundle-image-digest.path)", prefixParamsResultsVolumes, name),
	}

	// add credentials mount, if provided
	if source.Credentials != nil {
		AppendSecretVolume(taskSpec, source.Credentials.Name)

		secretMountPath := fmt.Sprintf("/workspace/%s-pull-secret", prefixParamsResultsVolumes)

		// define the volume mount on the container
		bundleStep.VolumeMounts = append(bundleStep.VolumeMounts, core.VolumeMount{
			Name:      SanitizeVolumeNameForSecretName(source.Credentials.Name),
			MountPath: secretMountPath,
			ReadOnly:  true,
		})

		// append the argument
		bundleStep.Container.Args = append(bundleStep.Container.Args,
			"--secret-path", secretMountPath,
		)
	}

	taskSpec.Steps = append(taskSpec.Steps, bundleStep)
}

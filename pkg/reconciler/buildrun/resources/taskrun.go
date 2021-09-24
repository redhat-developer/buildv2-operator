// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	taskv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	buildv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/env"
)

const (
	prefixParamsResultsVolumes = "shp"

	paramOutputImage   = "output-image"
	paramSourceRoot    = "source-root"
	paramSourceContext = "source-context"

	resultImageDigest = "image-digest"
	resultImageSize   = "image-size"

	workspaceSource = "source"

	inputParamBuilder    = "BUILDER_IMAGE"
	inputParamDockerfile = "DOCKERFILE"
	inputParamContextDir = "CONTEXT_DIR"

	imageMutateContainerName = "mutate-image"
)

// getStringTransformations gets us MANDATORY replacements using
// a poor man's templating mechanism - TODO: Use golang templating
func getStringTransformations(fullText string) string {

	stringTransformations := map[string]string{
		// this will be removed, build strategy author should use $(params.shp-output-image) directly
		"$(build.output.image)": fmt.Sprintf("$(params.%s-%s)", prefixParamsResultsVolumes, paramOutputImage),

		"$(build.builder.image)": fmt.Sprintf("$(inputs.params.%s)", inputParamBuilder),
		"$(build.dockerfile)":    fmt.Sprintf("$(inputs.params.%s)", inputParamDockerfile),

		// this will be removed, build strategy author should use $(params.shp-source-context); it is still needed by the ko build
		// strategy that mis-uses this setting to store the path to the main package; requires strategy parameter support to get rid
		"$(build.source.contextDir)": fmt.Sprintf("$(inputs.params.%s)", inputParamContextDir),
	}

	// Run the text through all possible replacements
	for k, v := range stringTransformations {
		fullText = strings.ReplaceAll(fullText, k, v)
	}
	return fullText
}

// GenerateTaskSpec creates Tekton TaskRun spec to be used for a build run
func GenerateTaskSpec(
	cfg *config.Config,
	build *buildv1alpha1.Build,
	buildRun *buildv1alpha1.BuildRun,
	buildSteps []buildv1alpha1.BuildStep,
	strategyParams []buildv1alpha1.Parameter,
) (*v1beta1.TaskSpec, error) {

	generatedTaskSpec := v1beta1.TaskSpec{
		Params: []v1beta1.ParamSpec{
			{
				Description: "Path to the Dockerfile",
				Name:        inputParamDockerfile,
				Default: &v1beta1.ArrayOrString{
					Type:      v1beta1.ParamTypeString,
					StringVal: "Dockerfile",
				},
			},
			{
				// CONTEXT_DIR comes from the git source specification
				// in the Build object
				Description: "The root of the code",
				Name:        inputParamContextDir,
				Default: &v1beta1.ArrayOrString{
					Type:      v1beta1.ParamTypeString,
					StringVal: ".",
				},
			},
			{
				Name:        fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramOutputImage),
				Description: "The URL of the image that the build produces",
				Type:        taskv1.ParamTypeString,
			},
			{
				Name:        fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramSourceContext),
				Description: "The context directory inside the source directory",
				Type:        taskv1.ParamTypeString,
			},
			{
				Name:        fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramSourceRoot),
				Description: "The source directory",
				Type:        taskv1.ParamTypeString,
			},
		},
		Results: []v1beta1.TaskResult{
			{
				Name:        fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, resultImageDigest),
				Description: "The digest of the image",
			},
			{
				Name:        fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, resultImageSize),
				Description: "The compressed size of the image",
			},
		},
		Workspaces: []v1beta1.WorkspaceDeclaration{
			// workspace for the source files
			{
				Name: workspaceSource,
			},
		},
	}

	if build.Spec.Builder != nil {
		InputBuilder := v1beta1.ParamSpec{
			Description: "Image containing the build tools/logic",
			Name:        inputParamBuilder,
			Default: &v1beta1.ArrayOrString{
				Type:      v1beta1.ParamTypeString,
				StringVal: build.Spec.Builder.Image,
			},
		}
		generatedTaskSpec.Params = append(generatedTaskSpec.Params, InputBuilder)
	}

	// define results, steps and volumes for sources
	AmendTaskSpecWithSources(cfg, &generatedTaskSpec, build)

	// Add the strategy defined parameters into the Task spec
	for _, p := range strategyParams {

		param := v1beta1.ParamSpec{
			Name:        p.Name,
			Description: p.Description,
		}

		// verify if the paramSpec Default requires a default
		// value or not
		if p.Default != nil {
			param.Default = &v1beta1.ArrayOrString{
				Type:      v1beta1.ParamTypeString,
				StringVal: *p.Default,
			}
		}

		generatedTaskSpec.Params = append(generatedTaskSpec.Params, param)

	}

	// define the steps coming from the build strategy
	for _, containerValue := range buildSteps {

		var taskCommand []string
		for _, buildStrategyCommandPart := range containerValue.Command {
			taskCommand = append(taskCommand, getStringTransformations(buildStrategyCommandPart))
		}

		var taskArgs []string
		for _, buildStrategyArgPart := range containerValue.Args {
			taskArgs = append(taskArgs, getStringTransformations(buildStrategyArgPart))
		}

		taskImage := getStringTransformations(containerValue.Image)

		step := v1beta1.Step{
			Container: corev1.Container{
				Image:           taskImage,
				ImagePullPolicy: containerValue.ImagePullPolicy,
				Name:            containerValue.Name,
				VolumeMounts:    containerValue.VolumeMounts,
				Command:         taskCommand,
				Args:            taskArgs,
				SecurityContext: containerValue.SecurityContext,
				WorkingDir:      containerValue.WorkingDir,
				Resources:       containerValue.Resources,
				Env:             containerValue.Env,
			},
		}

		generatedTaskSpec.Steps = append(generatedTaskSpec.Steps, step)

		// Get volumeMounts added to Task's spec.Volumes
		for _, volumeInBuildStrategy := range containerValue.VolumeMounts {
			newVolume := true
			for _, volumeInTask := range generatedTaskSpec.Volumes {
				if volumeInTask.Name == volumeInBuildStrategy.Name {
					newVolume = false
				}
			}
			if newVolume {
				generatedTaskSpec.Volumes = append(generatedTaskSpec.Volumes, corev1.Volume{
					Name: volumeInBuildStrategy.Name,
				})
			}
		}
	}

	// Amending task spec with image mutate step if annotations or labels are
	// specified in build manifest
	if len(build.Spec.Output.Annotations) > 0 || len(build.Spec.Output.Labels) > 0 {
		amendTaskSpecWithImageMutate(cfg, &generatedTaskSpec, build.Spec.Output)
	}

	// Combine the environment variables specified in the Build object and the BuildRun object
	// env vars in the BuildRun supercede those in the Build, overwriting them
	combinedEnvs, err := env.MergeEnvVars(buildRun.Spec.Env, build.Spec.Env, true)
	if err != nil {
		return nil, err
	}

	for i, step := range generatedTaskSpec.Steps {
		if step.Container.Env == nil {
			generatedTaskSpec.Steps[i].Container.Env = []corev1.EnvVar{}
		}

		// Merge the combined env vars from the Build and BuildRun into the Containers env vars
		// Any collision between the env vars in the Container step and those in the Build/BuildRun
		// will result in an error and cause a failed TaskRun
		if generatedTaskSpec.Steps[i].Container.Env, err = env.MergeEnvVars(combinedEnvs, generatedTaskSpec.Steps[i].Container.Env, false); err != nil {
			return nil, fmt.Errorf("%s in BuildStrategy %q", err.Error(), build.Spec.StrategyName())
		}
	}

	return &generatedTaskSpec, nil
}

// GenerateTaskRun creates a Tekton TaskRun to be used for a build run
func GenerateTaskRun(
	cfg *config.Config,
	build *buildv1alpha1.Build,
	buildRun *buildv1alpha1.BuildRun,
	serviceAccountName string,
	strategy buildv1alpha1.BuilderStrategy,
) (*v1beta1.TaskRun, error) {

	// retrieve expected imageURL form build or buildRun
	var image string
	if buildRun.Spec.Output != nil {
		image = buildRun.Spec.Output.Image
	} else {
		image = build.Spec.Output.Image
	}

	taskSpec, err := GenerateTaskSpec(
		cfg,
		build,
		buildRun,
		strategy.GetBuildSteps(),
		strategy.GetParameters(),
	)
	if err != nil {
		return nil, err
	}

	expectedTaskRun := &v1beta1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: buildRun.Name + "-",
			Namespace:    buildRun.Namespace,
			Labels: map[string]string{
				buildv1alpha1.LabelBuild:              build.Name,
				buildv1alpha1.LabelBuildGeneration:    strconv.FormatInt(build.Generation, 10),
				buildv1alpha1.LabelBuildRun:           buildRun.Name,
				buildv1alpha1.LabelBuildRunGeneration: strconv.FormatInt(buildRun.Generation, 10),
			},
		},
		Spec: v1beta1.TaskRunSpec{
			ServiceAccountName: serviceAccountName,
			TaskSpec:           taskSpec,
			Workspaces: []v1beta1.WorkspaceBinding{
				// workspace for the source files
				{
					Name:     workspaceSource,
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		},
	}

	// assign the annotations from the build strategy, filter out those that should not be propagated
	taskRunAnnotations := make(map[string]string)
	for key, value := range strategy.GetAnnotations() {
		if isPropagatableAnnotation(key) {
			taskRunAnnotations[key] = value
		}
	}
	if len(taskRunAnnotations) > 0 {
		expectedTaskRun.Annotations = taskRunAnnotations
	}

	for label, value := range strategy.GetResourceLabels() {
		expectedTaskRun.Labels[label] = value
	}

	expectedTaskRun.Spec.Timeout = effectiveTimeout(build, buildRun)

	params := []v1beta1.Param{
		{
			// shp-output-image
			Name: fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramOutputImage),
			Value: v1beta1.ArrayOrString{
				Type:      v1beta1.ParamTypeString,
				StringVal: image,
			},
		},
		{
			Name: fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramSourceRoot),
			Value: v1beta1.ArrayOrString{
				Type:      v1beta1.ParamTypeString,
				StringVal: "/workspace/source",
			},
		},
	}
	if build.Spec.Builder != nil {
		params = append(params, v1beta1.Param{
			Name: inputParamBuilder,
			Value: v1beta1.ArrayOrString{
				Type:      v1beta1.ParamTypeString,
				StringVal: build.Spec.Builder.Image,
			},
		})
	}
	if build.Spec.Dockerfile != nil {
		params = append(params, v1beta1.Param{
			Name: inputParamDockerfile,
			Value: v1beta1.ArrayOrString{
				Type:      v1beta1.ParamTypeString,
				StringVal: *build.Spec.Dockerfile,
			},
		})
	}
	if build.Spec.Source.ContextDir != nil {
		params = append(params, v1beta1.Param{
			Name: inputParamContextDir,
			Value: v1beta1.ArrayOrString{
				Type:      v1beta1.ParamTypeString,
				StringVal: *build.Spec.Source.ContextDir,
			},
		})
		params = append(params, v1beta1.Param{
			Name: fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramSourceContext),
			Value: v1beta1.ArrayOrString{
				Type:      v1beta1.ParamTypeString,
				StringVal: path.Join("/workspace/source", *build.Spec.Source.ContextDir),
			},
		})
	} else {
		params = append(params, v1beta1.Param{
			Name: fmt.Sprintf("%s-%s", prefixParamsResultsVolumes, paramSourceContext),
			Value: v1beta1.ArrayOrString{
				Type:      v1beta1.ParamTypeString,
				StringVal: "/workspace/source",
			},
		})
	}

	expectedTaskRun.Spec.Params = params

	// Ensure a proper override of params between Build and BuildRun
	// A BuildRun can override a param as long as it was defined in the Build
	buildUserParams := overrideParams(build.Spec.ParamValues, buildRun.Spec.ParamValues)

	// list of params that collide with reserved system strategy parameters
	undesiredParams := []string{}

	// Append params to the TaskRun spec definition
	for _, p := range buildUserParams {

		if isReserved := IsSystemReservedParameter(p.Name); isReserved {
			undesiredParams = append(undesiredParams, p.Name)
		}

		buildParam := v1beta1.Param{
			Name: p.Name,
			Value: v1beta1.ArrayOrString{
				Type:      v1beta1.ParamTypeString,
				StringVal: p.Value,
			},
		}
		expectedTaskRun.Spec.Params = append(expectedTaskRun.Spec.Params, buildParam)
	}
	// if system parameters names are being use, fail the taskRun creation and update the condition message
	// with a custom error
	if len(undesiredParams) > 0 {
		return nil, fmt.Errorf("restricted parameters in use: %s", strings.Join(undesiredParams, ","))
	}

	// check if there are parameters from strategies where a value was never set, if this is the case,
	// then throw a custom error message.
	paramsWithoutValues := []string{}

StrategyParametersLoop:
	for _, strategyParam := range strategy.GetParameters() {
		if strategyParam.Default == nil {
			for _, p := range buildUserParams {
				if strategyParam.Name == p.Name {
					// go back to the outer loop
					continue StrategyParametersLoop
				}
			}
			paramsWithoutValues = append(paramsWithoutValues, strategyParam.Name)
		}
	}

	if len(paramsWithoutValues) > 0 {
		return nil, fmt.Errorf("parameters without a value definition: %s", strings.Join(paramsWithoutValues, ","))
	}

	return expectedTaskRun, nil
}

func effectiveTimeout(build *buildv1alpha1.Build, buildRun *buildv1alpha1.BuildRun) *metav1.Duration {
	if buildRun.Spec.Timeout != nil {
		return buildRun.Spec.Timeout

	} else if build.Spec.Timeout != nil {
		return build.Spec.Timeout
	}

	return nil
}

// isPropagatableAnnotation filters the last-applied-configuration annotation from kubectl because this would break the meaning of this annotation on the target object;
// also, annotations using our own custom resource domains are filtered out because we have no annotations with a semantic for both TaskRun and Pod
func isPropagatableAnnotation(key string) bool {
	return key != "kubectl.kubernetes.io/last-applied-configuration" &&
		!strings.HasPrefix(key, buildv1alpha1.ClusterBuildStrategyDomain+"/") &&
		!strings.HasPrefix(key, buildv1alpha1.BuildStrategyDomain+"/") &&
		!strings.HasPrefix(key, buildv1alpha1.BuildDomain+"/") &&
		!strings.HasPrefix(key, buildv1alpha1.BuildRunDomain+"/")
}

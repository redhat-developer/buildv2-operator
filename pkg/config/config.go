// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

const (
	contextTimeout = 300 * time.Second
	// A number in seconds to define a context Timeout
	// E.g. if 5 seconds is wanted, the CTX_TIMEOUT=5
	contextTimeoutEnvVar = "CTX_TIMEOUT"

	remoteArtifactsDefaultImage = "quay.io/quay/busybox:latest"
	remoteArtifactsEnvVar       = "REMOTE_ARTIFACTS_CONTAINER_IMAGE"

	// the Git image is built using ko which can replace environment variable values in the deployment, so once we decide to move
	// from environment variables to a ConfigMap, then we should move the container template, but retain the environment variable
	// (or make it an argument like Tekton)
	gitDefaultImage            = "quay.io/shipwright/git:latest"
	gitImageEnvVar             = "GIT_CONTAINER_IMAGE"
	gitContainerTemplateEnvVar = "GIT_CONTAINER_TEMPLATE"

	mutateImageDefaultImage            = "quay.io/shipwright/mutate-image:latest"
	mutateImageEnvVar                  = "MUTATE_IMAGE_CONTAINER_IMAGE"
	mutateImageContainerTemplateEnvVar = "MUTATE_IMAGE_CONTAINER_TEMPLATE"

	// Analog to the Git image, the bundle image is also created by ko
	bundleDefaultImage            = "quay.io/shipwright/bundle:latest"
	bundleImageEnvVar             = "BUNDLE_CONTAINER_IMAGE"
	bundleContainerTemplateEnvVar = "BUNDLE_CONTAINER_TEMPLATE"

	// environment variable to override the buckets
	metricBuildRunCompletionDurationBucketsEnvVar = "PROMETHEUS_BR_COMP_DUR_BUCKETS"
	metricBuildRunEstablishDurationBucketsEnvVar  = "PROMETHEUS_BR_EST_DUR_BUCKETS"
	metricBuildRunRampUpDurationBucketsEnvVar     = "PROMETHEUS_BR_RAMPUP_DUR_BUCKETS"

	// environment variable to enable prometheus metric labels
	prometheusEnabledLabelsEnvVar = "PROMETHEUS_ENABLED_LABELS"

	leaderElectionNamespaceDefault = "default"
	leaderElectionNamespaceEnvVar  = "BUILD_CONTROLLER_LEADER_ELECTION_NAMESPACE"

	leaseDurationEnvVar = "BUILD_CONTROLLER_LEASE_DURATION"
	renewDeadlineEnvVar = "BUILD_CONTROLLER_RENEW_DEADLINE"
	retryPeriodEnvVar   = "BUILD_CONTROLLER_RETRY_PERIOD"

	// environment variables for the controllers
	controllerBuildMaxConcurrentReconciles                = "BUILD_MAX_CONCURRENT_RECONCILES"
	controllerBuildRunMaxConcurrentReconciles             = "BUILDRUN_MAX_CONCURRENT_RECONCILES"
	controllerBuildStrategyMaxConcurrentReconciles        = "BUILDSTRATEGY_MAX_CONCURRENT_RECONCILES"
	controllerClusterBuildStrategyMaxConcurrentReconciles = "CLUSTERBUILDSTRATEGY_MAX_CONCURRENT_RECONCILES"

	// environment variables for the kube API
	kubeAPIBurst = "KUBE_API_BURST"
	kubeAPIQPS   = "KUBE_API_QPS"

	terminationLogPathDefault = "/dev/termination-log"
	terminationLogPathEnvVar  = "TERMINATION_LOG_PATH"
)

var (
	// arrays are not possible as constants
	metricBuildRunCompletionDurationBuckets = prometheus.LinearBuckets(50, 50, 10)
	metricBuildRunEstablishDurationBuckets  = []float64{0, 1, 2, 3, 5, 7, 10, 15, 20, 30}
	metricBuildRunRampUpDurationBuckets     = prometheus.LinearBuckets(0, 1, 10)

	root    = pointer.Int64Ptr(0)
	nonRoot = pointer.Int64Ptr(1000)
)

// Config hosts different parameters that
// can be set to use on the Build controllers
type Config struct {
	CtxTimeOut time.Duration
	GitContainerTemplate,
	MutateImageContainerTemplate corev1.Container
	BundleContainerTemplate       corev1.Container
	RemoteArtifactsContainerImage string
	TerminationLogPath            string
	Prometheus                    PrometheusConfig
	ManagerOptions                ManagerOptions
	Controllers                   Controllers
	KubeAPIOptions                KubeAPIOptions
}

// PrometheusConfig contains the specific configuration for the
type PrometheusConfig struct {
	BuildRunCompletionDurationBuckets []float64
	BuildRunEstablishDurationBuckets  []float64
	BuildRunRampUpDurationBuckets     []float64
	EnabledLabels                     []string
}

// ManagerOptions contains configurable options for the Shipwright build controller manager
type ManagerOptions struct {
	LeaderElectionNamespace string
	LeaseDuration           *time.Duration
	RenewDeadline           *time.Duration
	RetryPeriod             *time.Duration
}

// Controllers contains the options for the different controllers
type Controllers struct {
	Build                ControllerOptions
	BuildRun             ControllerOptions
	BuildStrategy        ControllerOptions
	ClusterBuildStrategy ControllerOptions
}

// ControllerOptions contains configurable options for a controller
type ControllerOptions struct {
	MaxConcurrentReconciles int
}

// KubeAPIOptions contains configurable options for the kube API client
type KubeAPIOptions struct {
	QPS   int
	Burst int
}

// NewDefaultConfig returns a new Config, with context timeout and default Kaniko image.
func NewDefaultConfig() *Config {
	return &Config{
		CtxTimeOut: contextTimeout,
		GitContainerTemplate: corev1.Container{
			Image: gitDefaultImage,
			Command: []string{
				"/ko-app/git",
			},
			SecurityContext: &corev1.SecurityContext{
				RunAsUser:  nonRoot,
				RunAsGroup: nonRoot,
			},
		},
		BundleContainerTemplate: corev1.Container{
			Image: bundleDefaultImage,
			Command: []string{
				"/ko-app/bundle",
			},
			SecurityContext: &corev1.SecurityContext{
				RunAsUser:  nonRoot,
				RunAsGroup: nonRoot,
			},
		},
		RemoteArtifactsContainerImage: remoteArtifactsDefaultImage,
		MutateImageContainerTemplate: corev1.Container{
			Image: mutateImageDefaultImage,
			Command: []string{
				"/ko-app/mutate-image",
			},
			// We explicitly define HOME=/tekton/home because this was always set in the
			// default configuration of Tekton until v0.24.0, see https://github.com/tektoncd/pipeline/pull/3878
			Env: []corev1.EnvVar{
				{
					Name:  "HOME",
					Value: "/tekton/home",
				},
			},
			// The mutate image step runs after the build strategy steps where an arbitrary
			// user could have been used to write the result files for the image digest. The
			// mutate image step will overwrite the image digest file. To be able to do this
			// in all possible scenarios, we run this step as root with DAC_OVERRIDE
			// capability.
			SecurityContext: &corev1.SecurityContext{
				RunAsUser: root,
				Capabilities: &corev1.Capabilities{
					Add: []corev1.Capability{
						"DAC_OVERRIDE",
					},
				},
			},
		},
		Prometheus: PrometheusConfig{
			BuildRunCompletionDurationBuckets: metricBuildRunCompletionDurationBuckets,
			BuildRunEstablishDurationBuckets:  metricBuildRunEstablishDurationBuckets,
			BuildRunRampUpDurationBuckets:     metricBuildRunRampUpDurationBuckets,
		},
		ManagerOptions: ManagerOptions{
			LeaderElectionNamespace: leaderElectionNamespaceDefault,
		},
		Controllers: Controllers{
			Build: ControllerOptions{
				MaxConcurrentReconciles: 0,
			},
			BuildRun: ControllerOptions{
				MaxConcurrentReconciles: 0,
			},
			BuildStrategy: ControllerOptions{
				MaxConcurrentReconciles: 0,
			},
			ClusterBuildStrategy: ControllerOptions{
				MaxConcurrentReconciles: 0,
			},
		},
		KubeAPIOptions: KubeAPIOptions{
			QPS:   0,
			Burst: 0,
		},
		TerminationLogPath: terminationLogPathDefault,
	}
}

// SetConfigFromEnv updates the configuration managed by environment variables.
func (c *Config) SetConfigFromEnv() error {
	if timeout := os.Getenv(contextTimeoutEnvVar); timeout != "" {
		i, err := strconv.Atoi(timeout)
		if err != nil {
			return err
		}
		c.CtxTimeOut = time.Duration(i) * time.Second
	}

	if gitContainerTemplate := os.Getenv(gitContainerTemplateEnvVar); gitContainerTemplate != "" {
		c.GitContainerTemplate = corev1.Container{}
		if err := json.Unmarshal([]byte(gitContainerTemplate), &c.GitContainerTemplate); err != nil {
			return err
		}
		if c.GitContainerTemplate.Image == "" {
			c.GitContainerTemplate.Image = gitDefaultImage
		}
	}

	// the dedicated environment variable for the image overwrites what is defined in the git container template
	if gitImage := os.Getenv(gitImageEnvVar); gitImage != "" {
		c.GitContainerTemplate.Image = gitImage
	}

	if mutateImageContainerTemplate := os.Getenv(mutateImageContainerTemplateEnvVar); mutateImageContainerTemplate != "" {
		c.GitContainerTemplate = corev1.Container{}
		if err := json.Unmarshal([]byte(mutateImageContainerTemplate), &c.MutateImageContainerTemplate); err != nil {
			return err
		}
		if c.MutateImageContainerTemplate.Image == "" {
			c.MutateImageContainerTemplate.Image = mutateImageDefaultImage
		}
	}

	// the dedicated environment variable for the image overwrites
	// what is defined in the mutate image container template
	if mutateImage := os.Getenv(mutateImageEnvVar); mutateImage != "" {
		c.MutateImageContainerTemplate.Image = mutateImage
	}

	if bundleContainerTemplate := os.Getenv(bundleContainerTemplateEnvVar); bundleContainerTemplate != "" {
		c.BundleContainerTemplate = corev1.Container{}
		if err := json.Unmarshal([]byte(bundleContainerTemplate), &c.BundleContainerTemplate); err != nil {
			return err
		}
		if c.BundleContainerTemplate.Image == "" {
			c.BundleContainerTemplate.Image = bundleDefaultImage
		}
	}

	// the dedicated environment variable for the image overwrites what is defined in the bundle container template
	if bundleImage := os.Getenv(bundleImageEnvVar); bundleImage != "" {
		c.BundleContainerTemplate.Image = bundleImage
	}

	if remoteArtifactsImage := os.Getenv(remoteArtifactsEnvVar); remoteArtifactsImage != "" {
		c.RemoteArtifactsContainerImage = remoteArtifactsImage
	}

	if err := updateBucketsConfig(&c.Prometheus.BuildRunCompletionDurationBuckets, metricBuildRunCompletionDurationBucketsEnvVar); err != nil {
		return err
	}

	if err := updateBucketsConfig(&c.Prometheus.BuildRunEstablishDurationBuckets, metricBuildRunEstablishDurationBucketsEnvVar); err != nil {
		return err
	}

	if err := updateBucketsConfig(&c.Prometheus.BuildRunRampUpDurationBuckets, metricBuildRunRampUpDurationBucketsEnvVar); err != nil {
		return err
	}

	c.Prometheus.EnabledLabels = strings.Split(os.Getenv(prometheusEnabledLabelsEnvVar), ",")

	if leaderElectionNamespace := os.Getenv(leaderElectionNamespaceEnvVar); leaderElectionNamespace != "" {
		c.ManagerOptions.LeaderElectionNamespace = leaderElectionNamespace
	}

	if err := updateBuildControllerDurationOption(&c.ManagerOptions.LeaseDuration, leaseDurationEnvVar); err != nil {
		return err
	}

	if err := updateBuildControllerDurationOption(&c.ManagerOptions.RenewDeadline, renewDeadlineEnvVar); err != nil {
		return err
	}

	if err := updateBuildControllerDurationOption(&c.ManagerOptions.RetryPeriod, retryPeriodEnvVar); err != nil {
		return err
	}

	// controller settings
	if err := updateIntOption(&c.Controllers.Build.MaxConcurrentReconciles, controllerBuildMaxConcurrentReconciles); err != nil {
		return err
	}
	if err := updateIntOption(&c.Controllers.BuildRun.MaxConcurrentReconciles, controllerBuildRunMaxConcurrentReconciles); err != nil {
		return err
	}
	if err := updateIntOption(&c.Controllers.BuildStrategy.MaxConcurrentReconciles, controllerBuildStrategyMaxConcurrentReconciles); err != nil {
		return err
	}
	if err := updateIntOption(&c.Controllers.ClusterBuildStrategy.MaxConcurrentReconciles, controllerClusterBuildStrategyMaxConcurrentReconciles); err != nil {
		return err
	}

	// kube API settings
	if err := updateIntOption(&c.KubeAPIOptions.Burst, kubeAPIBurst); err != nil {
		return err
	}
	if err := updateIntOption(&c.KubeAPIOptions.QPS, kubeAPIQPS); err != nil {
		return err
	}

	if terminationLogPath := os.Getenv(terminationLogPathEnvVar); terminationLogPath != "" {
		c.TerminationLogPath = terminationLogPath
	}

	return nil
}

func stringToFloat64Array(strings []string) ([]float64, error) {
	floats := make([]float64, len(strings))

	for i, string := range strings {
		float, err := strconv.ParseFloat(string, 64)
		if err != nil {
			return nil, err
		}
		floats[i] = float
	}

	return floats, nil
}

func updateBucketsConfig(buckets *[]float64, envVarName string) error {
	if values, found := os.LookupEnv(envVarName); found {
		floats, err := stringToFloat64Array(strings.Split(values, ","))
		if err != nil {
			return err
		}

		*buckets = floats
	}

	return nil
}

func updateBuildControllerDurationOption(d **time.Duration, envVarName string) error {
	if value := os.Getenv(envVarName); value != "" {
		valueDuration, err := time.ParseDuration(value)
		if err != nil {
			return err
		}

		*d = &valueDuration
	}

	return nil
}

func updateIntOption(i *int, envVarName string) error {
	if value := os.Getenv(envVarName); value != "" {
		intValue, err := strconv.ParseInt(value, 10, 0)
		if err != nil {
			return err
		}
		*i = int(intValue)
	}

	return nil
}

// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"time"

	gomegaConfig "github.com/onsi/ginkgo/config"
	tektonClient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	buildapis "github.com/shipwright-io/build/pkg/apis"
	buildClient "github.com/shipwright-io/build/pkg/client/clientset/versioned"
	"github.com/shipwright-io/build/pkg/ctxlog"
	"github.com/shipwright-io/build/test"
	// from https://github.com/kubernetes/client-go/issues/345
)

var (
	namespaceCounter int32
)

// TestBuild wraps all required clients to run integration
// tests and also the namespace and operator channel used
// per each test case
type TestBuild struct {
	// TODO: Adding specific field for polling here, interval and timeout
	// but I think we need a small refactoring to make them global for all
	// tests under /test dir
	Interval                 time.Duration
	TimeOut                  time.Duration
	KubeConfig               *rest.Config
	Clientset                *kubernetes.Clientset
	Namespace                string
	StopBuildControllers     chan struct{}
	BuildClientSet           *buildClient.Clientset
	PipelineClientSet        *tektonClient.Clientset
	Catalog                  test.Catalog
	Context                  context.Context
	BuildControllerLogBuffer *bytes.Buffer
	Scheme                   *runtime.Scheme
}

// NewTestBuild returns an initialized instance of TestBuild
func NewTestBuild() (*TestBuild, error) {
	namespaceID := gomegaConfig.GinkgoConfig.ParallelNode*200 + int(atomic.AddInt32(&namespaceCounter, 1))
	testNamespace := "test-build-" + strconv.Itoa(namespaceID)

	// Scheme needed to search events by object
	// Add additional APIs here if tests need to search events for other resource types
	scheme := runtime.NewScheme()
	err := corev1.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}
	err = buildapis.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}

	logBuffer := &bytes.Buffer{}
	l := ctxlog.NewLoggerTo(logBuffer, testNamespace)

	ctx := ctxlog.NewParentContext(l)

	kubeConfig, restConfig, err := KubeConfig()
	if err != nil {
		return nil, err
	}

	// clientSet is required to communicate with our CRDs objects
	// see https://www.openshift.com/blog/kubernetes-deep-dive-code-generation-customresources
	buildClientSet, err := buildClient.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	pipelineClientSet, err := tektonClient.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	return &TestBuild{
		// TODO: interval and timeout can be configured via ENV vars
		Interval:                 time.Second * 3,
		TimeOut:                  time.Second * 180,
		KubeConfig:               restConfig,
		Clientset:                kubeConfig,
		Namespace:                testNamespace,
		BuildClientSet:           buildClientSet,
		PipelineClientSet:        pipelineClientSet,
		Context:                  ctx,
		BuildControllerLogBuffer: logBuffer,
		Scheme:                   scheme,
	}, nil
}

// KubeConfig returns all required clients to speak with
// the k8s API
func KubeConfig() (*kubernetes.Clientset, *rest.Config, error) {
	location := os.Getenv("KUBECONFIG")
	if location == "" {
		location = filepath.Join(os.Getenv("HOME"), ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", location)
	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, nil, err
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}

	return clientset, config, nil
}

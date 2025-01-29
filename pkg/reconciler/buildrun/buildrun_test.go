// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package buildrun_test

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	pipelineapi "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	knativeapi "knative.dev/pkg/apis"
	knativev1 "knative.dev/pkg/apis/duck/v1"
	crc "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/shipwright-io/build/pkg/apis"
	build "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/controller/fakes"
	buildrunctl "github.com/shipwright-io/build/pkg/reconciler/buildrun"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources"
	test "github.com/shipwright-io/build/test/v1beta1_samples"
)

var _ = Describe("Reconcile BuildRun", func() {
	var (
		manager                                                *fakes.FakeManager
		reconciler                                             reconcile.Reconciler
		taskRunRequest, buildRunRequest                        reconcile.Request
		client                                                 *fakes.FakeClient
		ctl                                                    test.Catalog
		buildSample                                            *build.Build
		buildRunSample                                         *build.BuildRun
		taskRunSample                                          *pipelineapi.TaskRun
		statusWriter                                           *fakes.FakeStatusWriter
		taskRunName, buildRunName, buildName, strategyName, ns string
	)

	// returns a reconcile.Request based on an resource name and namespace
	newReconcileRequest := func(name string, ns string) reconcile.Request {
		return reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      name,
				Namespace: ns,
			},
		}
	}

	// Basic stubs that simulate the output of all client calls in the Reconciler logic.
	// This applies only for a Build and BuildRun client get.
	getClientStub := func(_ context.Context, nn types.NamespacedName, object crc.Object, getOptions ...crc.GetOption) error {
		switch object := object.(type) {
		case *build.Build:
			buildSample.DeepCopyInto(object)
			return nil
		case *build.BuildRun:
			buildRunSample.DeepCopyInto(object)
			return nil
		case *pipelineapi.TaskRun:
			taskRunSample.DeepCopyInto(object)
			return nil
		}
		return k8serrors.NewNotFound(schema.GroupResource{}, nn.Name)
	}

	BeforeEach(func() {
		strategyName = "foobar-strategy"
		buildName = "foobar-build"
		buildRunName = "foobar-buildrun"
		taskRunName = "foobar-buildrun-p8nts"
		ns = "default"

		// ensure resources are added to the Scheme
		// via the manager and initialize the fake Manager
		apis.AddToScheme(scheme.Scheme)
		manager = &fakes.FakeManager{}
		manager.GetSchemeReturns(scheme.Scheme)

		// initialize the fake client and let the
		// client know on the stubs when get calls are executed
		client = &fakes.FakeClient{}
		client.GetCalls(getClientStub)

		// initialize the fake status writer, this is needed for
		// all status updates during reconciliation
		statusWriter = &fakes.FakeStatusWriter{}
		client.StatusCalls(func() crc.StatusWriter { return statusWriter })
		manager.GetClientReturns(client)

		// init the Build resource, this never change throughout this test suite
		buildSample = ctl.DefaultBuild(buildName, strategyName, build.ClusterBuildStrategyKind)
		buildRunSample = ctl.DefaultBuildRun(buildRunName, buildName)
		taskRunSample = ctl.DefaultTaskRunWithStatus(taskRunName, buildRunName, ns, corev1.ConditionTrue, "Succeeded")
	})

	// JustBeforeEach will always execute just before the It() specs,
	// this ensures that overrides on the BuildRun resource can happen under each
	// Context() BeforeEach() block
	JustBeforeEach(func() {
		reconciler = buildrunctl.NewReconciler(config.NewDefaultConfig(), manager, controllerutil.SetControllerReference)
	})

	Describe("Reconciling", func() {
		Context("from an existing TaskRun resource", func() {
			BeforeEach(func() {

				// Generate a new Reconcile Request using the existing TaskRun name and namespace
				taskRunRequest = newReconcileRequest(taskRunName, ns)

				// initialize a TaskRun, we need this to fake the existence of a Tekton TaskRun
				taskRunSample = ctl.DefaultTaskRunWithStatus(taskRunName, buildRunName, ns, corev1.ConditionTrue, "Succeeded")

				// initialize a BuildRun, we need this to fake the existence of a BuildRun
				buildRunSample = ctl.DefaultBuildRun(buildRunName, buildName)
			})

			It("is able to retrieve a TaskRun, Build and a BuildRun", func() {

				// stub the existence of a Build, BuildRun and
				// a TaskRun via the getClientStub, therefore we
				// expect the Reconcile to Succeed because all resources
				// exist
				result, err := reconciler.Reconcile(context.TODO(), taskRunRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(reconcile.Result{}).To(Equal(result))
				Expect(client.GetCallCount()).To(Equal(2))
			})
			It("does not fail when the BuildRun does not exist", func() {

				// override the initial getClientStub, and generate a new stub
				// that only contains a Build and TaskRun, none BuildRun
				stubGetCalls := ctl.StubBuildAndTaskRun(buildSample, taskRunSample)
				client.GetCalls(stubGetCalls)

				result, err := reconciler.Reconcile(context.TODO(), taskRunRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(reconcile.Result{}).To(Equal(result))
				Expect(client.GetCallCount()).To(Equal(3))
			})
			It("does not fail when the Build does not exist", func() {

				// override the initial getClientStub, and generate a new stub
				// that only contains a BuildRun and TaskRun, none Build
				stubGetCalls := ctl.StubBuildRunAndTaskRun(buildRunSample, taskRunSample)
				client.GetCalls(stubGetCalls)

				result, err := reconciler.Reconcile(context.TODO(), taskRunRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(reconcile.Result{}).To(Equal(result))
				Expect(client.GetCallCount()).To(Equal(2))
			})
			It("updates the BuildRun status", func() {

				// generated stub that asserts the BuildRun status fields when
				// status updates for a BuildRun take place
				statusCall := ctl.StubBuildRunStatus(
					"Succeeded",
					&taskRunName,
					build.Condition{
						Type:   build.Succeeded,
						Reason: "Succeeded",
						Status: corev1.ConditionTrue,
					},
					corev1.ConditionTrue,
					buildSample.Spec,
					false,
				)
				statusWriter.UpdateCalls(statusCall)

				// Assert for none errors while we exit the Reconcile
				// after updating the BuildRun status with the existing
				// TaskRun one
				result, err := reconciler.Reconcile(context.TODO(), taskRunRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(reconcile.Result{}).To(Equal(result))
				Expect(client.GetCallCount()).To(Equal(2))
				Expect(client.StatusCallCount()).To(Equal(1))
			})

			It("does not update the BuildRun status if the BuildRun is already completed", func() {
				buildRunSample = ctl.BuildRunWithSAGenerate(buildRunName, buildName)
				buildRunSample.Status.CompletionTime = &metav1.Time{
					Time: time.Now(),
				}

				client.GetCalls(ctl.StubBuildRunAndTaskRun(buildRunSample, taskRunSample))

				// Call the reconciler
				_, err := reconciler.Reconcile(context.TODO(), taskRunRequest)

				// Expect no error
				Expect(err).ToNot(HaveOccurred())

				// Expect no delete call and no status update
				Expect(client.GetCallCount()).To(Equal(2))
				Expect(client.DeleteCallCount()).To(Equal(0))
				Expect(client.StatusCallCount()).To(Equal(0))
			})

			It("deletes a generated service account when the task run ends", func() {

				// setup a buildrun to use a generated service account
				buildSample = ctl.DefaultBuild(buildName, "foobar-strategy", build.ClusterBuildStrategyKind)
				buildRunSample = ctl.BuildRunWithSAGenerate(buildRunSample.Name, buildName)
				buildRunSample.Status.BuildSpec = &buildSample.Spec
				buildRunSample.Labels = make(map[string]string)
				buildRunSample.Labels[build.LabelBuild] = buildName

				// Override Stub get calls to include a service account
				client.GetCalls(ctl.StubBuildRunGetWithTaskRunAndSA(
					buildSample,
					buildRunSample,
					taskRunSample,
					ctl.DefaultServiceAccount(buildRunSample.Name)),
				)

				// Call the reconciler
				_, err := reconciler.Reconcile(context.TODO(), taskRunRequest)

				// Expect no error
				Expect(err).ToNot(HaveOccurred())

				// Expect one delete call for the service account
				Expect(client.DeleteCallCount()).To(Equal(1))
				_, obj, _ := client.DeleteArgsForCall(0)
				serviceAccount, castSuccessful := obj.(*corev1.ServiceAccount)
				Expect(castSuccessful).To(BeTrue())
				Expect(serviceAccount.Name).To(Equal(buildRunSample.Name))
				Expect(serviceAccount.Namespace).To(Equal(buildRunSample.Namespace))
			})
		})

		Context("from an existing TaskRun with Conditions", func() {
			BeforeEach(func() {

				// Generate a new Reconcile Request using the existing TaskRun name and namespace
				taskRunRequest = newReconcileRequest(taskRunName, ns)

				// initialize a BuildRun, we need this to fake the existence of a BuildRun
				buildRunSample = ctl.DefaultBuildRun(buildRunName, buildName)
			})

			// Docs on the TaskRun conditions can be found here
			// https://github.com/tektoncd/pipeline/blob/main/docs/taskruns.md#monitoring-execution-status
			It("updates the BuildRun status with a PENDING reason", func() {

				// initialize a TaskRun, we need this to fake the existence of a Tekton TaskRun
				taskRunSample = ctl.DefaultTaskRunWithStatus(taskRunName, buildRunName, ns, corev1.ConditionUnknown, "Pending")

				// Stub that asserts the BuildRun status fields when
				// Status updates for a BuildRun take place
				statusCall := ctl.StubBuildRunStatus(
					"Pending",
					&taskRunName,
					build.Condition{
						Type:   build.Succeeded,
						Reason: "Pending",
						Status: corev1.ConditionUnknown,
					},
					corev1.ConditionUnknown,
					buildSample.Spec,
					false,
				)
				statusWriter.UpdateCalls(statusCall)

				// Assert for none errors while we exit the Reconcile
				// after updating the BuildRun status with the existing
				// TaskRun one
				result, err := reconciler.Reconcile(context.TODO(), taskRunRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(reconcile.Result{}).To(Equal(result))
				Expect(client.GetCallCount()).To(Equal(2))
				Expect(client.StatusCallCount()).To(Equal(1))
			})

			It("updates the BuildRun status with a RUNNING reason", func() {

				taskRunSample = ctl.DefaultTaskRunWithStatus(taskRunName, buildRunName, ns, corev1.ConditionUnknown, "Running")

				// Stub that asserts the BuildRun status fields when
				// Status updates for a BuildRun take place
				statusCall := ctl.StubBuildRunStatus(
					"Running",
					&taskRunName,
					build.Condition{
						Type:   build.Succeeded,
						Reason: "Running",
						Status: corev1.ConditionUnknown,
					},
					corev1.ConditionUnknown,
					buildSample.Spec,
					false,
				)
				statusWriter.UpdateCalls(statusCall)

				result, err := reconciler.Reconcile(context.TODO(), taskRunRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(reconcile.Result{}).To(Equal(result))
				Expect(client.GetCallCount()).To(Equal(2))
				Expect(client.StatusCallCount()).To(Equal(1))
			})

			It("updates the BuildRun status with a SUCCEEDED reason", func() {

				taskRunSample = ctl.DefaultTaskRunWithStatus(taskRunName, buildRunName, ns, corev1.ConditionTrue, "Succeeded")

				// Stub that asserts the BuildRun status fields when
				// Status updates for a BuildRun take place
				statusCall := ctl.StubBuildRunStatus(
					"Succeeded",
					&taskRunName,
					build.Condition{
						Type:   build.Succeeded,
						Reason: "Succeeded",
						Status: corev1.ConditionTrue,
					},
					corev1.ConditionTrue,
					buildSample.Spec,
					false,
				)
				statusWriter.UpdateCalls(statusCall)

				result, err := reconciler.Reconcile(context.TODO(), taskRunRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(reconcile.Result{}).To(Equal(result))
				Expect(client.GetCallCount()).To(Equal(2))
				Expect(client.StatusCallCount()).To(Equal(1))
			})

			It("should recognize the BuildRun is canceled", func() {
				// set cancel
				buildRunSampleCopy := buildRunSample.DeepCopy()
				buildRunSampleCopy.Spec.State = build.BuildRunRequestedStatePtr(build.BuildRunStateCancel)

				taskRunSample = ctl.DefaultTaskRunWithStatus(taskRunName, buildRunName, ns, corev1.ConditionUnknown, "Running")

				// Override Stub get calls to include a completed TaskRun
				// and a Pod with one initContainer Status
				client.GetCalls(ctl.StubBuildCRDsPodAndTaskRun(
					buildSample,
					buildRunSampleCopy,
					ctl.DefaultServiceAccount("foobar"),
					ctl.DefaultClusterBuildStrategy(),
					ctl.DefaultNamespacedBuildStrategy(),
					taskRunSample,
					ctl.PodWithInitContainerStatus("foobar", "init-foobar")),
				)

				cancelPatchCalled := false
				cancelUpdateCalled := false
				// override the updateClientStub so we can see the update on the BuildRun condition
				stubUpdateCalls := func(_ context.Context, object crc.Object, opts ...crc.SubResourceUpdateOption) error {
					switch v := object.(type) {
					case *build.BuildRun:
						c := v.Status.GetCondition(build.Succeeded)
						if c != nil && c.Reason == build.BuildRunStateCancel && c.Status == corev1.ConditionFalse {
							cancelUpdateCalled = true
						}

					}
					return nil
				}
				statusWriter.UpdateCalls(stubUpdateCalls)
				stubPatchCalls := func(_ context.Context, object crc.Object, patch crc.Patch, opts ...crc.PatchOption) error {
					switch v := object.(type) {
					case *pipelineapi.TaskRun:
						if v.Name == taskRunSample.Name {
							cancelPatchCalled = true
						}
					}
					return nil
				}
				client.PatchCalls(stubPatchCalls)

				_, err := reconciler.Reconcile(context.TODO(), buildRunRequest)
				Expect(err).To(BeNil())
				Expect(resources.IsClientStatusUpdateError(err)).To(BeFalse())
				Expect(cancelPatchCalled).To(BeTrue())

				// actually set value the patch would have set (but we overrode above)
				// for next call
				taskRunSample.Spec.Status = pipelineapi.TaskRunSpecStatusCancelled
				taskRunSample.Status.Conditions = knativev1.Conditions{
					{
						Type:   knativeapi.ConditionSucceeded,
						Reason: string(pipelineapi.TaskRunReasonCancelled),
						Status: corev1.ConditionFalse,
					},
				}

				_, err = reconciler.Reconcile(context.TODO(), buildRunRequest)
				Expect(err).To(BeNil())
				Expect(resources.IsClientStatusUpdateError(err)).To(BeFalse())
				Expect(cancelUpdateCalled).To(BeTrue())
			})

			It("updates the BuildRun status when a FALSE status occurs", func() {

				taskRunSample = ctl.DefaultTaskRunWithFalseStatus(taskRunName, buildRunName, ns)

				// Based on the current buildRun controller, if the TaskRun condition.Status
				// is FALSE, we will then populate our buildRun.Status.Reason with the
				// TaskRun condition.Message, rather than the condition.Reason
				statusCall := ctl.StubBuildRunStatus(
					"some message",
					&taskRunName,
					build.Condition{
						Type:   build.Succeeded,
						Reason: "something bad happened",
						Status: corev1.ConditionFalse,
					},
					corev1.ConditionFalse,
					buildSample.Spec,
					false,
				)
				statusWriter.UpdateCalls(statusCall)

				result, err := reconciler.Reconcile(context.TODO(), taskRunRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(reconcile.Result{}).To(Equal(result))
			})

			It("does not break the reconcile when a taskrun pod initcontainers are not ready", func() {
				taskRunSample = ctl.TaskRunWithCompletionAndStartTime(taskRunName, buildRunName, ns)

				buildRunSample = ctl.BuildRunWithBuildSnapshot(buildRunName, buildName)

				// Override Stub get calls to include a completed TaskRun
				// and a Pod with one initContainer Status
				client.GetCalls(ctl.StubBuildCRDsPodAndTaskRun(
					buildSample,
					buildRunSample,
					ctl.DefaultServiceAccount("foobar"),
					ctl.DefaultClusterBuildStrategy(),
					ctl.DefaultNamespacedBuildStrategy(),
					taskRunSample,
					ctl.PodWithInitContainerStatus("foobar", "init-foobar")),
				)

				result, err := reconciler.Reconcile(context.TODO(), taskRunRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(reconcile.Result{}).To(Equal(result))

				// Three client calls because based on the Stub, we should
				// trigger a call to get the related TaskRun pod.
				Expect(client.GetCallCount()).To(Equal(3))
			})

			It("does not break the reconcile when a failed taskrun has a pod with no failed container", func() {
				buildRunSample = ctl.BuildRunWithBuildSnapshot(buildRunName, buildName)
				taskRunSample = &pipelineapi.TaskRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      taskRunName,
						Namespace: ns,
						Labels:    map[string]string{"buildrun.shipwright.io/name": buildRunName},
					},
					Spec: pipelineapi.TaskRunSpec{},
					Status: pipelineapi.TaskRunStatus{
						TaskRunStatusFields: pipelineapi.TaskRunStatusFields{
							PodName: "foobar",
							CompletionTime: &metav1.Time{
								Time: time.Now(),
							},
							StartTime: &metav1.Time{
								Time: time.Now(),
							},
						},
						Status: knativev1.Status{
							Conditions: knativev1.Conditions{
								{
									Type:    knativeapi.ConditionSucceeded,
									Reason:  string(pipelineapi.TaskRunReasonFailed),
									Status:  corev1.ConditionFalse,
									Message: "some message",
								},
							},
						},
					},
				}

				client.GetCalls(ctl.StubBuildCRDsPodAndTaskRun(
					buildSample,
					buildRunSample,
					ctl.DefaultServiceAccount("foobar"),
					ctl.DefaultClusterBuildStrategy(),
					ctl.DefaultNamespacedBuildStrategy(),
					taskRunSample,
					&corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name: "foobar",
						},
						Status: corev1.PodStatus{},
					},
				))

				// Verify issue #591 by checking that Reconcile does not
				// fail with a panic due to a nil pointer dereference:
				// The pod has no container details and therefore the
				// look-up logic will find no container (result is nil).
				result, err := reconciler.Reconcile(context.TODO(), taskRunRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(reconcile.Result{}).To(Equal(result))
			})
		})

		Context("from an existing BuildRun resource", func() {
			var (
				saName           string
				emptyTaskRunName *string
			)
			BeforeEach(func() {
				saName = "foobar-sa"

				// Generate a new Reconcile Request using the existing BuildRun name and namespace
				buildRunRequest = newReconcileRequest(buildRunName, ns)

				// override the BuildRun resource to use a BuildRun with a specified
				// serviceaccount
				buildRunSample = ctl.BuildRunWithSA(buildRunName, buildName, saName)
			})

			It("should recognize the BuildRun is canceled even with TaskRun missing", func() {
				// set cancel
				buildRunSampleCopy := buildRunSample.DeepCopy()
				buildRunSampleCopy.Spec.State = build.BuildRunRequestedStatePtr(build.BuildRunStateCancel)

				client.GetCalls(ctl.StubBuildCRDs(
					buildSample,
					buildRunSampleCopy,
					ctl.DefaultServiceAccount("foobar"),
					ctl.DefaultClusterBuildStrategy(),
					ctl.DefaultNamespacedBuildStrategy(),
				))

				cancelUpdateCalled := false
				// override the updateClientStub so we can see the update on the BuildRun condition
				stubUpdateCalls := func(_ context.Context, object crc.Object, opts ...crc.SubResourceUpdateOption) error {
					switch v := object.(type) {
					case *build.BuildRun:
						c := v.Status.GetCondition(build.Succeeded)
						if c != nil && c.Reason == build.BuildRunStateCancel {
							cancelUpdateCalled = true
						}

					}
					return nil
				}
				statusWriter.UpdateCalls(stubUpdateCalls)

				_, err := reconciler.Reconcile(context.TODO(), buildRunRequest)
				Expect(err).To(BeNil())
				Expect(resources.IsClientStatusUpdateError(err)).To(BeFalse())

				Expect(cancelUpdateCalled).To(BeTrue())
			})

			It("should return none error and stop reconciling if referenced Build is not found", func() {
				buildRunSample = ctl.BuildRunWithoutSA(buildRunName, buildName)

				// override the initial getClientStub, and generate a new stub
				// that only contains a Buildrun
				stubGetCalls := func(_ context.Context, nn types.NamespacedName, object crc.Object, getOptions ...crc.GetOption) error {
					switch object := object.(type) {
					case *build.Build:
						return k8serrors.NewNotFound(schema.GroupResource{}, nn.Name)
					case *build.BuildRun:
						buildRunSample.DeepCopyInto(object)
						return nil
					}
					return k8serrors.NewNotFound(schema.GroupResource{}, nn.Name)
				}
				client.GetCalls(stubGetCalls)

				_, err := reconciler.Reconcile(context.TODO(), buildRunRequest)
				Expect(err).To(BeNil())
				Expect(resources.IsClientStatusUpdateError(err)).To(BeFalse())
			})

			It("should return an error and continue reconciling if referenced Build is not found and the status update fails", func() {
				buildRunSample = ctl.BuildRunWithoutSA(buildRunName, buildName)

				// override the initial getClientStub, and generate a new stub
				// that only contains a BuildRun
				stubGetCalls := func(_ context.Context, nn types.NamespacedName, object crc.Object, _ ...crc.GetOption) error {
					switch object := object.(type) {
					case *build.Build:
						return k8serrors.NewNotFound(schema.GroupResource{}, nn.Name)
					case *build.BuildRun:
						buildRunSample.DeepCopyInto(object)
						return nil
					}
					return k8serrors.NewNotFound(schema.GroupResource{}, nn.Name)
				}
				client.GetCalls(stubGetCalls)

				statusWriter.UpdateCalls(func(_ context.Context, object crc.Object, _ ...crc.SubResourceUpdateOption) error {
					switch buildRun := object.(type) {
					case *build.BuildRun:
						if buildRun.Status.IsFailed(build.Succeeded) {
							return fmt.Errorf("failed miserably")
						}
					}
					return nil
				})

				_, err := reconciler.Reconcile(context.TODO(), buildRunRequest)
				Expect(err).ToNot(BeNil())
				Expect(resources.IsClientStatusUpdateError(err)).To(BeTrue())
			})

			It("fails on a TaskRun creation due to service account not found", func() {
				// override the initial getClientStub, and generate a new stub
				// that only contains a Build and Buildrun, none TaskRun
				stubGetCalls := func(_ context.Context, nn types.NamespacedName, object crc.Object, _ ...crc.GetOption) error {
					switch object := object.(type) {
					case *build.Build:
						buildSample.DeepCopyInto(object)
						return nil
					case *build.BuildRun:
						buildRunSample.DeepCopyInto(object)
						return nil
					}
					return k8serrors.NewNotFound(schema.GroupResource{}, nn.Name)
				}

				client.GetCalls(stubGetCalls)

				// Stub that asserts the BuildRun status fields when
				// Status updates for a BuildRun take place
				statusCall := ctl.StubBuildRunStatus(
					fmt.Sprintf("service account %s not found", saName),
					emptyTaskRunName,
					build.Condition{
						Type:   build.Succeeded,
						Reason: "ServiceAccountNotFound",
						Status: corev1.ConditionFalse,
					},
					corev1.ConditionFalse,
					buildSample.Spec,
					true,
				)
				statusWriter.UpdateCalls(statusCall)

				// we mark the BuildRun as Failed and do not reconcile again
				_, err := reconciler.Reconcile(context.TODO(), buildRunRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(client.GetCallCount()).To(Equal(4))
				Expect(client.StatusCallCount()).To(Equal(2))
			})

			It("fails on a TaskRun creation due to issues when retrieving the service account", func() {
				// override the initial getClientStub, and generate a new stub
				// that only contains a Build, BuildRun and a random error when
				// retrieving a service account
				stubGetCalls := func(_ context.Context, nn types.NamespacedName, object crc.Object, _ ...crc.GetOption) error {
					switch object := object.(type) {
					case *build.Build:
						buildSample.DeepCopyInto(object)
						return nil
					case *build.BuildRun:
						buildRunSample.DeepCopyInto(object)
						return nil
					case *corev1.ServiceAccount:
						return fmt.Errorf("something wrong happen")
					}
					return k8serrors.NewNotFound(schema.GroupResource{}, nn.Name)
				}

				client.GetCalls(stubGetCalls)

				// we reconcile again on system call errors
				_, err := reconciler.Reconcile(context.TODO(), buildRunRequest)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("something wrong happen"))
				Expect(client.GetCallCount()).To(Equal(4))
				Expect(client.StatusCallCount()).To(Equal(1))
			})

			It("fails on a TaskRun creation due to namespaced buildstrategy not found", func() {
				// override the Build to use a namespaced BuildStrategy
				buildSample = ctl.DefaultBuild(buildName, strategyName, build.NamespacedBuildStrategyKind)

				// Override Stub get calls to include a service account
				// but none BuildStrategy
				client.GetCalls(ctl.StubBuildRunGetWithSA(
					buildSample,
					buildRunSample,
					ctl.DefaultServiceAccount(saName)),
				)

				// Stub that asserts the BuildRun status fields when
				// Status updates for a BuildRun take place
				statusCall := ctl.StubBuildRunStatus(
					fmt.Sprintf(" %q not found", strategyName),
					emptyTaskRunName,
					build.Condition{
						Type:   build.Succeeded,
						Reason: "BuildStrategyNotFound",
						Status: corev1.ConditionFalse,
					},
					corev1.ConditionFalse,
					buildSample.Spec,
					true,
				)
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(context.TODO(), buildRunRequest)
				// we mark the BuildRun as Failed and do not reconcile again
				Expect(err).ToNot(HaveOccurred())
			})

			It("fails on a TaskRun creation due to issues when retrieving the buildstrategy", func() {
				// override the Build to use a namespaced BuildStrategy
				buildSample = ctl.DefaultBuild(buildName, strategyName, build.NamespacedBuildStrategyKind)

				// stub get calls so that on namespaced strategy retrieval, we throw a random error
				stubGetCalls := func(_ context.Context, nn types.NamespacedName, object crc.Object, _ ...crc.GetOption) error {
					switch object := object.(type) {
					case *build.Build:
						buildSample.DeepCopyInto(object)
						return nil
					case *build.BuildRun:
						buildRunSample.DeepCopyInto(object)
						return nil
					case *corev1.ServiceAccount:
						ctl.DefaultServiceAccount(saName).DeepCopyInto(object)
						return nil
					case *build.BuildStrategy:
						return fmt.Errorf("something wrong happen")
					}
					return k8serrors.NewNotFound(schema.GroupResource{}, nn.Name)
				}

				// Override Stub get calls to include a service account
				// but none BuildStrategy
				client.GetCalls(stubGetCalls)

				_, err := reconciler.Reconcile(context.TODO(), buildRunRequest)
				// we reconcile again
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("something wrong happen"))
				Expect(client.GetCallCount()).To(Equal(5))
				Expect(client.StatusCallCount()).To(Equal(1))
			})

			It("fails on a TaskRun creation due to cluster buildstrategy not found", func() {
				// override the Build to use a cluster BuildStrategy
				buildSample = ctl.DefaultBuild(buildName, strategyName, build.ClusterBuildStrategyKind)

				// Override Stub get calls to include a service account
				// but none ClusterBuildStrategy
				client.GetCalls(ctl.StubBuildRunGetWithSA(
					buildSample,
					buildRunSample,
					ctl.DefaultServiceAccount(saName)),
				)

				// Stub that asserts the BuildRun status fields when
				// Status updates for a BuildRun take place
				statusCall := ctl.StubBuildRunStatus(
					fmt.Sprintf(" %q not found", strategyName),
					emptyTaskRunName,
					build.Condition{
						Type:   build.Succeeded,
						Reason: "ClusterBuildStrategyNotFound",
						Status: corev1.ConditionFalse,
					},
					corev1.ConditionFalse,
					buildSample.Spec,
					true,
				)
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(context.TODO(), buildRunRequest)
				// we mark the BuildRun as Failed and do not reconcile again
				Expect(err).ToNot(HaveOccurred())
			})

			It("fails on a TaskRun creation due to issues when retrieving the clusterbuildstrategy", func() {
				// override the Build to use a namespaced BuildStrategy
				buildSample = ctl.DefaultBuild(buildName, strategyName, build.ClusterBuildStrategyKind)

				// stub get calls so that on cluster strategy retrieval, we throw a random error
				stubGetCalls := func(_ context.Context, nn types.NamespacedName, object crc.Object, _ ...crc.GetOption) error {
					switch object := object.(type) {
					case *build.Build:
						buildSample.DeepCopyInto(object)
						return nil
					case *build.BuildRun:
						buildRunSample.DeepCopyInto(object)
						return nil
					case *corev1.ServiceAccount:
						ctl.DefaultServiceAccount(saName).DeepCopyInto(object)
						return nil
					case *build.ClusterBuildStrategy:
						return fmt.Errorf("something wrong happen")
					}
					return k8serrors.NewNotFound(schema.GroupResource{}, nn.Name)
				}

				client.GetCalls(stubGetCalls)

				_, err := reconciler.Reconcile(context.TODO(), buildRunRequest)
				// we reconcile again
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("something wrong happen"))
				Expect(client.GetCallCount()).To(Equal(5))
				Expect(client.StatusCallCount()).To(Equal(1))
			})

			It("fails on a TaskRun creation due to unknown buildStrategy kind", func() {
				buildSample = ctl.DefaultBuild(buildName, strategyName, "foobar")

				// Override Stub get calls to include a service account
				// but none BuildStrategy
				client.GetCalls(ctl.StubBuildRunGetWithSA(
					buildSample,
					buildRunSample,
					ctl.DefaultServiceAccount(saName)),
				)

				// Stub that asserts the BuildRun status fields when
				// Status updates for a BuildRun take place
				statusCall := ctl.StubBuildRunStatus(
					"unknown strategy foobar",
					emptyTaskRunName,
					build.Condition{
						Type:   build.Succeeded,
						Reason: "UnknownStrategyKind",
						Status: corev1.ConditionFalse,
					},
					corev1.ConditionFalse,
					buildSample.Spec,
					true,
				)
				statusWriter.UpdateCalls(statusCall)

				// we mark the BuildRun as Failed and do not reconcile again
				_, err := reconciler.Reconcile(context.TODO(), buildRunRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(client.GetCallCount()).To(Equal(4))
				Expect(client.StatusCallCount()).To(Equal(2))
			})

			It("defaults to a namespaced strategy if strategy kind is not set", func() {
				// use a Build object that does not defines the strategy Kind field
				buildSample = ctl.BuildWithoutStrategyKind(buildName, strategyName)

				// Override Stub get calls to include
				// a Build, a BuildRun, a SA and the default namespaced strategy
				client.GetCalls(ctl.StubBuildRunGetWithSAandStrategies(
					buildSample,
					buildRunSample,
					ctl.DefaultServiceAccount(saName),
					nil,
					ctl.DefaultNamespacedBuildStrategy()), // See how we include a namespaced strategy
				)

				// We do not expect an error because all resources are in place
				_, err := reconciler.Reconcile(context.TODO(), buildRunRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(client.GetCallCount()).To(Equal(5))
				Expect(client.StatusCallCount()).To(Equal(2))
			})

			It("should fail when strategy kind is not specied, because the namespaced strategy is not found", func() {
				// use a Build object that does not defines the strategy Kind field
				buildSample = ctl.BuildWithoutStrategyKind(buildName, strategyName)

				// Override Stub get calls to include
				// a Build, a BuildRun and a SA
				client.GetCalls(ctl.StubBuildRunGetWithSAandStrategies(
					buildSample,
					buildRunSample,
					ctl.DefaultServiceAccount(saName),
					nil,
					nil), // See how we do NOT include a namespaced strategy
				)

				// Stub that asserts the BuildRun status fields when
				// Status updates for a BuildRun take place
				statusWriter.UpdateCalls(ctl.StubBuildRunStatus(
					" \"foobar-strategy\" not found",
					emptyTaskRunName,
					build.Condition{
						Type:   build.Succeeded,
						Reason: "BuildStrategyNotFound",
						Status: corev1.ConditionFalse,
					},
					corev1.ConditionFalse,
					buildSample.Spec,
					true,
				))

				// We do not expect an error because we fail the BuildRun,
				// update its Status.Condition and stop reconciling
				_, err := reconciler.Reconcile(context.TODO(), buildRunRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(client.GetCallCount()).To(Equal(5))
				Expect(client.StatusCallCount()).To(Equal(2))
			})

			It("fails on a TaskRun creation due to owner references errors", func() {
				// override the Build to use a namespaced BuildStrategy
				buildSample = ctl.DefaultBuild(buildName, strategyName, build.NamespacedBuildStrategyKind)

				// Override Stub get calls to include a service account
				// and BuildStrategies
				client.GetCalls(ctl.StubBuildRunGetWithSAandStrategies(
					buildSample,
					buildRunSample,
					ctl.DefaultServiceAccount(saName),
					ctl.DefaultClusterBuildStrategy(),
					ctl.DefaultNamespacedBuildStrategy()),
				)

				// Stub that asserts the BuildRun status fields when
				// Status updates for a BuildRun take place
				statusCall := ctl.StubBuildRunStatus(
					"foobar error",
					emptyTaskRunName,
					build.Condition{
						Type:   build.Succeeded,
						Reason: "SetOwnerReferenceFailed",
						Status: corev1.ConditionFalse,
					},
					corev1.ConditionFalse,
					buildSample.Spec,
					true,
				)
				statusWriter.UpdateCalls(statusCall)

				reconciler = buildrunctl.NewReconciler(config.NewDefaultConfig(), manager,
					func(owner, object metav1.Object, scheme *runtime.Scheme, _ ...controllerutil.OwnerReferenceOption) error {
						return fmt.Errorf("foobar error")
					})

				// we mark the BuildRun as Failed and do not reconcile again
				_, err := reconciler.Reconcile(context.TODO(), buildRunRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(client.StatusCallCount()).To(Equal(2))
			})

			It("succeeds creating a TaskRun from a namespaced buildstrategy", func() {
				// override the Build to use a namespaced BuildStrategy
				buildSample = ctl.DefaultBuild(buildName, strategyName, build.NamespacedBuildStrategyKind)

				// Override Stub get calls to include a service account
				// and BuildStrategies
				client.GetCalls(ctl.StubBuildRunGetWithSAandStrategies(
					buildSample,
					buildRunSample,
					ctl.DefaultServiceAccount(saName),
					ctl.DefaultClusterBuildStrategy(),
					ctl.DefaultNamespacedBuildStrategy()),
				)

				// Stub the create calls for a TaskRun
				client.CreateCalls(func(_ context.Context, object crc.Object, _ ...crc.CreateOption) error {
					switch object := object.(type) {
					case *pipelineapi.TaskRun:
						ctl.DefaultTaskRunWithStatus(taskRunName, buildRunName, ns, corev1.ConditionTrue, "Succeeded").DeepCopyInto(object)
					}
					return nil
				})

				_, err := reconciler.Reconcile(context.TODO(), buildRunRequest)
				Expect(err).ToNot(HaveOccurred())

				Expect(client.CreateCallCount()).To(Equal(1))
			})

			It("succeeds creating a TaskRun from a cluster buildstrategy", func() {
				// override the Build to use a cluster BuildStrategy
				buildSample = ctl.DefaultBuild(buildName, strategyName, build.ClusterBuildStrategyKind)

				// Override Stub get calls to include a service account
				// and BuildStrategies
				client.GetCalls(ctl.StubBuildRunGetWithSAandStrategies(
					buildSample,
					buildRunSample,
					ctl.DefaultServiceAccount(saName),
					ctl.DefaultClusterBuildStrategy(),
					ctl.DefaultNamespacedBuildStrategy()),
				)

				// Stub the create calls for a TaskRun
				client.CreateCalls(func(_ context.Context, object crc.Object, _ ...crc.CreateOption) error {
					switch object := object.(type) {
					case *pipelineapi.TaskRun:
						ctl.DefaultTaskRunWithStatus(taskRunName, buildRunName, ns, corev1.ConditionTrue, "Succeeded").DeepCopyInto(object)
					}
					return nil
				})

				_, err := reconciler.Reconcile(context.TODO(), buildRunRequest)
				Expect(err).ToNot(HaveOccurred())
			})

			It("stops creation when a FALSE registered status of the build occurs", func() {
				// Init the Build with registered status false
				buildSample = ctl.DefaultBuildWithFalseRegistered(buildName, strategyName, build.ClusterBuildStrategyKind)
				getClientStub := func(_ context.Context, nn types.NamespacedName, object crc.Object, _ ...crc.GetOption) error {
					switch object := object.(type) {
					case *build.Build:
						buildSample.DeepCopyInto(object)
						return nil
					case *build.BuildRun:
						buildRunSample.DeepCopyInto(object)
						return nil
					}
					return k8serrors.NewNotFound(schema.GroupResource{}, nn.Name)
				}

				client.GetCalls(getClientStub)

				// Stub that asserts the BuildRun status fields when
				// Status updates for a BuildRun take place
				statusCall := ctl.StubBuildRunStatus(
					fmt.Sprintf("the Build is not registered correctly, build: %s, registered status: False, reason: something bad happened", buildName),
					emptyTaskRunName,
					build.Condition{
						Type:   build.Succeeded,
						Reason: resources.ConditionBuildRegistrationFailed,
						Status: corev1.ConditionFalse,
					},
					corev1.ConditionFalse,
					buildSample.Spec,
					true,
				)
				statusWriter.UpdateCalls(statusCall)

				// we mark the BuildRun as Failed and do not reconcile again
				_, err := reconciler.Reconcile(context.TODO(), buildRunRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(client.StatusCallCount()).To(Equal(1))
			})

			It("delays creation if the registered status of the build is not yet set", func() {
				buildSample = ctl.DefaultBuild(buildName, strategyName, build.ClusterBuildStrategyKind)
				buildSample.Status.Registered = ptr.To[corev1.ConditionStatus]("")
				buildSample.Status.Reason = ptr.To[build.BuildReason]("")

				client.GetCalls(ctl.StubBuildRunGetWithoutSA(buildSample, buildRunSample))

				_, err := reconciler.Reconcile(context.TODO(), buildRunRequest)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(fmt.Sprintf("the Build is not yet validated, build: %s", buildName)))
				Expect(client.StatusCallCount()).To(Equal(0))
			})

			It("succeeds creating a TaskRun even if the BuildSpec is already referenced", func() {
				// Set the build spec
				buildRunSample = ctl.DefaultBuildRun(buildRunName, buildName)
				buildRunSample.Status.BuildSpec = &buildSample.Spec

				// override the Build to use a namespaced BuildStrategy
				buildSample = ctl.DefaultBuild(buildName, strategyName, build.NamespacedBuildStrategyKind)

				// Override Stub get calls to include a service account
				// and BuildStrategies
				client.GetCalls(ctl.StubBuildRunGetWithSAandStrategies(
					buildSample,
					buildRunSample,
					ctl.DefaultServiceAccount(saName),
					ctl.DefaultClusterBuildStrategy(),
					ctl.DefaultNamespacedBuildStrategy()),
				)

				// Stub the create calls for a TaskRun
				client.CreateCalls(func(_ context.Context, object crc.Object, _ ...crc.CreateOption) error {
					switch object := object.(type) {
					case *pipelineapi.TaskRun:
						ctl.DefaultTaskRunWithStatus(taskRunName, buildRunName, ns, corev1.ConditionTrue, "Succeeded").DeepCopyInto(object)
					}
					return nil
				})

				_, err := reconciler.Reconcile(context.TODO(), buildRunRequest)
				Expect(err).ToNot(HaveOccurred())

				Expect(client.CreateCallCount()).To(Equal(1))
			})

			It("updates Build with error when BuildRun is already owned", func() {
				fakeOwnerName := "fakeOwner"

				// Set the build spec
				buildRunSample = ctl.BuildRunWithExistingOwnerReferences(buildRunName, buildName, fakeOwnerName)
				buildRunSample.Status.BuildSpec = &buildSample.Spec

				// override the Build to use a namespaced BuildStrategy
				buildSample = ctl.BuildWithBuildRunDeletions(buildName, strategyName, build.NamespacedBuildStrategyKind)

				// Override Stub get calls to include a service account
				// and BuildStrategies
				client.GetCalls(ctl.StubBuildRunGetWithSAandStrategies(
					buildSample,
					buildRunSample,
					ctl.DefaultServiceAccount(saName),
					ctl.DefaultClusterBuildStrategy(),
					ctl.DefaultNamespacedBuildStrategy()),
				)

				statusCall := ctl.StubBuildStatusReason(build.SetOwnerReferenceFailed,
					fmt.Sprintf("unexpected error when trying to set the ownerreference: Object /%s is already owned by another %s controller ", buildRunName, fakeOwnerName),
				)
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(context.TODO(), buildRunRequest)
				Expect(err).ToNot(HaveOccurred())

				Expect(client.CreateCallCount()).To(Equal(1))
			})

			It("updates Build with error when BuildRun and Build are not in the same ns when setting ownerreferences", func() {
				// Set the build spec
				buildRunSample = ctl.BuildRunWithFakeNamespace(buildRunName, buildName)
				buildRunSample.Status.BuildSpec = &buildSample.Spec

				// override the Build to use a namespaced BuildStrategy
				buildSample = ctl.BuildWithBuildRunDeletionsAndFakeNS(buildName, strategyName, build.NamespacedBuildStrategyKind)

				// Override Stub get calls to include a service account
				// and BuildStrategies
				client.GetCalls(ctl.StubBuildRunGetWithSAandStrategies(
					buildSample,
					buildRunSample,
					ctl.DefaultServiceAccount(saName),
					ctl.DefaultClusterBuildStrategy(),
					ctl.DefaultNamespacedBuildStrategy()),
				)

				statusCall := ctl.StubBuildStatusReason(build.SetOwnerReferenceFailed,
					fmt.Sprintf("unexpected error when trying to set the ownerreference: cross-namespace owner references are disallowed, owner's namespace %s, obj's namespace %s", buildSample.Namespace, buildRunSample.Namespace),
				)
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(context.TODO(), buildRunRequest)
				Expect(err).ToNot(HaveOccurred())

				Expect(client.CreateCallCount()).To(Equal(1))
			})

			It("ensure the Build can own a BuildRun when using the proper annotation", func() {

				buildRunSample = ctl.BuildRunWithoutSA(buildRunName, buildName)
				buildSample = ctl.BuildWithBuildRunDeletions(buildName, strategyName, build.NamespacedBuildStrategyKind)

				// Override Stub get calls to include a service account
				// and BuildStrategies
				client.GetCalls(ctl.StubBuildRunGetWithSAandStrategies(
					buildSample,
					buildRunSample,
					ctl.DefaultServiceAccount(saName),
					ctl.DefaultClusterBuildStrategy(),
					ctl.DefaultNamespacedBuildStrategy()),
				)

				// Ensure the BuildRun gets an ownershipReference when
				// the spec.Retention.AtBuildDeletion is set to true
				// in the build
				clientUpdateCalls := ctl.StubBuildUpdateOwnerReferences("Build",
					buildName,
					ptr.To(true),
					ptr.To(true),
				)
				client.UpdateCalls(clientUpdateCalls)

				_, err := reconciler.Reconcile(context.TODO(), buildRunRequest)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should return an error and stop reconciling if buildstrategy is not found", func() {
				buildRunSample = ctl.BuildRunWithoutSA(buildRunName, buildName)
				buildSample = ctl.BuildWithBuildRunDeletions(buildName, strategyName, build.ClusterBuildStrategyKind)

				// Override Stub get calls to include a service account
				// but none BuildStrategy
				client.GetCalls(ctl.StubBuildRunGetWithSA(
					buildSample,
					buildRunSample,
					ctl.DefaultServiceAccount(saName)),
				)

				statusWriter.UpdateCalls(func(_ context.Context, object crc.Object, _ ...crc.SubResourceUpdateOption) error {
					switch buildRun := object.(type) {
					case *build.BuildRun:
						if buildRun.Status.IsFailed(build.Succeeded) {
							return fmt.Errorf("failed miserably")
						}
					}
					return nil
				})

				_, err := reconciler.Reconcile(context.TODO(), buildRunRequest)
				// we expect an error because a Client.Status Update failed and we expect another reconciliation
				// to take place
				Expect(err).ToNot(BeNil())
				Expect(resources.IsClientStatusUpdateError(err)).To(BeTrue())
			})

			It("should mark buildrun succeeded false when BuildRun name is too long", func() {
				buildRunSample = ctl.BuildRunWithoutSA("f"+strings.Repeat("o", 63)+"bar", buildName)

				client.GetCalls(ctl.StubBuildRun(buildRunSample))
				statusWriter.UpdateCalls(func(_ context.Context, o crc.Object, _ ...crc.SubResourceUpdateOption) error {
					Expect(o).To(BeAssignableToTypeOf(&build.BuildRun{}))
					switch buildRun := o.(type) {
					case *build.BuildRun:
						condition := buildRun.Status.GetCondition(build.Succeeded)
						Expect(condition.Reason).To(Equal(resources.BuildRunNameInvalid))
						Expect(condition.Message).To(Equal("must be no more than 63 characters"))
					}

					return nil
				})

				_, err := reconciler.Reconcile(context.TODO(), buildRunRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
			})

			It("should mark buildrun succeeded false when BuildRun name contains illegal runes", func() {
				buildRunSample = ctl.BuildRunWithoutSA("fööbar", buildName)

				client.GetCalls(ctl.StubBuildRun(buildRunSample))
				statusWriter.UpdateCalls(func(_ context.Context, o crc.Object, _ ...crc.SubResourceUpdateOption) error {
					Expect(o).To(BeAssignableToTypeOf(&build.BuildRun{}))
					switch buildRun := o.(type) {
					case *build.BuildRun:
						condition := buildRun.Status.GetCondition(build.Succeeded)
						Expect(condition.Reason).To(Equal(resources.BuildRunNameInvalid))
						Expect(condition.Message).To(ContainSubstring("a valid label must be an empty string or consist of alphanumeric characters"))
					}

					return nil
				})

				_, err := reconciler.Reconcile(context.TODO(), buildRunRequest)
				Expect(err).To(BeNil())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
			})

			It("should fail the reconcile if an update call failed during a validation error", func() {
				buildRunSample = ctl.BuildRunWithoutSA("fööbar", buildName)

				client.GetCalls(ctl.StubBuildRun(buildRunSample))
				statusWriter.UpdateCalls(func(_ context.Context, _ crc.Object, _ ...crc.SubResourceUpdateOption) error {
					return k8serrors.NewInternalError(fmt.Errorf("something bad happened"))
				})

				_, err := reconciler.Reconcile(context.TODO(), buildRunRequest)
				Expect(err).To(HaveOccurred())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
			})
		})

		Context("from an existing BuildRun resource with embedded BuildSpec", func() {
			var clusterBuildStrategy = build.ClusterBuildStrategyKind

			BeforeEach(func() {
				buildRunRequest = newReconcileRequest(buildRunName, ns)
			})

			Context("invalid BuildRun resource", func() {
				simpleReconcileRunWithCustomUpdateCall := func(f func(*build.Condition)) {
					client.GetCalls(ctl.StubBuildRun(buildRunSample))
					statusWriter.UpdateCalls(func(_ context.Context, o crc.Object, _ ...crc.SubResourceUpdateOption) error {
						Expect(o).To(BeAssignableToTypeOf(&build.BuildRun{}))
						switch buildRun := o.(type) {
						case *build.BuildRun:
							f(buildRun.Status.GetCondition(build.Succeeded))
						}

						return nil
					})

					var taskRunCreates int
					client.CreateCalls(func(_ context.Context, o crc.Object, _ ...crc.CreateOption) error {
						switch o.(type) {
						case *pipelineapi.TaskRun:
							taskRunCreates++
						}

						return nil
					})

					// Reconcile should run through without an error
					_, err := reconciler.Reconcile(context.TODO(), buildRunRequest)
					Expect(err).To(BeNil())

					// But, make sure no TaskRun is created based upon an invalid BuildRun
					Expect(taskRunCreates).To(Equal(0))
				}

				It("should mark BuildRun as invalid if both BuildRef and BuildSpec are unspecified", func() {
					buildRunSample = &build.BuildRun{
						ObjectMeta: metav1.ObjectMeta{
							Name: buildRunName,
						},
						Spec: build.BuildRunSpec{},
					}

					simpleReconcileRunWithCustomUpdateCall(func(condition *build.Condition) {
						Expect(condition.Reason).To(Equal(resources.BuildRunNoRefOrSpec))
						Expect(condition.Message).To(Equal("no build referenced or specified, either 'buildRef' or 'buildSpec' has to be set"))
					})
				})

				It("should mark BuildRun as invalid if Build name and BuildSpec are used", func() {
					buildRunSample = &build.BuildRun{
						ObjectMeta: metav1.ObjectMeta{
							Name: buildRunName,
						},
						Spec: build.BuildRunSpec{
							Build: build.ReferencedBuild{
								Spec: &build.BuildSpec{},
								Name: ptr.To("foobar"),
							},
						},
					}

					simpleReconcileRunWithCustomUpdateCall(func(condition *build.Condition) {
						Expect(condition.Reason).To(Equal(resources.BuildRunAmbiguousBuild))
						Expect(condition.Message).To(Equal("fields 'buildRef' and 'buildSpec' are mutually exclusive"))
					})
				})

				It("should mark BuildRun as invalid if Output and BuildSpec are used", func() {
					buildRunSample = &build.BuildRun{
						ObjectMeta: metav1.ObjectMeta{
							Name: buildRunName,
						},
						Spec: build.BuildRunSpec{
							Output: &build.Image{Image: "foo:bar"},
							Build: build.ReferencedBuild{
								Spec: &build.BuildSpec{},
							},
						},
					}

					simpleReconcileRunWithCustomUpdateCall(func(condition *build.Condition) {
						Expect(condition.Reason).To(Equal(resources.BuildRunBuildFieldOverrideForbidden))
						Expect(condition.Message).To(Equal("cannot use 'output' override and 'buildSpec' simultaneously"))
					})
				})

				It("should mark BuildRun as invalid if ParamValues and BuildSpec are used", func() {
					buildRunSample = &build.BuildRun{
						ObjectMeta: metav1.ObjectMeta{
							Name: buildRunName,
						},
						Spec: build.BuildRunSpec{
							ParamValues: []build.ParamValue{{
								Name:        "foo",
								SingleValue: &build.SingleValue{Value: ptr.To("bar")},
							}},
							Build: build.ReferencedBuild{
								Spec: &build.BuildSpec{},
							},
						},
					}

					simpleReconcileRunWithCustomUpdateCall(func(condition *build.Condition) {
						Expect(condition.Reason).To(Equal(resources.BuildRunBuildFieldOverrideForbidden))
						Expect(condition.Message).To(Equal("cannot use 'paramValues' override and 'buildSpec' simultaneously"))
					})
				})

				It("should mark BuildRun as invalid if Env and BuildSpec are used", func() {
					buildRunSample = &build.BuildRun{
						ObjectMeta: metav1.ObjectMeta{
							Name: buildRunName,
						},
						Spec: build.BuildRunSpec{
							Env: []corev1.EnvVar{{Name: "foo", Value: "bar"}},
							Build: build.ReferencedBuild{
								Spec: &build.BuildSpec{},
							},
						},
					}

					simpleReconcileRunWithCustomUpdateCall(func(condition *build.Condition) {
						Expect(condition.Reason).To(Equal(resources.BuildRunBuildFieldOverrideForbidden))
						Expect(condition.Message).To(Equal("cannot use 'env' override and 'buildSpec' simultaneously"))
					})
				})

				It("should mark BuildRun as invalid if Timeout and BuildSpec are used", func() {
					buildRunSample = &build.BuildRun{
						ObjectMeta: metav1.ObjectMeta{
							Name: buildRunName,
						},
						Spec: build.BuildRunSpec{
							Timeout: &metav1.Duration{Duration: time.Second},
							Build: build.ReferencedBuild{
								Spec: &build.BuildSpec{},
							},
						},
					}

					simpleReconcileRunWithCustomUpdateCall(func(condition *build.Condition) {
						Expect(condition.Reason).To(Equal(resources.BuildRunBuildFieldOverrideForbidden))
						Expect(condition.Message).To(Equal("cannot use 'timeout' override and 'buildSpec' simultaneously"))
					})
				})
			})

			Context("valid BuildRun resource", func() {
				It("should reconcile a BuildRun with an embedded BuildSpec", func() {
					buildRunSample = &build.BuildRun{
						ObjectMeta: metav1.ObjectMeta{
							Name: buildRunName,
						},
						Spec: build.BuildRunSpec{
							Build: build.ReferencedBuild{
								Spec: &build.BuildSpec{
									Source: &build.Source{
										Type: build.GitType,
										Git: &build.Git{
											URL: "https://github.com/shipwright-io/sample-go.git",
										},
										ContextDir: ptr.To("source-build"),
									},
									Strategy: build.Strategy{
										Kind: &clusterBuildStrategy,
										Name: strategyName,
									},
									Output: build.Image{
										Image: "foo/bar:latest",
									},
								},
							},
							ServiceAccount: ptr.To(".generate"),
						},
					}

					client.GetCalls(func(_ context.Context, nn types.NamespacedName, o crc.Object, _ ...crc.GetOption) error {
						switch object := o.(type) {
						case *build.BuildRun:
							buildRunSample.DeepCopyInto(object)
							return nil
						case *build.ClusterBuildStrategy:
							ctl.ClusterBuildStrategy(strategyName).DeepCopyInto(object)
							return nil
						}

						return k8serrors.NewNotFound(schema.GroupResource{}, nn.Name)
					})

					statusWriter.UpdateCalls(func(_ context.Context, o crc.Object, _ ...crc.SubResourceUpdateOption) error {
						switch buildRun := o.(type) {
						case *build.BuildRun:
							Expect(buildRun.Labels).ToNot(HaveKey(build.LabelBuild), "no build name label is suppose to be set")
							Expect(buildRun.Labels).ToNot(HaveKey(build.LabelBuildGeneration), "no build generation label is suppose to be set")
							return nil
						}

						return nil
					})

					var taskRunCreates int
					client.CreateCalls(func(_ context.Context, o crc.Object, _ ...crc.CreateOption) error {
						switch taskRun := o.(type) {
						case *pipelineapi.TaskRun:
							taskRunCreates++
							Expect(taskRun.Labels).ToNot(HaveKey(build.LabelBuild), "no build name label is suppose to be set")
							Expect(taskRun.Labels).ToNot(HaveKey(build.LabelBuildGeneration), "no build generation label is suppose to be set")
						}

						return nil
					})

					_, err := reconciler.Reconcile(context.TODO(), buildRunRequest)
					Expect(err).ToNot(HaveOccurred())
					Expect(statusWriter.UpdateCallCount()).To(Equal(2))
					Expect(taskRunCreates).To(Equal(1))
				})

				It("should validate the embedded BuildSpec to identify that the strategy kind is unknown", func() {
					client.GetCalls(func(_ context.Context, nn types.NamespacedName, o crc.Object, _ ...crc.GetOption) error {
						switch object := o.(type) {
						case *build.BuildRun:
							(&build.BuildRun{
								ObjectMeta: metav1.ObjectMeta{Name: buildRunName},
								Spec: build.BuildRunSpec{
									Build: build.ReferencedBuild{
										Spec: &build.BuildSpec{
											Source: &build.Source{
												Type: build.GitType,
												Git:  &build.Git{URL: "https://github.com/shipwright-io/sample-go.git"},
											},
											Strategy: build.Strategy{
												Kind: (*build.BuildStrategyKind)(ptr.To("foo")), // problematic value
												Name: strategyName,
											},
											Output: build.Image{Image: "foo/bar:latest"},
										},
									},
								},
							}).DeepCopyInto(object)
							return nil
						}

						return k8serrors.NewNotFound(schema.GroupResource{}, nn.Name)
					})

					statusWriter.UpdateCalls(func(ctx context.Context, o crc.Object, sruo ...crc.SubResourceUpdateOption) error {
						Expect(o).To(BeAssignableToTypeOf(&build.BuildRun{}))
						condition := o.(*build.BuildRun).Status.GetCondition(build.Succeeded)
						Expect(condition.Status).To(Equal(corev1.ConditionFalse))
						Expect(condition.Reason).To(Equal(resources.ConditionBuildRegistrationFailed))
						Expect(condition.Message).To(ContainSubstring("unknown strategy kind"))
						return nil
					})

					_, err := reconciler.Reconcile(context.TODO(), buildRunRequest)
					Expect(err).ToNot(HaveOccurred())
					Expect(statusWriter.UpdateCallCount()).ToNot(BeZero())
				})
			})
		})

		Context("when environment variables are specified", func() {
			It("fails when the name is blank", func() {
				buildRunSample.Spec.Env = []corev1.EnvVar{
					{
						Name:  "",
						Value: "some-value",
					},
				}

				statusCall := ctl.StubFunc(corev1.ConditionFalse, build.SpecEnvNameCanNotBeBlank, "name for environment variable must not be blank")
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(context.TODO(), buildRunRequest)
				Expect(err).To(BeNil())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
			})

			It("fails when the name is blank using valueFrom", func() {
				buildRunSample.Spec.Env = []corev1.EnvVar{
					{
						Name: "",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								FieldPath: "my-field-path",
							},
						},
					},
				}

				statusCall := ctl.StubFunc(corev1.ConditionFalse, build.SpecEnvNameCanNotBeBlank, "name for environment variable must not be blank")
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(context.TODO(), buildRunRequest)
				Expect(err).To(BeNil())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
			})

			It("fails when both value and valueFrom are specified", func() {
				buildRunSample.Spec.Env = []corev1.EnvVar{
					{
						Name:  "some-name",
						Value: "some-value",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								FieldPath: "my-field-path",
							},
						},
					},
				}

				statusCall := ctl.StubFunc(corev1.ConditionFalse, build.SpecEnvOnlyOneOfValueOrValueFromMustBeSpecified, "only one of value or valueFrom must be specified")
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(context.TODO(), buildRunRequest)
				Expect(err).To(BeNil())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
			})

			It("succeeds with compliant env var using Value", func() {
				buildRunSample.Spec.Env = []corev1.EnvVar{
					{
						Name:  "some-name",
						Value: "some-value",
					},
				}

				statusCall := ctl.StubFunc(corev1.ConditionTrue, build.BuildReason(build.Succeeded), "all validations succeeded")
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(context.TODO(), buildRunRequest)
				Expect(err).To(BeNil())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
			})

			It("succeeds with compliant env var using ValueFrom", func() {
				buildRunSample.Spec.Env = []corev1.EnvVar{
					{
						Name: "some-name",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{
								FieldPath: "my-field-path",
							},
						},
					},
				}

				statusCall := ctl.StubFunc(corev1.ConditionTrue, build.BuildReason(build.Succeeded), "all validations succeeded")
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(context.TODO(), buildRunRequest)
				Expect(err).To(BeNil())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
			})
		})

		Context("when nodeSelector is specified", func() {
			It("fails when the nodeSelector is invalid", func() {
				// set nodeSelector to be invalid
				buildRunSample.Spec.NodeSelector = map[string]string{strings.Repeat("s", 64): "amd64"}

				statusCall := ctl.StubFunc(corev1.ConditionFalse, build.NodeSelectorNotValid, "name part "+validation.MaxLenError(63))
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(context.TODO(), buildRunRequest)
				Expect(err).To(BeNil())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
			})
		})

		Context("when Tolerations is specified", func() {
			It("should fail to validate when the Toleration is invalid", func() {
				// set Toleration to be invalid
				buildRunSample.Spec.Tolerations = []corev1.Toleration{{Key: strings.Repeat("s", 64), Operator: "Equal", Value: "test-value"}}

				statusCall := ctl.StubFunc(corev1.ConditionFalse, build.TolerationNotValid, validation.MaxLenError(63))
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(context.TODO(), buildRunRequest)
				Expect(err).To(BeNil())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))
			})
		})
	})
})

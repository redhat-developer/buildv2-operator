// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package buildrun_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/pointer"
	knativeapi "knative.dev/pkg/apis"
	knativev1beta1 "knative.dev/pkg/apis/duck/v1beta1"
	crc "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/shipwright-io/build/pkg/apis"
	build "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/controller/fakes"
	"github.com/shipwright-io/build/pkg/ctxlog"
	buildrunctl "github.com/shipwright-io/build/pkg/reconciler/buildrun"
	"github.com/shipwright-io/build/pkg/reconciler/buildrun/resources"
	"github.com/shipwright-io/build/test"
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
		taskRunSample                                          *v1beta1.TaskRun
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
	getClientStub := func(context context.Context, nn types.NamespacedName, object runtime.Object) error {
		switch object := object.(type) {
		case *build.Build:
			buildSample.DeepCopyInto(object)
			return nil
		case *build.BuildRun:
			buildRunSample.DeepCopyInto(object)
			return nil
		case *v1beta1.TaskRun:
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

	})

	// JustBeforeEach will always execute just before the It() specs,
	// this ensures that overrides on the BuildRun resource can happen under each
	// Context() BeforeEach() block
	JustBeforeEach(func() {
		// l := ctxlog.NewLogger("buildrun-controller-test")
		// testCtx := ctxlog.NewParentContext(l)
		testCtx := ctxlog.NewContext(context.TODO(), "fake-logger")
		reconciler = buildrunctl.NewReconciler(testCtx, config.NewDefaultConfig(), manager, controllerutil.SetControllerReference)
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
				result, err := reconciler.Reconcile(taskRunRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(reconcile.Result{}).To(Equal(result))
				Expect(client.GetCallCount()).To(Equal(2))
			})
			It("does not fail when the BuildRun does not exist", func() {

				// override the initial getClientStub, and generate a new stub
				// that only contains a Build and TaskRun, none BuildRun
				stubGetCalls := ctl.StubBuildAndTaskRun(buildSample, taskRunSample)
				client.GetCalls(stubGetCalls)

				result, err := reconciler.Reconcile(taskRunRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(reconcile.Result{}).To(Equal(result))
				Expect(client.GetCallCount()).To(Equal(3))
			})
			It("does not fail when the Build does not exist", func() {

				// override the initial getClientStub, and generate a new stub
				// that only contains a BuildRun and TaskRun, none Build
				stubGetCalls := ctl.StubBuildRunAndTaskRun(buildRunSample, taskRunSample)
				client.GetCalls(stubGetCalls)

				result, err := reconciler.Reconcile(taskRunRequest)
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
				result, err := reconciler.Reconcile(taskRunRequest)
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
				_, err := reconciler.Reconcile(taskRunRequest)

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
					ctl.DefaultServiceAccount(buildRunSample.Name+"-sa")),
				)

				// Call the reconciler
				_, err := reconciler.Reconcile(taskRunRequest)

				// Expect no error
				Expect(err).ToNot(HaveOccurred())

				// Expect one delete call for the service account
				Expect(client.DeleteCallCount()).To(Equal(1))
				_, obj, _ := client.DeleteArgsForCall(0)
				serviceAccount, castSuccessful := obj.(*corev1.ServiceAccount)
				Expect(castSuccessful).To(BeTrue())
				Expect(serviceAccount.Name).To(Equal(buildRunSample.Name + "-sa"))
				Expect(serviceAccount.Namespace).To(Equal(buildRunSample.Namespace))
			})

			It("should not panic in case the build spec strategy is nil", func() {
				// As long as the Strategy is a pointer, it can happen that the
				// field is nil. During processing, the BuildSpec is copied
				// from the respective Build. In case Strategy is nil, the
				// controller should not panic.
				buildRunSample.Status.BuildSpec.Strategy = nil
				Expect(func() {
					result, err := reconciler.Reconcile(taskRunRequest)
					Expect(err).ToNot(HaveOccurred())
					Expect(result).ToNot(BeNil())
				}).ShouldNot(Panic())
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
				result, err := reconciler.Reconcile(taskRunRequest)
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

				result, err := reconciler.Reconcile(taskRunRequest)
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

				result, err := reconciler.Reconcile(taskRunRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(reconcile.Result{}).To(Equal(result))
				Expect(client.GetCallCount()).To(Equal(2))
				Expect(client.StatusCallCount()).To(Equal(1))
			})

			It("should recognize the BuildRun is canceled", func() {
				// set cancel
				buildRunSampleCopy := buildRunSample.DeepCopy()
				buildRunSampleCopy.Spec.State = build.BuildRunStateCancel

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
				stubUpdateCalls := func(context context.Context, object runtime.Object, opts ...crc.UpdateOption) error {
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
				stubPatchCalls := func(context context.Context, object runtime.Object, patch crc.Patch, opts ...crc.PatchOption) error {
					switch v := object.(type) {
					case *v1beta1.TaskRun:
						if v.Name == taskRunSample.Name {
							cancelPatchCalled = true
						}
					}
					return nil
				}
				client.PatchCalls(stubPatchCalls)

				_, err := reconciler.Reconcile(buildRunRequest)
				Expect(err).To(BeNil())
				Expect(resources.IsClientStatusUpdateError(err)).To(BeFalse())
				Expect(cancelPatchCalled).To(BeTrue())

				// actually set value the patch would have set (but we overrode above)
				// for next call
				taskRunSample.Spec.Status = v1beta1.TaskRunSpecStatusCancelled
				taskRunSample.Status.Conditions = knativev1beta1.Conditions{
					{
						Type:   knativeapi.ConditionSucceeded,
						Reason: string(v1beta1.TaskRunReasonCancelled),
						Status: corev1.ConditionFalse,
					},
				}

				_, err = reconciler.Reconcile(buildRunRequest)
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

				result, err := reconciler.Reconcile(taskRunRequest)
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

				result, err := reconciler.Reconcile(taskRunRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(reconcile.Result{}).To(Equal(result))

				// Three client calls because based on the Stub, we should
				// trigger a call to get the related TaskRun pod.
				Expect(client.GetCallCount()).To(Equal(3))
			})

			It("does not break the reconcile when a failed taskrun has a pod with no failed container", func() {
				buildRunSample = ctl.BuildRunWithBuildSnapshot(buildRunName, buildName)
				taskRunSample = &v1beta1.TaskRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      taskRunName,
						Namespace: ns,
						Labels:    map[string]string{"buildrun.shipwright.io/name": buildRunName},
					},
					Spec: v1beta1.TaskRunSpec{},
					Status: v1beta1.TaskRunStatus{
						TaskRunStatusFields: v1beta1.TaskRunStatusFields{
							PodName: "foobar",
							CompletionTime: &metav1.Time{
								Time: time.Now(),
							},
							StartTime: &metav1.Time{
								Time: time.Now(),
							},
						},
						Status: knativev1beta1.Status{
							Conditions: knativev1beta1.Conditions{
								{
									Type:    knativeapi.ConditionSucceeded,
									Reason:  string(v1beta1.TaskRunReasonFailed),
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
				result, err := reconciler.Reconcile(taskRunRequest)
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
				buildRunSampleCopy.Spec.State = build.BuildRunStateCancel

				client.GetCalls(ctl.StubBuildCRDs(
					buildSample,
					buildRunSampleCopy,
					ctl.DefaultServiceAccount("foobar"),
					ctl.DefaultClusterBuildStrategy(),
					ctl.DefaultNamespacedBuildStrategy(),
				))

				cancelUpdateCalled := false
				// override the updateClientStub so we can see the update on the BuildRun condition
				stubUpdateCalls := func(context context.Context, object runtime.Object, opts ...crc.UpdateOption) error {
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

				_, err := reconciler.Reconcile(buildRunRequest)
				Expect(err).To(BeNil())
				Expect(resources.IsClientStatusUpdateError(err)).To(BeFalse())

				Expect(cancelUpdateCalled).To(BeTrue())
			})

			It("should return none error and stop reconciling if referenced Build is not found", func() {
				buildRunSample = ctl.BuildRunWithoutSA(buildRunName, buildName)

				// override the initial getClientStub, and generate a new stub
				// that only contains a Buildrun
				stubGetCalls := func(context context.Context, nn types.NamespacedName, object runtime.Object) error {
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

				_, err := reconciler.Reconcile(buildRunRequest)
				Expect(err).To(BeNil())
				Expect(resources.IsClientStatusUpdateError(err)).To(BeFalse())
			})

			It("should return an error and continue reconciling if referenced Build is not found and the status update fails", func() {
				buildRunSample = ctl.BuildRunWithoutSA(buildRunName, buildName)

				// override the initial getClientStub, and generate a new stub
				// that only contains a BuildRun
				stubGetCalls := func(context context.Context, nn types.NamespacedName, object runtime.Object) error {
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

				statusWriter.UpdateCalls(func(_ context.Context, object runtime.Object, _ ...crc.UpdateOption) error {
					switch buildRun := object.(type) {
					case *build.BuildRun:
						if buildRun.Status.IsFailed(build.Succeeded) {
							return fmt.Errorf("failed miserably")
						}
					}
					return nil
				})

				_, err := reconciler.Reconcile(buildRunRequest)
				Expect(err).ToNot(BeNil())
				Expect(resources.IsClientStatusUpdateError(err)).To(BeTrue())
			})

			It("fails on a TaskRun creation due to service account not found", func() {

				// override the initial getClientStub, and generate a new stub
				// that only contains a Build and Buildrun, none TaskRun
				stubGetCalls := func(context context.Context, nn types.NamespacedName, object runtime.Object) error {
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
				_, err := reconciler.Reconcile(buildRunRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(client.GetCallCount()).To(Equal(4))
				Expect(client.StatusCallCount()).To(Equal(2))
			})
			It("fails on a TaskRun creation due to issues when retrieving the service account", func() {

				// override the initial getClientStub, and generate a new stub
				// that only contains a Build, BuildRun and a random error when
				// retrieving a service account
				stubGetCalls := func(context context.Context, nn types.NamespacedName, object runtime.Object) error {
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
				_, err := reconciler.Reconcile(buildRunRequest)
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
					fmt.Sprintf(" \"%s\" not found", strategyName),
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

				_, err := reconciler.Reconcile(buildRunRequest)
				// we mark the BuildRun as Failed and do not reconcile again
				Expect(err).ToNot(HaveOccurred())
			})

			It("fails on a TaskRun creation due to issues when retrieving the buildstrategy", func() {

				// override the Build to use a namespaced BuildStrategy
				buildSample = ctl.DefaultBuild(buildName, strategyName, build.NamespacedBuildStrategyKind)

				// stub get calls so that on namespaced strategy retrieval, we throw a random error
				stubGetCalls := func(context context.Context, nn types.NamespacedName, object runtime.Object) error {
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

				_, err := reconciler.Reconcile(buildRunRequest)
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
					fmt.Sprintf(" \"%s\" not found", strategyName),
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

				_, err := reconciler.Reconcile(buildRunRequest)
				// we mark the BuildRun as Failed and do not reconcile again
				Expect(err).ToNot(HaveOccurred())
			})

			It("fails on a TaskRun creation due to issues when retrieving the clusterbuildstrategy", func() {

				// override the Build to use a namespaced BuildStrategy
				buildSample = ctl.DefaultBuild(buildName, strategyName, build.ClusterBuildStrategyKind)

				// stub get calls so that on cluster strategy retrieval, we throw a random error
				stubGetCalls := func(context context.Context, nn types.NamespacedName, object runtime.Object) error {
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

				_, err := reconciler.Reconcile(buildRunRequest)
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
				_, err := reconciler.Reconcile(buildRunRequest)
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
				_, err := reconciler.Reconcile(buildRunRequest)
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
				_, err := reconciler.Reconcile(buildRunRequest)
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

				testCtx := ctxlog.NewContext(context.TODO(), "fake-logger")
				reconciler = buildrunctl.NewReconciler(testCtx, config.NewDefaultConfig(), manager,
					func(owner, object metav1.Object, scheme *runtime.Scheme) error {
						return fmt.Errorf("foobar error")
					})

				// we mark the BuildRun as Failed and do not reconcile again
				_, err := reconciler.Reconcile(buildRunRequest)
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
				client.CreateCalls(func(context context.Context, object runtime.Object, _ ...crc.CreateOption) error {
					switch object := object.(type) {
					case *v1beta1.TaskRun:
						ctl.DefaultTaskRunWithStatus(taskRunName, buildRunName, ns, corev1.ConditionTrue, "Succeeded").DeepCopyInto(object)
					}
					return nil
				})

				_, err := reconciler.Reconcile(buildRunRequest)
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
				client.CreateCalls(func(context context.Context, object runtime.Object, _ ...crc.CreateOption) error {
					switch object := object.(type) {
					case *v1beta1.TaskRun:
						ctl.DefaultTaskRunWithStatus(taskRunName, buildRunName, ns, corev1.ConditionTrue, "Succeeded").DeepCopyInto(object)
					}
					return nil
				})

				_, err := reconciler.Reconcile(buildRunRequest)
				Expect(err).ToNot(HaveOccurred())
			})
			It("stops creation when a FALSE registered status of the build occurs", func() {
				// Init the Build with registered status false
				buildSample = ctl.DefaultBuildWithFalseRegistered(buildName, strategyName, build.ClusterBuildStrategyKind)
				getClientStub := func(context context.Context, nn types.NamespacedName, object runtime.Object) error {
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
						Reason: "BuildRegistrationFailed",
						Status: corev1.ConditionFalse,
					},
					corev1.ConditionFalse,
					buildSample.Spec,
					true,
				)
				statusWriter.UpdateCalls(statusCall)

				// we mark the BuildRun as Failed and do not reconcile again
				_, err := reconciler.Reconcile(buildRunRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(client.StatusCallCount()).To(Equal(1))
			})

			It("delays creation if the registered status of the build is not yet set", func() {
				buildSample = ctl.DefaultBuild(buildName, strategyName, build.ClusterBuildStrategyKind)
				buildSample.Status.Registered = ""
				buildSample.Status.Reason = ""

				client.GetCalls(ctl.StubBuildRunGetWithoutSA(buildSample, buildRunSample))

				_, err := reconciler.Reconcile(buildRunRequest)
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
				client.CreateCalls(func(context context.Context, object runtime.Object, _ ...crc.CreateOption) error {
					switch object := object.(type) {
					case *v1beta1.TaskRun:
						ctl.DefaultTaskRunWithStatus(taskRunName, buildRunName, ns, corev1.ConditionTrue, "Succeeded").DeepCopyInto(object)
					}
					return nil
				})

				_, err := reconciler.Reconcile(buildRunRequest)
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

				_, err := reconciler.Reconcile(buildRunRequest)
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

				_, err := reconciler.Reconcile(buildRunRequest)
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
				// the buildv1alpha1.AnnotationBuildRunDeletion is set to true
				// in the build
				clientUpdateCalls := ctl.StubBuildUpdateOwnerReferences("Build",
					buildName,
					pointer.BoolPtr(true),
					pointer.BoolPtr(true),
				)
				client.UpdateCalls(clientUpdateCalls)

				_, err := reconciler.Reconcile(buildRunRequest)
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

				statusWriter.UpdateCalls(func(_ context.Context, object runtime.Object, _ ...crc.UpdateOption) error {
					switch buildRun := object.(type) {
					case *build.BuildRun:
						if buildRun.Status.IsFailed(build.Succeeded) {
							return fmt.Errorf("failed miserably")
						}
					}
					return nil
				})

				_, err := reconciler.Reconcile(buildRunRequest)
				// we expect an error because a Client.Status Update failed and we expect another reconciliation
				// to take place
				Expect(err).ToNot(BeNil())
				Expect(resources.IsClientStatusUpdateError(err)).To(BeTrue())
			})
		})

		Context("when environment variables are specified", func() {
			JustBeforeEach(func() {

			})
			It("fails when the name is blank", func() {
				buildRunSample.Spec.Env = []corev1.EnvVar{
					{
						Name:  "",
						Value: "some-value",
					},
				}

				statusCall := ctl.StubFunc(corev1.ConditionFalse, build.SpecEnvNameCanNotBeBlank, "name for environment variable must not be blank")
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(buildRunRequest)
				Expect(err).To(BeNil())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))

			})
			It("fails when the value is blank", func() {
				buildRunSample.Spec.Env = []corev1.EnvVar{
					{
						Name:  "some-name",
						Value: "",
					},
				}

				statusCall := ctl.StubFunc(corev1.ConditionFalse, build.SpecEnvValueCanNotBeBlank, "value for environment variable \"some-name\" must not be blank")
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(buildRunRequest)
				Expect(err).To(BeNil())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))

			})
			It("succeeds when name and value are not blank", func() {
				buildRunSample.Spec.Env = []corev1.EnvVar{
					{
						Name:  "some-name",
						Value: "some-value",
					},
				}

				statusCall := ctl.StubFunc(corev1.ConditionTrue, build.BuildReason(build.Succeeded), "all validations succeeded")
				statusWriter.UpdateCalls(statusCall)

				_, err := reconciler.Reconcile(buildRunRequest)
				Expect(err).To(BeNil())
				Expect(statusWriter.UpdateCallCount()).To(Equal(1))

			})
		})
	})
})

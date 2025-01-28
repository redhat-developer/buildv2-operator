// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package build

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	build "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	"github.com/shipwright-io/build/pkg/config"
	"github.com/shipwright-io/build/pkg/ctxlog"
	buildmetrics "github.com/shipwright-io/build/pkg/metrics"
	"github.com/shipwright-io/build/pkg/validate"
)

// build a list of current validation types
var validationTypes = [...]string{
	validate.OwnerReferences,
	validate.SourceURL,
	validate.Secrets,
	validate.Strategies,
	validate.Source,
	validate.Output,
	validate.BuildName,
	validate.Envs,
	validate.Triggers,
	validate.NodeSelector,
	validate.Tolerations,
}

// ReconcileBuild reconciles a Build object
type ReconcileBuild struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	config                *config.Config
	client                client.Client
	scheme                *runtime.Scheme
	setOwnerReferenceFunc setOwnerReferenceFunc
}

// NewReconciler returns a new reconcile.Reconciler
func NewReconciler(c *config.Config, mgr manager.Manager, ownerRef setOwnerReferenceFunc) reconcile.Reconciler {
	return &ReconcileBuild{
		config:                c,
		client:                mgr.GetClient(),
		scheme:                mgr.GetScheme(),
		setOwnerReferenceFunc: ownerRef,
	}
}

// Reconcile reads that state of the cluster for a Build object and makes changes based on the state read
// and what is in the Build.Spec
func (r *ReconcileBuild) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	// Set the ctx to be Background, as the top-level context for incoming requests.
	ctx, cancel := context.WithTimeout(ctx, r.config.CtxTimeOut)
	defer cancel()

	ctxlog.Debug(ctx, "start reconciling Build", namespace, request.Namespace, name, request.Name)

	b := &build.Build{}
	if err := r.client.Get(ctx, request.NamespacedName, b); err != nil {
		if apierrors.IsNotFound(err) {
			ctxlog.Debug(ctx, "finish reconciling build. build was not found", namespace, request.Namespace, name, request.Name)
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, err
	}

	// Populate the status struct with default values
	b.Status.Registered = ptr.To[corev1.ConditionStatus](corev1.ConditionFalse)
	b.Status.Reason = ptr.To[build.BuildReason](build.SucceedStatus)

	// trigger all current validations
	for _, validationType := range validationTypes {
		v, err := validate.NewValidation(validationType, b, r.client, r.scheme)
		if err != nil {
			// when the validation type is unknown
			return reconcile.Result{}, err
		}

		if err := v.ValidatePath(ctx); err != nil {
			// We enqueue another reconcile here. This is done only for validation
			// types where the error can be produced from a failed API call.
			if validationType == validate.Secrets || validationType == validate.Strategies {
				return reconcile.Result{}, err
			}

			if validationType == validate.OwnerReferences {
				// we do not want to bail out here if the owerreference validation fails, we ignore this error on purpose
				// In case we just created the Build, we want the Build reconcile logic to continue, in order to
				// validate the Build references ( e.g secrets, strategies )
				ctxlog.Info(ctx, "unexpected error during ownership reference validation",
					namespace, b.Namespace,
					name, b.Name,
					"error", err)
			}
		}

		if b.Status.Reason == nil || *b.Status.Reason != build.SucceedStatus {
			if err := r.client.Status().Update(ctx, b); err != nil {
				return reconcile.Result{}, err
			}

			return reconcile.Result{}, nil
		}
	}

	b.Status.Registered = ptr.To[corev1.ConditionStatus](corev1.ConditionTrue)
	b.Status.Message = ptr.To(build.AllValidationsSucceeded)
	if err := r.client.Status().Update(ctx, b); err != nil {
		return reconcile.Result{}, err
	}

	// Increase Build count in metrics
	buildmetrics.BuildCountInc(b.Spec.Strategy.Name, b.Namespace, b.Name)

	ctxlog.Debug(ctx, "finishing reconciling Build", namespace, request.Namespace, name, request.Name)
	return reconcile.Result{}, nil
}

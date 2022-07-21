/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	resiliencyv1 "grainger.com/api/v1"
	"grainger.com/pkg/faultinjection"
	clientNetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
)

// FaultInjectionReconciler reconciles a FaultInjection object
type FaultInjectionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=resiliency.grainger.com,resources=faultinjections,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=resiliency.grainger.com,resources=faultinjections/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=resiliency.grainger.com,resources=faultinjections/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the FaultInjection object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *FaultInjectionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)
	reqLogger.Info("=== Reconciling Forward Map")

	// _ = context.Background()
	// reqLogger := r.Log.WithValues("workload", req.NamespacedName)

	// your logic here
	reqLogger.Info("====== Reconciling Workload =======")
	instance := &resiliencyv1.FaultInjection{}

	err := r.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		// object not found, could have been deleted after
		// reconcile request, hence don't requeue
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		// error reading the object, requeue the request
		return ctrl.Result{}, err
	}

	// check current status of owned resources
	if instance.Status.Phase == "" {
		instance.Status.Phase = resiliencyv1.PhasePending
	}

	// check virtual service status
	vsResult, err := r.checkEnvoyFilter(instance, req, reqLogger)
	if err != nil {
		return vsResult, err
	}

	// update status
	err = r.Status().Update(context.TODO(), instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil

}

// SetupWithManager sets up the controller with the Manager.
func (r *FaultInjectionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// reqLogger := log.FromContext(context.TODO())

	err := ctrl.NewControllerManagedBy(mgr).
		For(&resiliencyv1.FaultInjection{}).
		Owns(&clientNetworking.EnvoyFilter{}).
		Complete(r)

	if err != nil {
		// reqLogger.Error(err, "unable to create controller")
		return err
	}
	return nil
}

func (r *FaultInjectionReconciler) checkEnvoyFilter(instance *resiliencyv1.FaultInjection, req ctrl.Request, logger logr.Logger) (ctrl.Result, error) {
	switch instance.Status.Phase {
	case resiliencyv1.PhasePending:
		// logger.Info("Transitioning state to create EnvoyFilter")
		instance.Status.Phase = resiliencyv1.PhaseCreated
	case resiliencyv1.PhaseCreated:
		// logger.Info("EnvoyFilter", "PHASE:", instance.Status.Phase)
		query := &clientNetworking.EnvoyFilter{}
		// query := clientNetworking.VirtualService{}

		// check if virtual service already exists
		lookupKey := types.NamespacedName{
			// Name:      instance.GetIstioResourceName(),
			// Namespace: instance.GetNamespace(),
			Name:      "python-api-faultinjection",
			Namespace: "default",
		}

		err := r.Get(context.TODO(), lookupKey, query)
		if err != nil && errors.IsNotFound(err) {
			// logger.Info("virtual service not found but should exist", "lookup key", lookupKey, "error", err)
			// virtual service got deleted or hasn't been created yet
			// create one now
			vs := faultinjection.CreateFaultInjectionEnvoyFilter(instance)
			logger.Info("after creating fault injection envoy filter")
			err = ctrl.SetControllerReference(instance, vs, r.Scheme)
			if err != nil {
				logger.Info("Error setting controller reference", err)
				return ctrl.Result{}, err
			}

			err = r.Create(context.TODO(), vs)
			if err != nil {
				logger.Error(err, "Unable to create EnvoyFilter")
				return ctrl.Result{}, err
			}

			logger.Info("Successfully created EnvoyFilter")
		} else if err != nil {
			// logger.Error(err, "Unable to create EnvoyFilter", err)
			return ctrl.Result{}, err
		} else {
			// don't requeue, it will happen automatically when
			// virtual service status changes
			return ctrl.Result{}, nil
		}

	default:
		logger.Info("Default switch EnvoyFilter")

		// more fields related to virtual service status can be checked
		// see more at https://pkg.go.dev/istio.io/api/meta/v1alpha1#IstioStatus

	}

	return ctrl.Result{}, nil
}

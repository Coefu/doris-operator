/*
Copyright 2023.

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

package fectl

import (
	"context"
	"k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	dorisv1alpha1 "doris-operator/api/v1alpha1"
)

// FeReconciler reconciles a Fe object
type FeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=operator.doris.io,resources=ves,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.doris.io,resources=ves/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.doris.io,resources=ves/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Fe object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *FeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the Fe instance
	Fe := &dorisv1alpha1.Fe{}
	err := r.Get(ctx, req.NamespacedName, Fe)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("Fe resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Fe", Fe.Name)
		return ctrl.Result{}, err
	}

	// Check all fe dependent resources and create them if they do not exist.
	// Consists of the following resources: statefulset, service, configmap, ingress.
	notFoundResources := r.check(ctx, Fe)
	if len(notFoundResources) != 0 {
		err = r.createDepResource(ctx, Fe, notFoundResources)
		if err != nil {
			log.Error(err, "Create fe dep resources failed.", Fe.Namespace, Fe.Name)
			return ctrl.Result{}, err
		}
	}

	// If the fe dependent resource exists, compare the fe dependent statefulSet configuration to the latest configuration.
	_, err = r.compareUpdate(ctx, Fe)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *FeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dorisv1alpha1.Fe{}).
		Complete(r)
}

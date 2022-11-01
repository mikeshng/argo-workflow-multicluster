/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by workflowlicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package workflow

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	argov1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	workv1 "open-cluster-management.io/api/work/v1"
)

// WorkflowStatusReconciler reconciles a Workflow object
type WorkflowStatusReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=argoproj.io,resources=workflows,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=work.open-cluster-management.io,resources=manifestworks,verbs=get;list;watch

// ManifestWorkPredicateFunctions defines which ManifestWork this controller should watch
var ManifestWorkPredicateFunctions = predicate.Funcs{
	UpdateFunc: func(e event.UpdateEvent) bool {
		newManifestWork := e.ObjectNew.(*workv1.ManifestWork)
		return containsValidOCMStatusSyncLabel(*newManifestWork) && containsValidOCMHubWorkflowAnnotation(*newManifestWork)

	},
	CreateFunc: func(e event.CreateEvent) bool {
		manifestWork := e.Object.(*workv1.ManifestWork)
		return containsValidOCMStatusSyncLabel(*manifestWork) && containsValidOCMHubWorkflowAnnotation(*manifestWork)
	},

	DeleteFunc: func(e event.DeleteEvent) bool {
		manifestWork := e.Object.(*workv1.ManifestWork)
		return containsValidOCMStatusSyncLabel(*manifestWork) && containsValidOCMHubWorkflowAnnotation(*manifestWork)
	},
}

// SetupWithManager sets up the controller with the Manager.
func (re *WorkflowStatusReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&workv1.ManifestWork{}).
		WithEventFilter(ManifestWorkPredicateFunctions).
		Complete(re)
}

// Reconcile populates the Workflow status based on the associated ManifestWork's status feedback
func (r *WorkflowStatusReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("reconciling Workflow for status update..")
	defer log.Info("done reconciling Workflow for status update")

	var manifestWork workv1.ManifestWork
	if err := r.Get(ctx, req.NamespacedName, &manifestWork); err != nil {
		log.Error(err, "unable to fetch ManifestWork")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if manifestWork.ObjectMeta.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	resourceManifests := manifestWork.Status.ResourceStatus.Manifests

	phase := ""
	if len(resourceManifests) > 0 {
		statusFeedbacks := resourceManifests[0].StatusFeedbacks.Values
		if len(statusFeedbacks) > 0 && statusFeedbacks[0].Value.String != nil {
			phase = *statusFeedbacks[0].Value.String
		}
	}

	if len(phase) == 0 {
		log.Info("phase is not ManifestWork status feedback yet")
		return ctrl.Result{}, nil
	}

	log.Info("updating Workflow status with ManifestWork status feedback")
	workflowNamespace := manifestWork.Annotations[AnnotationKeyHubWorkflowNamespace]
	workflowName := manifestWork.Annotations[AnnotationKeyHubWorkflowName]

	workflow := argov1alpha1.Workflow{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: workflowNamespace, Name: workflowName}, &workflow); err != nil {
		log.Error(err, "unable to fetch Workflow")
		return ctrl.Result{}, err
	}

	workflow.Status.Phase = argov1alpha1.WorkflowPhase(phase)
	err := r.Client.Update(ctx, &workflow)
	if err != nil {
		log.Error(err, "unable to update Workflow")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

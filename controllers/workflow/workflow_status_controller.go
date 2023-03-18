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
	"sigs.k8s.io/controller-runtime/pkg/log"

	argov1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	workflowv1alpha1 "open-cluster-management.io/argo-workflow-multicluster/api/v1alpha1"
)

// WorkflowStatusReconciler reconciles a Workflow object
type WorkflowStatusReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=argoproj.io,resources=workflows,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=argoproj.io,resources=workflowstatusresults,verbs=get;list;watch

// SetupWithManager sets up the controller with the Manager.
func (re *WorkflowStatusReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&workflowv1alpha1.WorkflowStatusResult{}).
		Complete(re)
}

// Reconcile populates the Workflow status based on the associated WorkflowStatusResult
// The status sync flow:
// Workflow (dormant) on hub cluster is created and it will be propagated to managed cluster(s)
// => Workflow on managed cluster (contains annotations that reference the hub cluster dormant Workflow)
// => The managed cluster status sync agent will create/update a WorkflowStatusResult on the hub cluster (contains annotations that reference the hub cluster dormant Workflow)
// => using the references from WorkflowStatusResult this reconciler finds the dormant Workflow and populate the status.
func (r *WorkflowStatusReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("reconciling WorkflowStatusResult for status update..")
	defer log.Info("done reconciling WorkflowStatusResult for status update")

	var workflowStatusResult workflowv1alpha1.WorkflowStatusResult
	if err := r.Get(ctx, req.NamespacedName, &workflowStatusResult); err != nil {
		log.Error(err, "unable to fetch WorkflowStatusResult")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if workflowStatusResult.ObjectMeta.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	workflowName := workflowStatusResult.Annotations[AnnotationKeyHubWorkflowName]
	workflowNamespace := workflowStatusResult.Annotations[AnnotationKeyHubWorkflowNamespace]

	workflow := argov1alpha1.Workflow{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: workflowNamespace, Name: workflowName}, &workflow); err != nil {
		log.Error(err, "unable to fetch Workflow")
		return ctrl.Result{}, err
	}

	workflow.Status = workflowStatusResult.WorkflowStatus

	err := r.Client.Update(ctx, &workflow)
	if err != nil {
		log.Error(err, "unable to update Workflow")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

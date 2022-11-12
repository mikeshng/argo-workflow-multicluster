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
	"time"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	argov1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
)

const (
	// Workflow annotation that dictates which OCM Placement this Workflow should use to determine the managed cluster.
	AnnotationKeyOCMPlacement = "workflows.argoproj.io/ocm-placement"
)

// WorkflowPlacementReconciler reconciles a Workflow object
type WorkflowPlacementReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=argoproj.io,resources=workflows,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=cluster.open-cluster-management.io,resources=placementdecisions,verbs=get;list;watch

// WorkflowPredicateFunctions defines which Workflow this controller evaluate the placement decision
var WorkflowPlacementPredicateFunctions = predicate.Funcs{
	UpdateFunc: func(e event.UpdateEvent) bool {
		newWorkflow := e.ObjectNew.(*argov1alpha1.Workflow)
		return containsValidOCMLabel(*newWorkflow) && containsValidOCMPlacementAnnotation(*newWorkflow)

	},
	CreateFunc: func(e event.CreateEvent) bool {
		workflow := e.Object.(*argov1alpha1.Workflow)
		return containsValidOCMLabel(*workflow) && containsValidOCMPlacementAnnotation(*workflow)
	},

	DeleteFunc: func(e event.DeleteEvent) bool {
		return false
	},
}

// SetupWithManager sets up the controller with the Manager.
func (re *WorkflowPlacementReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&argov1alpha1.Workflow{}).
		WithEventFilter(WorkflowPlacementPredicateFunctions).
		Complete(re)
}

// Reconcile evaluates the PlacementDecision based on the Placement reference then populates the ManagedCluster annotation with the reuslt
func (r *WorkflowPlacementReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("reconciling Workflow for Placement evaluation...")

	var workflow argov1alpha1.Workflow
	if err := r.Get(ctx, req.NamespacedName, &workflow); err != nil {
		log.Error(err, "unable to fetch Workflow")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if workflow.ObjectMeta.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	placementRef := workflow.Annotations[AnnotationKeyOCMPlacement]

	// query all placementdecisions of the placement
	requirement, err := labels.NewRequirement(clusterv1beta1.PlacementLabel, selection.Equals, []string{placementRef})
	if err != nil {
		log.Error(err, "unable to create new PlacementDecision label requirement")
		return ctrl.Result{}, err
	}

	labelSelector := labels.NewSelector().Add(*requirement)
	placementDecisions := &clusterv1beta1.PlacementDecisionList{}
	listopts := &client.ListOptions{}
	listopts.LabelSelector = labelSelector
	listopts.Namespace = workflow.Namespace

	err = r.List(ctx, placementDecisions, listopts)
	if err != nil {
		log.Error(err, "unable to list PlacementDecisions")
		return ctrl.Result{}, err
	}

	if len(placementDecisions.Items) == 0 {
		log.Info("unable to find any PlacementDecision, try again after 10 seconds")
		return ctrl.Result{RequeueAfter: time.Second * 10}, nil
	}

	// TODO only handle one PlacementDecision target for now
	pd := placementDecisions.Items[0]
	if len(pd.Status.Decisions) == 0 {
		log.Info("unable to find any Decisions from PlacementDecision, try again after 10 seconds")
		return ctrl.Result{RequeueAfter: time.Second * 10}, nil
	}

	// TODO only using the first decision
	managedClusterName := pd.Status.Decisions[0].ClusterName
	if len(managedClusterName) == 0 {
		log.Info("unable to find a valid ManagedCluster from PlacementDecision, try again after 10 seconds")
		return ctrl.Result{RequeueAfter: time.Second * 10}, nil
	}

	log.Info("updating Workflow with annotation ManagedCluster: " + managedClusterName)

	workflow.Annotations[AnnotationKeyOCMPlacement] = ""
	workflow.Annotations[AnnotationKeyOCMManagedCluster] = managedClusterName

	err = r.Client.Update(ctx, &workflow)
	if err != nil {
		log.Error(err, "unable to update Workflow")
		return ctrl.Result{}, err
	}

	log.Info("done reconciling Workflow for Placement evaluation")

	return ctrl.Result{}, nil
}

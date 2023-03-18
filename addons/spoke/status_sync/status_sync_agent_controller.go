package status_sync

import (
	"context"
	"fmt"
	"reflect"

	argov1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	workflowv1alpha1 "open-cluster-management.io/argo-workflow-multicluster/api/v1alpha1"
	workflowcontroller "open-cluster-management.io/argo-workflow-multicluster/controllers/workflow"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type ArgoWorkflowStatusController struct {
	spokeClient client.Client
	hubClient   client.Client
	log         logr.Logger
	clusterName string
}

var WorkflowPredicateFunctions = predicate.Funcs{
	// only reconcile on status chagne
	UpdateFunc: func(e event.UpdateEvent) bool {
		newWf := e.ObjectNew.(*argov1alpha1.Workflow)
		oldWf := e.ObjectOld.(*argov1alpha1.Workflow)

		return containsValidOCMAnnotations(*newWf) && !reflect.DeepEqual(newWf.Status, oldWf.Status)
	},
	CreateFunc: func(e event.CreateEvent) bool {
		workflow := e.Object.(*argov1alpha1.Workflow)
		return containsValidOCMAnnotations(*workflow)
	},
	// do not reconcile on delete
	DeleteFunc: func(e event.DeleteEvent) bool {
		return false
	},
}

func (c *ArgoWorkflowStatusController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&argov1alpha1.Workflow{}).
		WithEventFilter(WorkflowPredicateFunctions).
		Complete(c)
}

// Reconcile Workflow status changes and create/update a WorkflowStatusResult CR in the hub cluster's managed cluster namespace.
// This agent only has permission to create/update WorkflowStatusResult in that particular namespace.
// The hub cluster workflow status controller will sync the WorkflowStatusResult from hub's managed cluster namespace to the original dormant Workflow's status.
func (c *ArgoWorkflowStatusController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	c.log.Info(fmt.Sprintf("reconciling... %s", req))
	defer c.log.Info(fmt.Sprintf("done reconcile %s", req))

	workflow := argov1alpha1.Workflow{}
	err := c.spokeClient.Get(ctx, req.NamespacedName, &workflow)
	switch {
	case errors.IsNotFound(err):
		return ctrl.Result{}, nil
	case err != nil:
		c.log.Error(err, "unable to get Workflow")
		return ctrl.Result{}, err
	}

	if !workflow.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	hubWorkflowStatusResult := workflowv1alpha1.WorkflowStatusResult{}
	hubWorkflowStatusResult.Namespace = c.clusterName
	hubWorkflowStatusResult.Name = generateHubWorkflowStatusResultName(workflow)
	hubWorkflowStatusResult.WorkflowStatus = workflow.Status
	hubWorkflowStatusResult.Annotations = map[string]string{
		workflowcontroller.AnnotationKeyHubWorkflowName:      workflow.Annotations[workflowcontroller.AnnotationKeyHubWorkflowName],
		workflowcontroller.AnnotationKeyHubWorkflowNamespace: workflow.Annotations[workflowcontroller.AnnotationKeyHubWorkflowNamespace],
	}
	err = c.hubClient.Get(ctx, types.NamespacedName{Namespace: hubWorkflowStatusResult.Namespace, Name: hubWorkflowStatusResult.Name}, &hubWorkflowStatusResult)
	switch {
	case errors.IsNotFound(err):
		if err = c.hubClient.Create(ctx, &hubWorkflowStatusResult); err != nil {
			c.log.Error(err, "unable to create hub WorkflowStatusResult")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	case err != nil:
		c.log.Error(err, "unable to get hub WorkflowStatusResult")
		return ctrl.Result{}, err
	}

	hubWorkflowStatusResult.WorkflowStatus = workflow.Status
	err = c.hubClient.Update(ctx, &hubWorkflowStatusResult)
	if err != nil {
		c.log.Error(err, "unable to update hub WorkflowStatusResult")
	}

	return ctrl.Result{}, err
}

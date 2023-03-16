package status_sync

import (
	"context"
	"fmt"
	"reflect"

	argov1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	AnnotationKeyHubWorkflowUID = "workflows.argoproj.io/ocm-hub-workflow-uid"
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

		return containsValidOCMAnnotation(*newWf) && !reflect.DeepEqual(newWf.Status, oldWf.Status)
	},
	CreateFunc: func(e event.CreateEvent) bool {
		workflow := e.Object.(*argov1alpha1.Workflow)
		return containsValidOCMAnnotation(*workflow)
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

// Reconcile Workflow status changes and create/update a Workflow CR in the hub cluster's managed cluster namespace
// This agent only has permission to create/update in that particular namespace so the hub cluster
// will need another controller that sync the Workflow status from hub's managed cluster namespace to the original dormant Workflow.
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

	// note: this is not the orginial dormant workflow but a new/updated one that helps bridge the status sync.
	hubWorkflow := argov1alpha1.Workflow{}
	hubWorkflow.Namespace = c.clusterName
	hubWorkflow.Name = generateHubWorkflowStatusSyncName(workflow)
	err = c.hubClient.Get(ctx, types.NamespacedName{Namespace: hubWorkflow.Namespace, Name: hubWorkflow.Name}, &hubWorkflow)
	switch {
	case errors.IsNotFound(err):
		hubWorkflow.Spec.Entrypoint = "workflow-status-sync-only"
		if err = c.hubClient.Create(ctx, &hubWorkflow); err != nil {
			c.log.Error(err, "unable to create hub Workflow")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	case err != nil:
		c.log.Error(err, "unable to get hub Workflow")
		return ctrl.Result{}, err
	}

	hubWorkflow.Status = workflow.Status
	err = c.hubClient.Update(ctx, &hubWorkflow)
	if err != nil {
		c.log.Error(err, "unable to update hub Workflow status")
	}

	return ctrl.Result{}, err
}

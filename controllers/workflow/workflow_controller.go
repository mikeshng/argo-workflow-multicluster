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

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	argov1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	workv1 "open-cluster-management.io/api/work/v1"
	workflowv1alpha1 "open-cluster-management.io/argo-workflow-multicluster/api/v1alpha1"
)

const (
	// Workflow annotation that dictates which managed cluster this Workflow should be propgated to.
	AnnotationKeyOCMManagedCluster = "workflows.argoproj.io/ocm-managed-cluster"
	// Workflow annotation that dictates which managed cluster namespace this Workflow should be propgated to.
	AnnotationKeyOCMManagedClusterNamespace = "workflows.argoproj.io/ocm-managed-cluster-namespace"
	// ManifestWork annotation that shows the namespace of the hub Workflow.
	AnnotationKeyHubWorkflowNamespace = "workflows.argoproj.io/ocm-hub-workflow-namespace"
	// ManifestWork annotation that shows the name of the hub Workflow.
	AnnotationKeyHubWorkflowName = "workflows.argoproj.io/ocm-hub-workflow-name"
	// Workflow annotation that shows the first 5 characters of the dormant hub cluster Workflow
	AnnotationKeyHubWorkflowUID = "workflows.argoproj.io/ocm-hub-workflow-uid"
	// Workflow label that enables the controller to wrap the Workflow in ManifestWork payload.
	LabelKeyEnableOCMMulticluster = "workflows.argoproj.io/enable-ocm-multicluster"
	// ManifestWork label that enables the controller to sync the status of the Workflow from the managed cluster to the hub cluster.
	LabelKeyEnableOCMStatusSync = "workflows.argoproj.io/enable-ocm-status-sync"
	// FinalizerCleanupManifestWork is added to the Workflow so the associated ManifestWork gets cleaned up after a Workflow deletion.
	FinalizerCleanupManifestWork = "workflows.argoproj.io/cleanup-ocm-manifestwork"
)

// WorkflowReconciler reconciles a Workflow object
type WorkflowReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=argoproj.io,resources=workflows,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=argoproj.io,resources=workflowstatusresults,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cluster.open-cluster-management.io,resources=managedclusters,verbs=get;list;watch
//+kubebuilder:rbac:groups=work.open-cluster-management.io,resources=manifestworks,verbs=get;list;watch;create;update;patch;delete

// WorkflowPredicateFunctions defines which Workflow this controller should wrap inside ManifestWork's payload
var WorkflowPredicateFunctions = predicate.Funcs{
	UpdateFunc: func(e event.UpdateEvent) bool {
		newWorkflow := e.ObjectNew.(*argov1alpha1.Workflow)
		return containsValidOCMLabel(*newWorkflow) && containsValidOCMAnnotation(*newWorkflow)

	},
	CreateFunc: func(e event.CreateEvent) bool {
		workflow := e.Object.(*argov1alpha1.Workflow)
		return containsValidOCMLabel(*workflow) && containsValidOCMAnnotation(*workflow)
	},

	DeleteFunc: func(e event.DeleteEvent) bool {
		workflow := e.Object.(*argov1alpha1.Workflow)
		return containsValidOCMLabel(*workflow) && containsValidOCMAnnotation(*workflow)
	},
}

// SetupWithManager sets up the controller with the Manager.
func (r *WorkflowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&argov1alpha1.Workflow{}).
		WithEventFilter(WorkflowPredicateFunctions).
		Complete(r)
}

// Reconcile create/update/delete ManifestWork with the Workflow as its payload
func (r *WorkflowReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("reconciling Workflow...")

	var workflow argov1alpha1.Workflow
	if err := r.Get(ctx, req.NamespacedName, &workflow); err != nil {
		log.Error(err, "unable to fetch Workflow")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	managedClusterName := workflow.GetAnnotations()[AnnotationKeyOCMManagedCluster]
	mwName := generateManifestWorkName(workflow)

	// the Workflow is being deleted, find the ManifestWork and delete that as well
	if workflow.ObjectMeta.DeletionTimestamp != nil {
		// remove the WorkflowStatusResult in the managed cluster namespace that holds the full status
		// it might not exist so if it's not found it's ok.
		var workflowWithStatus workflowv1alpha1.WorkflowStatusResult
		err := r.Get(ctx, types.NamespacedName{Namespace: managedClusterName,
			Name: req.Name + "-" + string(workflow.UID)[0:5]}, &workflowWithStatus)
		if errors.IsNotFound(err) {
			log.Info("missing Workflow containing status")
		} else if err != nil {
			log.Error(err, "unable to fetch WorkflowStatusResult")
			return ctrl.Result{}, err
		} else if err == nil {
			if err := r.Delete(ctx, &workflowWithStatus); err != nil {
				log.Error(err, "unable to delete WorkflowStatusResult")
				return ctrl.Result{}, err
			}
		}

		// remove finalizer from Workflow but do not 'commit' yet
		if len(workflow.Finalizers) != 0 {
			f := workflow.GetFinalizers()
			for i := 0; i < len(f); i++ {
				if f[i] == FinalizerCleanupManifestWork {
					f = append(f[:i], f[i+1:]...)
					i--
				}
			}
			workflow.SetFinalizers(f)
		}

		// delete the ManifestWork associated with this Workflow
		var work workv1.ManifestWork
		err = r.Get(ctx, types.NamespacedName{Name: mwName, Namespace: managedClusterName}, &work)
		if errors.IsNotFound(err) {
			// already deleted ManifestWork, commit the Workflow finalizer removal
			if err = r.Update(ctx, &workflow); err != nil {
				log.Error(err, "unable to update Workflow")
				return ctrl.Result{}, err
			}
		} else if err != nil {
			log.Error(err, "unable to fetch ManifestWork")
			return ctrl.Result{}, err
		}

		if err := r.Delete(ctx, &work); err != nil {
			log.Error(err, "unable to delete ManifestWork")
			return ctrl.Result{}, err
		}

		// deleted ManifestWork, commit the Workflow finalizer removal
		if err := r.Update(ctx, &workflow); err != nil {
			log.Error(err, "unable to update Workflow")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// verify the ManagedCluster actually exists
	var managedCluster clusterv1.ManagedCluster
	if err := r.Get(ctx, types.NamespacedName{Name: managedClusterName}, &managedCluster); err != nil {
		log.Error(err, "unable to fetch ManagedCluster")
		return ctrl.Result{}, err
	}

	if !ContainsCleanupFinalizer(workflow) {
		log.Info("adding finalizer for Workflow")
		workflow.SetFinalizers(append(workflow.GetFinalizers(), FinalizerCleanupManifestWork))
		err := r.Client.Update(ctx, &workflow)
		if err != nil {
			log.Error(err, "unable to add finalizer to Workflow")
			return ctrl.Result{}, err
		}

		// the reconcile will retrigger from the above resource update
		return ctrl.Result{Requeue: false}, nil
	}

	log.Info("generating ManifestWork for Workflow")
	wf := prepareWorkflowForWorkPayload(workflow)
	w := generateManifestWork(mwName, managedClusterName, wf)

	// create or update the ManifestWork depends if it already exists or not
	var mw workv1.ManifestWork
	err := r.Get(ctx, types.NamespacedName{Name: mwName, Namespace: managedClusterName}, &mw)
	if errors.IsNotFound(err) {
		err = r.Client.Create(ctx, w)
		if err != nil {
			log.Error(err, "unable to create ManifestWork")
			return ctrl.Result{}, err
		}
	} else if err == nil {
		mw.Spec.Workload.Manifests = []workv1.Manifest{{RawExtension: runtime.RawExtension{Object: &wf}}}
		err = r.Client.Update(ctx, &mw)
		if err != nil {
			log.Error(err, "unable to update ManifestWork")
			return ctrl.Result{}, err
		}
	} else {
		log.Error(err, "unable to fetch ManifestWork")
		return ctrl.Result{}, err
	}

	log.Info("done reconciling Workflow")

	return ctrl.Result{}, nil
}

func ContainsCleanupFinalizer(workflow argov1alpha1.Workflow) bool {
	f := workflow.GetFinalizers()
	for _, e := range f {
		if e == FinalizerCleanupManifestWork {
			return true
		}
	}
	return false
}

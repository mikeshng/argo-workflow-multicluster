package status_sync

import (
	argov1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	workflowcontroller "open-cluster-management.io/argo-workflow-multicluster/controllers/workflow"
)

func containsValidOCMAnnotations(workflow argov1alpha1.Workflow) bool {
	annos := workflow.GetAnnotations()
	if len(annos) == 0 {
		return false
	}

	name, ok := annos[workflowcontroller.AnnotationKeyHubWorkflowName]
	if !ok || len(name) == 0 {
		return false
	}

	namespace, ok := annos[workflowcontroller.AnnotationKeyHubWorkflowNamespace]
	return ok && len(namespace) > 0
}

func generateHubWorkflowStatusResultName(workflow argov1alpha1.Workflow) string {
	uid := string(workflow.UID)
	return workflow.Name + "-" + uid[0:5]
}

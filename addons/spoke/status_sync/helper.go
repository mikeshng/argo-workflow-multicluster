package status_sync

import (
	argov1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func containsValidOCMAnnotation(workflow argov1alpha1.Workflow) bool {
	annos := workflow.GetAnnotations()
	if len(annos) == 0 {
		return false
	}

	uid, ok := annos[AnnotationKeyHubWorkflowUID]
	return ok && len(uid) > 0
}

func generateHubWorkflowStatusSyncName(workflow argov1alpha1.Workflow) string {
	uid := workflow.GetAnnotations()[AnnotationKeyHubWorkflowUID]
	return workflow.Name + "-" + uid[0:5]
}

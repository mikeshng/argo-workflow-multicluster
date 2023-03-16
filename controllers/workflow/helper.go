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

package workflow

import (
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	argov1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	workv1 "open-cluster-management.io/api/work/v1"
)

func containsValidOCMLabel(workflow argov1alpha1.Workflow) bool {
	labels := workflow.GetLabels()
	if len(labels) == 0 {
		return false
	}

	if ocmLabelStr, ok := labels[LabelKeyEnableOCMMulticluster]; ok {
		isEnable, err := strconv.ParseBool(ocmLabelStr)
		if err != nil {
			return false
		}
		return isEnable
	}

	return false
}

func containsValidOCMAnnotation(workflow argov1alpha1.Workflow) bool {
	annos := workflow.GetAnnotations()
	if len(annos) == 0 {
		return false
	}

	managedClusterName, ok := annos[AnnotationKeyOCMManagedCluster]
	return ok && len(managedClusterName) > 0
}

func containsValidOCMPlacementAnnotation(workflow argov1alpha1.Workflow) bool {
	annos := workflow.GetAnnotations()
	if len(annos) == 0 {
		return false
	}

	placementName, ok := annos[AnnotationKeyOCMPlacement]
	return ok && len(placementName) > 0
}

func containsValidOCMStatusSyncLabel(manifestWork workv1.ManifestWork) bool {
	labels := manifestWork.GetLabels()
	if len(labels) == 0 {
		return false
	}

	if ocmLabelStr, ok := labels[LabelKeyEnableOCMStatusSync]; ok {
		isEnable, err := strconv.ParseBool(ocmLabelStr)
		if err != nil {
			return false
		}
		return isEnable
	}

	return false
}

func containsValidOCMHubWorkflowAnnotation(manifestWork workv1.ManifestWork) bool {
	annos := manifestWork.GetAnnotations()
	if len(annos) == 0 {
		return false
	}

	namespace, ok := annos[AnnotationKeyHubWorkflowNamespace]
	name, ok2 := annos[AnnotationKeyHubWorkflowName]

	return ok && ok2 && len(namespace) > 0 && len(name) > 0
}

// generateWorkflowNamespace returns the intended namespace for the Workflow in the following priority
// 1) Annotation specified custom namespace
// 2) Workflow's namespace value
// 3) Fallsback to 'argo'
func generateWorkflowNamespace(workflow argov1alpha1.Workflow) string {
	annos := workflow.GetAnnotations()
	appNamespace := annos[AnnotationKeyOCMManagedClusterNamespace]
	if len(appNamespace) > 0 {
		return appNamespace
	}

	appNamespace = workflow.GetNamespace()
	if len(appNamespace) > 0 {
		return appNamespace
	}

	return "argo" // TODO find the constant value from the argo API for this field
}

// generateManifestWorkName returns the ManifestWork name for a given workflow.
// It uses the Workflow name with the suffix of the first 5 characters of the UID
func generateManifestWorkName(workflow argov1alpha1.Workflow) string {
	return workflow.Name + "-" + string(workflow.UID)[0:5]
}

// prepareWorkflowForWorkPayload modifies the Workflow:
// - reste the type and object meta
// - set the namespace value
// - empty the status
func prepareWorkflowForWorkPayload(workflow argov1alpha1.Workflow) argov1alpha1.Workflow {
	workflow.TypeMeta = metav1.TypeMeta{
		APIVersion: argov1alpha1.SchemeGroupVersion.String(),
		Kind:       argov1alpha1.WorkflowSchemaGroupVersionKind.Kind,
	}

	// TODO better handling of the managed cluster Workflow labels and annotations
	workflow.Labels[LabelKeyEnableOCMMulticluster] = "false"
	workflow.Annotations[AnnotationKeyHubWorkflowUID] = string(workflow.UID)[0:5]

	workflow.ObjectMeta = metav1.ObjectMeta{
		Name:        workflow.Name,
		Namespace:   generateWorkflowNamespace(workflow),
		Labels:      workflow.Labels,
		Annotations: workflow.Annotations,
	}

	// empty the status
	workflow.Status = argov1alpha1.WorkflowStatus{}

	return workflow
}

// generateManifestWork creates the ManifestWork that wraps the Workflow as payload
// With the status sync feedback of Workflow's phase
func generateManifestWork(name, namespace string, workflow argov1alpha1.Workflow) *workv1.ManifestWork {
	return &workv1.ManifestWork{ // TODO use OCM API helper to generate manifest work.
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{LabelKeyEnableOCMStatusSync: strconv.FormatBool(true)},
			Annotations: map[string]string{AnnotationKeyHubWorkflowNamespace: workflow.Namespace,
				AnnotationKeyHubWorkflowName: workflow.Name},
		},
		Spec: workv1.ManifestWorkSpec{
			Workload: workv1.ManifestsTemplate{
				Manifests: []workv1.Manifest{{RawExtension: runtime.RawExtension{Object: &workflow}}},
			},
			// TODO always assume the status sync addon is installed
			// ManifestConfigs: []workv1.ManifestConfigOption{
			// 	{
			// 		ResourceIdentifier: workv1.ResourceIdentifier{
			// 			Group:     argov1alpha1.SchemeGroupVersion.Group,
			// 			Resource:  "workflows", // TODO find the constant value from the argo API for this field
			// 			Namespace: workflow.Namespace,
			// 			Name:      workflow.Name,
			// 		},
			// 		FeedbackRules: []workv1.FeedbackRule{
			// 			{Type: workv1.JSONPathsType, JsonPaths: []workv1.JsonPath{{Name: "phase", Path: ".status.phase"}}},
			// 		},
			// 	},
			// },
		},
	}
}

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

package status_sync

import (
	"testing"

	argov1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	workflowcontroller "open-cluster-management.io/argo-workflow-multicluster/controllers/workflow"
)

func Test_containsValidOCMAnnotations(t *testing.T) {
	type args struct {
		workflow argov1alpha1.Workflow
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "valid OCM annotations",
			args: args{
				argov1alpha1.Workflow{
					ObjectMeta: v1.ObjectMeta{
						Annotations: map[string]string{
							workflowcontroller.AnnotationKeyHubWorkflowName:      "workflow1",
							workflowcontroller.AnnotationKeyHubWorkflowNamespace: "argo",
						},
					},
				},
			},
			want: true,
		},
		{
			name: "invalid OCM annotations",
			args: args{
				argov1alpha1.Workflow{
					ObjectMeta: v1.ObjectMeta{
						Annotations: map[string]string{
							workflowcontroller.AnnotationKeyHubWorkflowName: "workflow1",
						},
					},
				},
			},
			want: false,
		},
		{
			name: "empty value",
			args: args{
				argov1alpha1.Workflow{
					ObjectMeta: v1.ObjectMeta{
						Annotations: map[string]string{
							workflowcontroller.AnnotationKeyHubWorkflowName:      "",
							workflowcontroller.AnnotationKeyHubWorkflowNamespace: "",
						},
					},
				},
			},
			want: false,
		},
		{
			name: "no OCM annotation",
			args: args{
				argov1alpha1.Workflow{},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containsValidOCMAnnotations(tt.args.workflow); got != tt.want {
				t.Errorf("containsValidOCMAnnotations() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_generateHubWorkflowStatusResultName(t *testing.T) {
	type args struct {
		workflow argov1alpha1.Workflow
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "generate name",
			args: args{
				argov1alpha1.Workflow{
					ObjectMeta: v1.ObjectMeta{
						Name: "workflow1",
						UID:  "abcde",
					},
				},
			},
			want: "workflow1-abcde",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := generateHubWorkflowStatusResultName(tt.args.workflow); got != tt.want {
				t.Errorf("generateHubWorkflowStatusResultName() = %v, want %v", got, tt.want)
			}
		})
	}
}

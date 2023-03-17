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
	"reflect"
	"testing"

	argov1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_containsValidOCMLabel(t *testing.T) {
	type args struct {
		workflow argov1alpha1.Workflow
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "valid OCM label",
			args: args{
				argov1alpha1.Workflow{
					ObjectMeta: v1.ObjectMeta{
						Labels: map[string]string{LabelKeyEnableOCMMulticluster: "true"},
					},
				},
			},
			want: true,
		},
		{
			name: "valid OCM label case",
			args: args{
				argov1alpha1.Workflow{
					ObjectMeta: v1.ObjectMeta{
						Labels: map[string]string{LabelKeyEnableOCMMulticluster: "True"},
					},
				},
			},
			want: true,
		},
		{
			name: "invalid OCM label",
			args: args{
				argov1alpha1.Workflow{
					ObjectMeta: v1.ObjectMeta{
						Labels: map[string]string{LabelKeyEnableOCMMulticluster + "a": "true"},
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
						Labels: map[string]string{LabelKeyEnableOCMMulticluster: ""},
					},
				},
			},
			want: false,
		},
		{
			name: "false value",
			args: args{
				argov1alpha1.Workflow{
					ObjectMeta: v1.ObjectMeta{
						Labels: map[string]string{LabelKeyEnableOCMMulticluster: "false"},
					},
				},
			},
			want: false,
		},
		{
			name: "no OCM label",
			args: args{
				argov1alpha1.Workflow{},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containsValidOCMLabel(tt.args.workflow); got != tt.want {
				t.Errorf("containsValidOCMLabel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_containsValidOCMAnnotation(t *testing.T) {
	type args struct {
		workflow argov1alpha1.Workflow
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "valid OCM annotation",
			args: args{
				argov1alpha1.Workflow{
					ObjectMeta: v1.ObjectMeta{
						Annotations: map[string]string{AnnotationKeyOCMManagedCluster: "cluster1"},
					},
				},
			},
			want: true,
		},
		{
			name: "invalid OCM annotation",
			args: args{
				argov1alpha1.Workflow{
					ObjectMeta: v1.ObjectMeta{
						Annotations: map[string]string{AnnotationKeyOCMManagedCluster + "a": "cluster1"},
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
						Annotations: map[string]string{AnnotationKeyOCMManagedCluster: ""},
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
			if got := containsValidOCMAnnotation(tt.args.workflow); got != tt.want {
				t.Errorf("containsValidOCMAnnotation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_containsValidOCMPlacementAnnotation(t *testing.T) {
	type args struct {
		workflow argov1alpha1.Workflow
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "valid OCM Placement annotation",
			args: args{
				argov1alpha1.Workflow{
					ObjectMeta: v1.ObjectMeta{
						Annotations: map[string]string{AnnotationKeyOCMPlacement: "placement1"},
					},
				},
			},
			want: true,
		},
		{
			name: "invalid OCM Placement annotation",
			args: args{
				argov1alpha1.Workflow{
					ObjectMeta: v1.ObjectMeta{
						Annotations: map[string]string{AnnotationKeyOCMPlacement + "a": "placement1"},
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
						Annotations: map[string]string{AnnotationKeyOCMPlacement: ""},
					},
				},
			},
			want: false,
		},
		{
			name: "no OCM Placement annotation",
			args: args{
				argov1alpha1.Workflow{},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containsValidOCMPlacementAnnotation(tt.args.workflow); got != tt.want {
				t.Errorf("containsValidOCMPlacementAnnotation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_generateWorkflowNamespace(t *testing.T) {
	type args struct {
		workflow argov1alpha1.Workflow
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "annotation only",
			args: args{
				argov1alpha1.Workflow{
					ObjectMeta: v1.ObjectMeta{
						Annotations: map[string]string{AnnotationKeyOCMManagedClusterNamespace: "argo"},
					},
				},
			},
			want: "argo",
		},
		{
			name: "annotation and namespace",
			args: args{
				argov1alpha1.Workflow{
					ObjectMeta: v1.ObjectMeta{
						Annotations: map[string]string{AnnotationKeyOCMManagedClusterNamespace: "argo"},
						Namespace:   "default",
					},
				},
			},
			want: "argo",
		},
		{
			name: "namespace only",
			args: args{
				argov1alpha1.Workflow{
					ObjectMeta: v1.ObjectMeta{
						Namespace: "argo",
					},
				},
			},
			want: "argo",
		},
		{
			name: "annotation and namespace not found",
			args: args{
				argov1alpha1.Workflow{},
			},
			want: "argo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := generateWorkflowNamespace(tt.args.workflow); got != tt.want {
				t.Errorf("generateWorkflowNamespace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_generateManifestWorkName(t *testing.T) {
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
						UID:  "abcdefghijk",
					},
				},
			},
			want: "workflow1-abcde",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := generateManifestWorkName(tt.args.workflow); got != tt.want {
				t.Errorf("generateManifestWorkName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_prepareWorkflowForWorkPayload(t *testing.T) {
	type args struct {
		workflow argov1alpha1.Workflow
	}
	tests := []struct {
		name string
		args args
		want argov1alpha1.Workflow
	}{
		{
			name: "modified workflow",
			args: args{
				argov1alpha1.Workflow{
					ObjectMeta: v1.ObjectMeta{
						Name:        "workflow1",
						UID:         "abcdefghijk",
						Labels:      map[string]string{LabelKeyEnableOCMMulticluster: "true"},
						Annotations: map[string]string{AnnotationKeyHubWorkflowUID: "workflow1"},
					},
				},
			},
			want: argov1alpha1.Workflow{
				ObjectMeta: v1.ObjectMeta{
					Name:        "workflow1",
					Namespace:   "argo",
					Labels:      map[string]string{LabelKeyEnableOCMMulticluster: "false"},
					Annotations: map[string]string{AnnotationKeyHubWorkflowUID: "abcde"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := prepareWorkflowForWorkPayload(tt.args.workflow)
			if !reflect.DeepEqual(got.Name, tt.want.Name) {
				t.Errorf("prepareWorkflowForWorkPayload() Name = %v, want %v", got.Name, tt.want.Name)
			}
			if !reflect.DeepEqual(got.Namespace, tt.want.Namespace) {
				t.Errorf("prepareWorkflowForWorkPayload() Namespace = %v, want %v", got.Namespace, tt.want.Namespace)
			}
			if !reflect.DeepEqual(got.Labels, tt.want.Labels) {
				t.Errorf("prepareWorkflowForWorkPayload() Labels = %v, want %v", got.Labels, tt.want.Labels)
			}
			if !reflect.DeepEqual(got.Annotations, tt.want.Annotations) {
				t.Errorf("prepareWorkflowForWorkPayload() Annotations = %v, want %v", got.Annotations, tt.want.Annotations)
			}
		})
	}
}

func Test_generateManifestWork(t *testing.T) {
	workflow := argov1alpha1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      "workflow1",
			Namespace: "argo",
		},
	}

	type args struct {
		name      string
		namespace string
		workflow  argov1alpha1.Workflow
	}
	type results struct {
		workLabel map[string]string
		workAnno  map[string]string
	}
	tests := []struct {
		name string
		args args
		want results
	}{
		{
			name: "sunny",
			args: args{
				name:      "workflow1-abcde",
				namespace: "cluster1",
				workflow:  workflow,
			},
			want: results{
				workLabel: map[string]string{LabelKeyEnableOCMStatusSync: "true"},
				workAnno: map[string]string{
					AnnotationKeyHubWorkflowNamespace: "argo",
					AnnotationKeyHubWorkflowName:      "workflow1",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateManifestWork(tt.args.name, tt.args.namespace, tt.args.workflow)
			if !reflect.DeepEqual(got.Annotations, tt.want.workAnno) {
				t.Errorf("generateManifestWork() = %v, want %v", got.Annotations, tt.want.workAnno)
			}
			if !reflect.DeepEqual(got.Labels, tt.want.workLabel) {
				t.Errorf("generateManifestWork() = %v, want %v", got.Labels, tt.want.workLabel)
			}
		})
	}
}

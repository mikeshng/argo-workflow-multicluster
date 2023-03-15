package install

import (
	"context"
	"embed"
	"fmt"

	"github.com/openshift/library-go/pkg/assets"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	"open-cluster-management.io/addon-framework/pkg/addonmanager"
	"open-cluster-management.io/addon-framework/pkg/agent"
	addonapiv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	workapiv1 "open-cluster-management.io/api/work/v1"
)

var (
	genericScheme = runtime.NewScheme()
	genericCodecs = serializer.NewCodecFactory(genericScheme)
	genericCodec  = genericCodecs.UniversalDeserializer()
)

const (
	addonName = "argoworkflow-install"
)

func init() {
	utilruntime.Must(scheme.AddToScheme(genericScheme))
	utilruntime.Must(v1.AddToScheme(genericScheme))
}

//go:embed manifests
var fs embed.FS

var manifestFiles = []string{
	"manifests/00-ocm-clusterrole.yaml",        // OCM specific
	"manifests/01-ocm-clusterrolebinding.yaml", // OCM specific
	"manifests/02-argo-namespace.yaml",
	"manifests/argo-aggregate-to-admin-cr.yaml",
	"manifests/argo-aggregate-to-edit-cr.yaml",
	"manifests/argo-aggregate-to-view-cr.yaml",
	"manifests/argo-binding.yaml",
	"manifests/argo-cluster-binding.yaml",
	"manifests/argo-cluster-role.yaml",
	"manifests/argo-role-sa.yaml",
	"manifests/argo-sa.yaml",
	"manifests/argo-server-binding.yaml",
	"manifests/argo-server-cluster-role.yaml",
	"manifests/argo-server-deploy.yaml",
	"manifests/argo-server-sa.yaml",
	"manifests/argo-server-service.yaml",
	"manifests/clusterworkflowtemplates-crd.yaml",
	"manifests/cronworkflows-crd.yaml",
	"manifests/workflowartifactgctasks-crd.yaml",
	"manifests/workflow-controller-configmap.yaml",
	"manifests/workflow-controller-deploy.yaml",
	"manifests/workflow-controller-pc.yaml",
	"manifests/workfloweventbindings-crd.yaml",
	"manifests/workflows-crd.yaml",
	"manifests/workflowtaskresults-crd.yaml",
	"manifests/workflowtasksets-crd.yaml",
	"manifests/workflowtemplates-crd.yaml",
}

type argoWorkflowAgent struct {
	kubeConfig *rest.Config
}

var _ agent.AgentAddon = &argoWorkflowAgent{}

func (h *argoWorkflowAgent) Manifests(cluster *clusterv1.ManagedCluster,
	addon *addonapiv1alpha1.ManagedClusterAddOn) ([]runtime.Object, error) {
	objects := []runtime.Object{}
	for _, file := range manifestFiles {
		object, err := loadManifestFromFile(file, cluster, addon)
		if err != nil {
			return nil, err
		}
		objects = append(objects, object)
	}
	return objects, nil
}

func (h *argoWorkflowAgent) GetAgentAddonOptions() agent.AgentAddonOptions {
	return agent.AgentAddonOptions{
		AddonName:       addonName,
		InstallStrategy: agent.InstallAllStrategy("open-cluster-management-agent-addon"),
		HealthProber: &agent.HealthProber{
			Type: agent.HealthProberTypeWork,
			WorkProber: &agent.WorkHealthProber{
				ProbeFields: []agent.ProbeField{
					{
						ResourceIdentifier: workapiv1.ResourceIdentifier{
							Group:     "apps",
							Resource:  "deployments",
							Name:      "workflow-controller",
							Namespace: "argo",
						},
						ProbeRules: []workapiv1.FeedbackRule{
							{
								Type: workapiv1.WellKnownStatusType,
							},
						},
					},
				},
				HealthCheck: func(identifier workapiv1.ResourceIdentifier, result workapiv1.StatusFeedbackResult) error {
					if len(result.Values) == 0 {
						return fmt.Errorf("no values are probed for deployment %s/%s", identifier.Namespace, identifier.Name)
					}
					for _, value := range result.Values {
						if value.Name != "ReadyReplicas" {
							continue
						}

						if *value.Value.Integer >= 1 {
							return nil
						}

						return fmt.Errorf("readyReplica is %d for deployement %s/%s", *value.Value.Integer, identifier.Namespace, identifier.Name)
					}
					return fmt.Errorf("readyReplica is not probed")
				},
			},
		},
	}
}

func loadManifestFromFile(file string, cluster *clusterv1.ManagedCluster,
	addon *addonapiv1alpha1.ManagedClusterAddOn) (runtime.Object, error) {

	template, err := fs.ReadFile(file)
	if err != nil {
		return nil, err
	}

	raw := assets.MustCreateAssetFromTemplate(file, template, nil).Data
	object, _, err := genericCodec.Decode(raw, nil, nil)
	if err != nil {
		klog.ErrorS(err, "Error decoding manifest file", "filename", file)
		return nil, err
	}
	return object, nil
}

func StartControllers(ctx context.Context, config *rest.Config) error {
	mgr, err := addonmanager.New(config)
	if err != nil {
		return err
	}
	err = mgr.AddAgent(&argoWorkflowAgent{config})
	if err != nil {
		return err
	}

	err = mgr.Start(ctx)
	if err != nil {
		return err
	}

	<-ctx.Done()

	return nil
}

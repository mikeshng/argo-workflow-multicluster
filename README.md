# Argo Workflow Multicluster
Enable [Argo Workflow](https://argoproj.github.io/argo-workflows/) Multi-cluster capabilities by using
the [Open Cluster Management (OCM)](https://open-cluster-management.io/) APIs and components.

## Description
By using this project, users can propgate Argo Workflow to remote clusters based on availabile cluster resource usages.

![multi-cluster](assets/multicluster.png)

There are multiple components in this project.

- Placement controller watches the Workflow CR and evaluate the [Placement](https://open-cluster-management.io/concepts/placement/) decision then annotates the Workflow with the remote cluster target.
- ManifestWork controller watches the Workflow CR with the remote cluster target and create [ManifestWork](https://open-cluster-management.io/concepts/manifestwork/) CR which will propgate the Workflow to the intended remote cluster.
- Status controller that updates the Workflow CR status on the hub cluster with the remote cluster's Workflow's execution results.
- Install Add-on that automates the installation of Argo Workflow to all the managed clusters.
See the [Install Add-on README](addons/hub/install/README.md) for more details.

See the [Workflow example](example/hello-world.yaml) for the required label and annotation.

## Dependencies
- The Open Cluster Management (OCM) multi-cluster environment needs to be setup. See the [OCM website](https://open-cluster-management.io/) on how to setup the environment.
- In this multicluster model, OCM will provide the cluster inventory and ability to deliver workload to the remote/managed clusters.
- Optional: To enhance the Workflow status feedback from the managed cluster to hub cluster, install the [Argo Workflow Status OCM Addon](https://github.com/mikeshng/argoworkflow-status-addon). By default, only a limited amount of status information is sync back. With the Argo Workflow Status Addon installed, the entire status is sync back to the dormant Workflow on the hub cluster.

## Getting Started
1. Setup an OCM Hub cluster and registered at least one OCM Managed cluster.

2. On the hub cluster, install the OCM Argo Workflow Install Addon by running:
```
kubectl apply -f deploy/addon/hub/install/
```
This will automate the installation of Argo Workflow to the managed clusters.
For manual installation of Argo Workflow, elevate the OCM agent permission to access 
the Workflow objects by `kubectl apply -f example/managed`.
This manual privilege escalation is not necessary when using the OCM Argo Workflow Addon.

Optional: On the hub cluster, install the [OCM Argo Workflow Status Addon](https://github.com/mikeshng/argoworkflow-status-addon#install-the-argoworkflow-status-addon-to-the-hub-cluster). This will enhance the status sync from the managed cluster to 
the hub cluster.

3. On the hub cluster, install just the Argo Workflow CRD. Using the CRD from this repo:
```
kubectl apply -f hack/crds/workflows_crd.yaml
```

4. On the hub cluster, clone this project and run the controllers:
```
export KUBECONFIG=/path/to/<hub-kubeconfig>
git clone ...
cd argo-workflow-multicluster
make run
```

6. On the hub cluster, apply the ManagedClusterSetBinding and Placement.
```
kubectl apply -f example/clusterset-binding.yaml
kubectl apply -f example/workflow-placement.yaml
```

7. On the hub cluster, create the example Workflow.
```
kubectl apply -f example/hello-world.yaml
```

8. On the managed cluster, check the Workflow that was executed.
```
kubectl get workflow
NAME                       STATUS      AGE     MESSAGE
hello-world-multicluster   Succeeded   3m52s
```

9. On the hub cluster, check the ManifestWork. Replace `cluster1` namespace value with your managed cluster name
```
kubectl -n cluster1 get manifestwork -o yaml
...
        statusFeedback:
          values:
          - fieldValue:
              string: Succeeded
              type: String
            name: phase
...
```

10. On the hub cluster, check the Workflow to see the status phase is now synced from the managed cluster.
```
kubectl get workflow
NAME                       STATUS      AGE     MESSAGE
hello-world-multicluster   Succeeded
```

If the optional [OCM Argo Workflow Status Addon](https://github.com/mikeshng/argoworkflow-status-addon#install-the-argoworkflow-status-addon-to-the-hub-cluster) is installed, then you can see the full Workflow status.
```
kubectl get workflow hello-world-multicluster -o yaml 
...
status:
...
  conditions:
  - status: "False"
    type: PodRunning
  - status: "True"
    type: Completed
  finishedAt: "2022-11-12T21:57:19Z"
...
  phase: Succeeded
  progress: 1/1
  resourcesDuration:
    cpu: 9
    memory: 9
...
```

## What's next

See the OCM [Extend the multicluster scheduling capabilities with Placement API](https://open-cluster-management.io/scenarios/extend-multicluster-scheduling-capabilities/) 
documentation on how to schedule workload based on available cluster resources.

## Community, discussion, contribution, and support

Check the [CONTRIBUTING Doc](CONTRIBUTING.md) for how to contribute to the repo.

### Communication channels

Slack channel: [#open-cluster-mgmt](https://kubernetes.slack.com/channels/open-cluster-mgmt)

## License

This code is released under the Apache 2.0 license. See the file [LICENSE](LICENSE) for more information.

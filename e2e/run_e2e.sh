#!/bin/bash

set -o nounset
set -o pipefail

kubectl config use-context kind-hub
kubectl apply -f hack/crds/
kubectl apply -f deploy/argo-workflow-multicluster/
kubectl apply -f deploy/addon/install
kubectl apply -f deploy/addon/status-sync

if kubectl wait deployment -n open-cluster-management argo-workflow-multicluster --for condition=Available=True --timeout=90s; then
    echo "argo-workflow-multicluster is Available"
else
    echo "argo-workflow-multicluster is not Available"
    exit 1
fi

if kubectl wait deployment -n open-cluster-management argoworkflow-install-addon --for condition=Available=True --timeout=90s; then
    echo "argoworkflow-install-addon is Available"
else
    echo "argoworkflow-install-addon is not Available"
    exit 1
fi

if kubectl wait deployment -n open-cluster-management argoworkflow-status-sync-addon --for condition=Available=True --timeout=90s; then
    echo "argoworkflow-status-sync-addon is Available"
else
    echo "argoworkflow-status-sync-addon is not Available"
    exit 1
fi

kubectl -n default apply -f example/clusterset-binding.yaml
kubectl -n default apply -f example/workflow-placement.yaml
kubectl -n default apply -f example/hello-world.yaml

sleep 120

if kubectl -n default get workflow | grep Succeeded; then
    echo "workflow Succeeded"
else
    echo "workflow not Succeeded"
    kubectl -n default get workflow hello-world-multicluster -o yaml
    kubectl -n cluster1 get manifestwork -o yaml
    exit 1
fi

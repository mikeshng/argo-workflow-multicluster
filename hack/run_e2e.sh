#!/bin/bash

set -o nounset
set -o pipefail

kubectl config use-context kind-hub
kubectl apply -f hack/crds/

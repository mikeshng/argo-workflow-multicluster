# Argo Workflow Installation Add-on
By applying this add-on to your OCM hub cluster, the Argo Workflow installation will automatically be applied to all your existing `managed` (`spoke`) clusters and all your to be registered `managed` (`spoke`) clusters.

Under the `manifests` folder it contains the Argo Workflow install version of:

```
v3.4.2
```

# Prerequisite

Setup an Open Cluster Management environment. See: https://open-cluster-management.io/getting-started/quick-start/ for more details

# Get started

Deploy the add-on the OCM `Hub` cluster:

```
$ kubectl apply -f deploy/addon/hub/install/
$ kubectl -n open-cluster-management get deploy
NAME                         READY   UP-TO-DATE   AVAILABLE   AGE
argoworkflow-install-addon   1/1     1            1           32m
```

The controller will automatically install the add-on to all `managed` (`spoke`) clusters.

Validate the add-on is installed on a `managed` (`spoke`) cluster:

```
kubectl -n argo get deploy
NAME                  READY   UP-TO-DATE   AVAILABLE   AGE
argo-server           1/1     1            1           24s
workflow-controller   1/1     1            1           24s
```

You can also validate and check the status of the add-on on the `Hub` cluster:

```
$ kubectl -n cluster1 get managedclusteraddon # replace "cluster1" with your managed cluster name
NAME                    AVAILABLE   DEGRADED   PROGRESSING
argoworkflow-install    True                   
```

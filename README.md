zfs-operator
=============

![envtest](https://github.com/yuanying/zfs-operator/workflows/envtest/badge.svg)

zfs-operator is a light-weight Kubernetes Operator that operates ZFS Dataset on Node (currently only ZVOL).

Architecture
------------

zfs-operator manages zfs datasets using zfs cli, so this operator should be placed on each nodes.

Concepts
--------

### Volume

Volume is a representation of  ZVOL.

```yaml
apiVersion: zfs.unstable.cloud/v1alpha1
kind: Volume
metadata:
  name: volume-sample
spec:
  nodeName: node01
  volumeName: "tank/sample"
  capacity:
    storage: 5Gi
  properties:
    key: value
```

Installation
------------

zfs-operator runs within your Kubernetes cluster as a series of daemonset resources. It utilizes `CustomResourceDefinitions` to configure zfs datasets.

It is deployed using regular YAML manifests, like any other application on Kubernetes.

```bash
$ kubectl kustomize "github.com/yuanying/zfs-operator.git/config/default?ref=master" | kubectl apply -f -
```

Usually, there are only a few nodes where you want to install zfs-operator to manage zfs dataset. This manifest assumes such cases. Please label your node where you want to install this operator like following.

```bash
$ kubectl label node ${ZFS_NODE} zfs.unstable.cloud/storage=
```

If zfs-operator is installed successfully, you can see some pods in zfs-operator-system namespace like this.

```bash
$ kubectl get pod -o wide
NAME                          READY   STATUS    RESTARTS   AGE   IP            NODE            NOMINATED NODE   READINESS GATES
zfs-operator-zfs-node-74wkf   2/2     Running   42         11d   10.244.1.10   ${ZFS_NODE}     <none>           <none>
```

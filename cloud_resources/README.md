# Clusters

This directory contains the Pulumi code that defines each of the clusters in
each of the respective cloud platforms and regions.

## Requirements

- Pulumi CLI
  - `>= v3.115`
- Go
  - `>= v1.22`
- Create a Deploy token on the repository that contains your ArgoCD manifests
  - This is required for ArgoCD to have read access to the kubernetes manifests

## How it Works

Stacks are used to group configuration values for groups of clusters for a cloud provider
in a specific region. For example, for a cluster running in AWS within the US-East-1 region,
the stack name would be `aws_us-east-1`. Multiple clusters can be defined within a stack.

Cluster definitions exist in `cluster.yaml` configuration values within the `clusters` directory.
Cluster definitions contain information about the number of control plane nodes, number of
worker nodes, and the name of the cluster.

Using the CLI within this project, you can create a stack and cluster definition that will populate
the configurations with valid values, while also creating the stack for the provider and region.

### Legacy behavior

For each cluster you want to create, you need to create a stack specifically for that cluster.
The stack name will be the **same name** as the cluster name. A cluster is only provisioned if
the currently selected stack matches the cluster's name. Essentially, we want to declaratively
define each cluster, add them to a list of all the clusters in a region, and then select the
stack we want to deploy, and run a unified script that loops over all of our clusters, and
only creates/updates the currently selected stack.

## How to create a new cluster

### Create the stack for the cluster

#### With CLI

```go
go run cmd/cluster/main.go
```

### Create the cluster

```bash
# Pulumi up creates all of the resources for the currently selected stack
pulumi up
```

### Delete the cluster

```bash
# Pulumi down deletes all the resources associated with the currently selected stack
pulumi down

# Delete the stack if the clusters are no longer needed
pulumi stack rm <stack name>
```

### Global Variables

To share global values that should be shared across all stacks,
we have a `global` environment. This environment should simply export all of
it's global values, such that stacks can just import the environment to gain
access to the values.

### Getting the Kubeconfig for a Cluster

```bash
# Kubeconfig is saved as a stack output, so we can just save the config output locally
pulumi stack output <clusterID>:kubeconfig > config.yml

kubectl --kubeconfig ./config.yml get po -A
```

### ArgoCD Info

Get the Admin Secret

```bash
kubectl --kubeconfig ./config.yml -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d
```

Port forward the ArgoCD UI

```bash
kubectl --kubeconfig ./config.yml -n argocd port-forward svc/argocd-server 8080:443
```

# polyqube

A Pulumi solution for deploying hybrid K3S clusters to AWS, Azure, and GCP.

Polyqube fills in the gap for organizations that are unable to use cloud provider
specific Kubernetes distributions, like EKS, while also providing similar features found
by cloud provider Kubernetes distributions.

## Features

- High-Availability K3S deployments
- GitHub actions for deploying new clusters
- Packer and Ansible for creating golden templates for new K3S releases
- Configures cloud provider application and network load balancers for your clusters
  - Application load balancers for your Kubernetes services and ingresses
  - Network load balancers for load balancing traffic to the control plane nodes
- Supports hybrid clusters where worker nodes exist outside of the cloud platform
- Rolling upgrades of K3S versions through VM Template red/blue/green deployments
- Developer and Operator friendly CLI for creating new cluster definitions
- Support for ArgoCD for bootstrapping and managing Kubernetes resources on all the clusters

## Getting Started

- Pulumi CLI
  - `>= v3.115`
- Go
  - `>= v1.22`
- Packer
  - `>= v1.10.1`
- Ansible
  - `>= v2.16.3`
- Create a Deploy token on the repository that contains your ArgoCD manifests
  - This is required for ArgoCD to have read access to the kubernetes manifests
- Cloud provider credentials

### Create a K3S VM Template

Here is an example using Packer to create a K3S VM Template for AWS

```bash
# Init the project
packer init aws-k3s-control-plane.pkr.hcl

# Validate the project
packer validate aws-k3s-control-plane.pkr.hcl

# Build the AMI
packer build aws-k3s-control-plane.pkr.hcl
```

### Create a Cluster Definition and Stack

```bash
cd cloud_resources/

# CLI will create a valid cluster definition and Stack for the cluster
go run cmd/cluster/main.go
```

### Bring up the Cluster

```bash
pulumi stack select <cloud platform and region>
pulumi up
```

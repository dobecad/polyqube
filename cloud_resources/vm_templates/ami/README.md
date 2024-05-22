# AMI

## Requirements

- Packer CLI installed

## Getting Started

```bash
# Init the project
packer init aws-k3s-control-plane.pkr.hcl

# Validate the project
packer validate aws-k3s-control-plane.pkr.hcl

# Build the AMI
packer build aws-k3s-control-plane.pkr.hcl
```

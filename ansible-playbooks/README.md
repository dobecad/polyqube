# Ansible Playbooks

A collection of Ansible playbooks and roles for easily installing K3S for client and server machines

## Requirements

- Ansible (I'd recommend installing with `pipx install --include-deps ansible`)
- Ubuntu machine
- Root privileges

## Setup a Kubernetes Worker

This will install the necessary Nvidia dependencies for your Ubuntu machine

```bash
# If you need to use a password to switch to root user:
ansible-playbook setup-k3s-client.yml --ask-become-pass

# If you do NOT need a password to switch to root user:
ansible-playbook setup-k3s-client.yml
```
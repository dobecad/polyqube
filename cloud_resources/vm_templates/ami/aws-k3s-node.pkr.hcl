packer {
  required_plugins {
    amazon = {
      version = ">= 1.3.0"
      source  = "github.com/hashicorp/amazon"
    }
    ansible = {
      version = ">= 1.1.1"
      source  = "github.com/hashicorp/ansible"
    }

  }
}

variable "ami_name" {
  default = "polyqube-k3s-control-plane-{{timestamp}}"
}

variable "aws_region" {
  default = "us-east-2"
}

variable "base_ami_name" {
  default = "ubuntu/images/*ubuntu-jammy-22.04-amd64-server-*"
}

variable "ansible_playbook_path" {
  default = "../../../ansible-playbooks/setup-k3s-server.yml"
}

source "amazon-ebs" "ubuntu" {
  ami_name      = var.ami_name
  instance_type = "t2.large"
  region        = "us-east-2"
  source_ami_filter {
    filters = {
      name                = var.base_ami_name
      root-device-type    = "ebs"
      virtualization-type = "hvm"
    }
    most_recent = true
    owners      = ["099720109477"]
  }
  ssh_username = "ubuntu"

  tags = {
    Name = var.ami_name
  }
}

build {
  name = "Create and configure AMI"
  sources = [
    "source.amazon-ebs.ubuntu"
  ]

  provisioner "ansible" {
    playbook_file = var.ansible_playbook_path
    user = "ubuntu"
    ansible_env_vars = [ "ANSIBLE_HOST_KEY_CHECKING=False" ]
  }
}

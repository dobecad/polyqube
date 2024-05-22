package ec2

import (
	"errors"
	"fmt"
	"polyqube/pkg/aws/utils"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var (
	ErrInvalidEC2InstanceParams = errors.New("invalid or missing values used for EC2 instance creation")
)

// Create security group for EC2 cluster nodes
func K3SControlPlaneSecurityGroup(ctx *pulumi.Context, resourceName string) (*ec2.SecurityGroup, error) {
	securityGroup, err := ec2.NewSecurityGroup(ctx, resourceName, &ec2.SecurityGroupArgs{
		Description: pulumi.String("Kubernetes Security group"),
		Ingress: ec2.SecurityGroupIngressArray{
			ec2.SecurityGroupIngressArgs{
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				FromPort:   pulumi.Int(22),
				ToPort:     pulumi.Int(22),
				Protocol:   pulumi.String("tcp"),
			},
			ec2.SecurityGroupIngressArgs{
				CidrBlocks:  pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				FromPort:    pulumi.Int(0),
				ToPort:      pulumi.Int(0),
				Protocol:    pulumi.String("-1"),
				Description: pulumi.String("All"),
			},
		},
		Egress: ec2.SecurityGroupEgressArray{
			ec2.SecurityGroupEgressArgs{
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				FromPort:   pulumi.Int(22),
				ToPort:     pulumi.Int(22),
				Protocol:   pulumi.String("tcp"),
			},
			ec2.SecurityGroupEgressArgs{
				CidrBlocks:  pulumi.StringArray{pulumi.String("0.0.0.0/0")},
				FromPort:    pulumi.Int(0),
				ToPort:      pulumi.Int(0),
				Protocol:    pulumi.String("-1"),
				Description: pulumi.String("All"),
			},
		},
		Tags: pulumi.StringMap{
			"Name": pulumi.String("k3s-control-plane-node-sg"),
		},
	})
	if err != nil {
		return nil, err
	}

	return securityGroup, nil
}

// Create an SSH KeyPair to SSH into the EC2 resources
func CreateKeyPair(ctx *pulumi.Context, name string, publicKey pulumi.StringOutput) (*ec2.KeyPair, error) {
	key, err := ec2.NewKeyPair(ctx, fmt.Sprintf("%s-keypair", name), &ec2.KeyPairArgs{
		KeyName:   pulumi.String(name),
		PublicKey: publicKey,
	})

	if err != nil {
		return nil, err
	}

	return key, nil
}

type ec2InstanceOpts struct {
	ctx           *pulumi.Context
	name          string
	ami           string
	instanceType  string
	securityGroup *ec2.SecurityGroup
	az            string
	keypair       *ec2.KeyPair
}

type ec2Instance struct {
	ctx           *pulumi.Context
	name          string
	ami           string
	instanceType  string
	securityGroup *ec2.SecurityGroup
	az            string
	keypair       *ec2.KeyPair
}

func NewEC2InstanceOpts() *ec2InstanceOpts {
	return &ec2InstanceOpts{
		ctx:           nil,
		name:          "",
		ami:           "",
		instanceType:  utils.T2_large.String(),
		securityGroup: nil,
		az:            "",
		keypair:       nil,
	}
}

func (o *ec2InstanceOpts) Ctx(val *pulumi.Context) *ec2InstanceOpts {
	o.ctx = val
	return o
}

func (o *ec2InstanceOpts) Name(val string) *ec2InstanceOpts {
	o.name = val
	return o
}

func (o *ec2InstanceOpts) Ami(val string) *ec2InstanceOpts {
	o.ami = val
	return o
}

func (o *ec2InstanceOpts) InstanceType(val utils.InstanceType) *ec2InstanceOpts {
	o.instanceType = val.String()
	return o
}

func (o *ec2InstanceOpts) SecurityGroup(val *ec2.SecurityGroup) *ec2InstanceOpts {
	o.securityGroup = val
	return o
}

func (o *ec2InstanceOpts) AvailabilityZone(val string) *ec2InstanceOpts {
	o.az = val
	return o
}

func (o *ec2InstanceOpts) KeyPair(val *ec2.KeyPair) *ec2InstanceOpts {
	o.keypair = val
	return o
}

func (o *ec2InstanceOpts) Build() (*ec2Instance, error) {
	if o.ctx == nil || o.keypair == nil || o.securityGroup == nil || o.name == "" || o.az == "" {
		return nil, ErrInvalidEC2InstanceParams
	}

	ec2Instance := &ec2Instance{
		ctx:           o.ctx,
		name:          o.name,
		ami:           o.ami,
		instanceType:  o.instanceType,
		securityGroup: o.securityGroup,
		az:            o.az,
		keypair:       o.keypair,
	}
	return ec2Instance, nil
}

// Create a new EC2 Instance
func (e *ec2Instance) Create() (*ec2.Instance, error) {
	instance, err := ec2.NewInstance(e.ctx, e.name, &ec2.InstanceArgs{
		Ami:                 pulumi.String(e.ami),
		InstanceType:        pulumi.String(e.instanceType),
		VpcSecurityGroupIds: pulumi.StringArray{e.securityGroup.ID()},
		AvailabilityZone:    pulumi.String(e.az),
		Tags: pulumi.StringMap{
			"Name": pulumi.String(e.name),
		},
		KeyName: e.keypair.KeyName,
	}, pulumi.DeleteBeforeReplace(false))
	if err != nil {
		return nil, err
	}

	return instance, nil
}

package k3s

import (
	"errors"
	"fmt"
	"math"
	"polyqube/pkg/aws/alb"
	aws "polyqube/pkg/aws/ec2"
	"polyqube/pkg/aws/utils"
	sanitization "polyqube/pkg/utils"

	"polyqube/internal/config"
	k3s "polyqube/internal/k3s"
	kube "polyqube/internal/kubernetes"
	azs "polyqube/pkg/aws/availability_zones"
	"polyqube/pkg/aws/nlb"
	postgres "polyqube/pkg/aws/rds"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	DefaultControlPlaneCount = uint8(3)
	DefaultWorkerCount       = uint8(1)
)

var (
	ErrEmptyName         = errors.New("name cannot be empty")
	ErrEmptyAmi          = errors.New("ami cannot be empty")
	ErrEmptyPublicSshKey = errors.New("ssh public key cannot be empty")
)

type clusterOpts struct {
	name                 string
	numControlPlaneNodes uint8
	numWorkerNodes       uint8
	ami                  string
}

type clusterOptsBuilder struct {
	name                 string
	numControlPlaneNodes uint8
	numWorkerNodes       uint8
	ami                  string
}

// Create a set of options for defining a cluster
//
// Note that the cluster name must be equal to the stack name
func NewClusterOpts() *clusterOptsBuilder {
	opts := &clusterOptsBuilder{
		name:                 "",
		numControlPlaneNodes: DefaultControlPlaneCount,
		numWorkerNodes:       DefaultWorkerCount,
		ami:                  "",
	}
	return opts
}

// Name of the cluster. Must be exactly the same as the stack name for the cluster
func (b *clusterOptsBuilder) Name(val string) *clusterOptsBuilder {
	b.name = val
	return b
}

// Set number of control plane nodes to deploy. Must be at least 3
func (b *clusterOptsBuilder) ControlPlaneNodes(val uint8) *clusterOptsBuilder {
	val = sanitization.Clamp(val, DefaultControlPlaneCount, math.MaxUint8)
	b.numControlPlaneNodes = val
	return b
}

// Set number of cloud worker nodes to deploy. Must be at least 2
func (b *clusterOptsBuilder) WorkerNodes(val uint8) *clusterOptsBuilder {
	val = sanitization.Clamp(val, 2, math.MaxUint8)
	b.numWorkerNodes = val
	return b
}

// Set AMI for ControlPlane and Worker nodes to use
func (b *clusterOptsBuilder) Ami(val string) *clusterOptsBuilder {
	b.ami = val
	return b
}

// Create ClusterOpts that defines all configurations for a new cluster
func (b *clusterOptsBuilder) Build() (*clusterOpts, error) {
	if b.name == "" {
		return nil, ErrEmptyName
	}
	if b.ami == "" {
		return nil, ErrEmptyAmi
	}

	opts := &clusterOpts{
		name:                 b.name,
		ami:                  b.ami,
		numControlPlaneNodes: b.numControlPlaneNodes,
		numWorkerNodes:       b.numWorkerNodes,
	}
	return opts, nil
}

// Cluster definition. Should align with the clusterOpts/Builder
type cluster struct {
	name                 string
	numControlPlaneNodes uint8
	numWorkerNodes       uint8
	ami                  string
}

// Create a new cluster definition. Note that the stack name
// must the same as the cluster name for the cluster to
// be created.
func NewCluster(opts *clusterOpts) *cluster {
	cluster := &cluster{
		name:                 opts.name,
		numControlPlaneNodes: opts.numControlPlaneNodes,
		numWorkerNodes:       opts.numWorkerNodes,
		ami:                  opts.ami,
	}
	return cluster
}

// With the given Pulumi context, create the resources necessary to
// provision the cluster with the current settings
func (c *cluster) Create(ctx *pulumi.Context, isDev bool) error {
	// if stack := ctx.Stack(); stack != c.name {
	// 	fmt.Printf("Stack: %s, Cluster name: %s\n", stack, c.name)
	// 	return nil
	// }

	conf := config.LoadConfig(ctx, "aws", c.name)

	keypair, err := aws.CreateKeyPair(ctx, c.name, conf.SSHPubKey)
	if err != nil {
		return err
	}

	securityGroup, err := aws.K3SControlPlaneSecurityGroup(ctx, fmt.Sprintf("%s-control-sg", c.name))
	if err != nil {
		return err
	}

	db := postgres.NewDatabase(c.name, ctx, securityGroup, 20, conf.DatabasePassword)
	dbInstance, err := db.Create()
	if err != nil {
		return err
	}

	var nodes []*ec2.Instance
	var controlplaneNodes []*ec2.Instance
	var cloudworkerNodes []*ec2.Instance
	var outputIps pulumi.StringArray
	var controlplaneIps pulumi.StringArray
	var workerIps pulumi.StringArray

	// Get the AZs for a region, so that we can evenly disperse the nodes across AZs for high availability
	region := conf.Region
	availabilityZones := azs.GetAZsFromRegion(region)
	azLen := uint8(len(availabilityZones))

	controlPlaneNodeOpts := aws.
		NewEC2InstanceOpts().
		Ctx(ctx).
		KeyPair(keypair).
		SecurityGroup(securityGroup)
	workerNodeOpts := aws.
		NewEC2InstanceOpts().
		Ctx(ctx).
		KeyPair(keypair).
		SecurityGroup(securityGroup)

	// Create multiple EC2 instances across multiplze availability zones for HA Kubernetes cluster
	for i := uint8(0); i < c.numControlPlaneNodes; i++ {
		name := fmt.Sprintf("%s-cluster-control-plane-%d", c.name, i)
		controlPlaneNode, err := controlPlaneNodeOpts.
			Name(name).
			Ami(c.ami).
			InstanceType(utils.T3a_large).
			AvailabilityZone(availabilityZones[i%azLen]).
			Build()
		if err != nil {
			return err
		}
		node, err := controlPlaneNode.Create()
		if err != nil {
			return err
		}
		nodes = append(nodes, node)
		controlplaneNodes = append(controlplaneNodes, node)
		outputIps = append(outputIps, node.PublicIp)
		controlplaneIps = append(controlplaneIps, node.PublicIp)
	}

	for i := uint8(0); i < c.numWorkerNodes; i++ {
		name := fmt.Sprintf("%s-cluster-worker-%d", c.name, i)
		workerNode, err := workerNodeOpts.
			Name(name).
			Ami(c.ami).
			InstanceType(utils.T3a_large).
			AvailabilityZone(availabilityZones[i%azLen]).
			Build()
		if err != nil {
			return err
		}

		node, err := workerNode.Create()
		if err != nil {
			return err
		}

		nodes = append(nodes, node)
		cloudworkerNodes = append(cloudworkerNodes, node)
		outputIps = append(outputIps, node.PublicIp)
		workerIps = append(workerIps, node.PublicIp)
	}

	nlb := nlb.NewLoadBalancer(ctx, c.name, c.name, nodes)
	clusterNlb, controlPlaneTargetGroup, err := nlb.Create()
	if err != nil {
		return err
	}

	alb := alb.NewLoadBalancer(ctx, c.name, c.name, cloudworkerNodes)
	clusterAlb, workerTargetGroup, err := alb.Create()
	if err != nil {
		return err
	}

	clusterSetup, err := k3s.
		NewClusterSetup(c.name).
		Ctx(ctx).
		ControlPlaneNodes(controlplaneNodes).
		WorkerNodes(cloudworkerNodes).
		Database(dbInstance).
		NetworkLoadBalancer(clusterNlb).
		NetworkLoadBalancerTargetGroup(controlPlaneTargetGroup).
		ApplicationLoadBalancer(clusterAlb).
		ApplicationLoadBalancerTargetGroup(workerTargetGroup).
		Config(conf).
		Build()
	if err != nil {
		return err
	}

	kubeconfig, clusterStartDependencies, err := clusterSetup.Setup()
	if err != nil {
		return err
	}

	kubernetes, err := kubernetes.NewProvider(ctx, fmt.Sprintf("%s-k8s-conf", c.name), &kubernetes.ProviderArgs{
		Kubeconfig: kubeconfig,
	})
	if err != nil {
		return err
	}

	bootstrapper := kube.NewBootstrapper(ctx, kubernetes, c.name, clusterSetup, clusterStartDependencies)
	bootstrapper.Setup()

	ctx.Export(fmt.Sprintf("%s:leader", c.name), controlplaneIps[0])
	ctx.Export(fmt.Sprintf("%s:control_plane", c.name), controlplaneIps)
	ctx.Export(fmt.Sprintf("%s:workers", c.name), workerIps)
	ctx.Export(fmt.Sprintf("%s:all", c.name), outputIps)
	ctx.Export(fmt.Sprintf("%s:databaseEndpoint", c.name), dbInstance.Endpoint)
	ctx.Export(fmt.Sprintf("%s:networkloadbalancerDnsName", c.name), clusterNlb.DnsName)
	ctx.Export(fmt.Sprintf("%s:applicationloadbalancerDnsName", c.name), clusterAlb.DnsName)
	ctx.Export(fmt.Sprintf("%s:workerJoinCmd", c.name), pulumi.Sprintf("curl -sfL https://get.k3s.io | K3S_URL=https://%s:6443 K3S_TOKEN=%s sh -s - --node-external-ip=<your pub IPv4> --node-label \"nvidia-worker=true\"", clusterNlb.DnsName, conf.AgentToken))

	return nil
}

package utils

import (
	"errors"
	"fmt"
	"strings"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/alb"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/rds"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"

	"polyqube/internal/config"
)

const (
	vmUsername = pulumi.String("ubuntu")
	sshPort    = pulumi.Float64(22)
)

var (
	ErrInsufficientClusterRequirements  = errors.New("setup cannot accept nil values and must contain more than 1 Control Plane node")
	ErrInvalidNumberOfControlPlaneNodes = errors.New("must have more than one control plane node")
)

type ClusterSetupBuilder struct {
	ctx               *pulumi.Context
	clusterName       string
	controlplaneNodes []*ec2.Instance
	workerNodes       []*ec2.Instance
	db                *rds.Instance
	nlb               *alb.LoadBalancer
	alb               *alb.LoadBalancer
	nlbTargetGroup    *alb.TargetGroup
	albTargetGroup    *alb.TargetGroup
	config            *config.StackConfig
}

type ClusterSetup struct {
	ctx               *pulumi.Context
	clusterName       string
	controlplaneNodes []*ec2.Instance
	workerNodes       []*ec2.Instance
	db                *rds.Instance
	nlb               *alb.LoadBalancer
	alb               *alb.LoadBalancer
	nlbTargetGroup    *alb.TargetGroup
	albTargetGroup    *alb.TargetGroup
	config            *config.StackConfig
}

// Setup a HA K3S cluster on top of newly provisioned infrastructure
func NewClusterSetup(clusterName string) *ClusterSetupBuilder {
	// TODO: Create an interface such that the DB and LB are generic.
	// This could just be an interface like the one I use in sanitization.go
	return &ClusterSetupBuilder{
		ctx:               nil,
		clusterName:       clusterName,
		controlplaneNodes: []*ec2.Instance{},
		workerNodes:       []*ec2.Instance{},
		db:                nil,
		nlb:               nil,
		alb:               nil,
		config:            nil,
	}
}

// Set the pulumi context
func (c *ClusterSetupBuilder) Ctx(val *pulumi.Context) *ClusterSetupBuilder {
	c.ctx = val
	return c
}

// Set the references to the ec2 machines that make up the control plane
func (c *ClusterSetupBuilder) ControlPlaneNodes(val []*ec2.Instance) *ClusterSetupBuilder {
	c.controlplaneNodes = val
	return c
}

// Set the references to the ec2 machines that make up the worker nodes
func (c *ClusterSetupBuilder) WorkerNodes(val []*ec2.Instance) *ClusterSetupBuilder {
	c.workerNodes = val
	return c
}

// Pass in the reference to the RDS database
func (c *ClusterSetupBuilder) Database(val *rds.Instance) *ClusterSetupBuilder {
	c.db = val
	return c
}

// Pass in the reference to the cluster loadbalancer
func (c *ClusterSetupBuilder) NetworkLoadBalancer(val *alb.LoadBalancer) *ClusterSetupBuilder {
	c.nlb = val
	return c
}

// Pass in the reference to the loadbalancers target group
func (c *ClusterSetupBuilder) NetworkLoadBalancerTargetGroup(val *alb.TargetGroup) *ClusterSetupBuilder {
	c.nlbTargetGroup = val
	return c
}

// Pass in the reference to the cluster loadbalancer
func (c *ClusterSetupBuilder) ApplicationLoadBalancer(val *alb.LoadBalancer) *ClusterSetupBuilder {
	c.alb = val
	return c
}

// Pass in the reference to the loadbalancers target group
func (c *ClusterSetupBuilder) ApplicationLoadBalancerTargetGroup(val *alb.TargetGroup) *ClusterSetupBuilder {
	c.albTargetGroup = val
	return c
}

// Pass in a reference to the cluster's stack config
func (c *ClusterSetupBuilder) Config(val *config.StackConfig) *ClusterSetupBuilder {
	c.config = val
	return c
}

// Build the cluster setup options
func (c *ClusterSetupBuilder) Build() (*ClusterSetup, error) {
	if c.ctx == nil || len(c.controlplaneNodes) < 2 || c.db == nil || c.nlb == nil || c.alb == nil || c.nlbTargetGroup == nil || c.config == nil {
		return nil, ErrInsufficientClusterRequirements
	}

	clusterSetup := &ClusterSetup{
		ctx:               c.ctx,
		clusterName:       c.clusterName,
		controlplaneNodes: c.controlplaneNodes,
		workerNodes:       c.workerNodes,
		db:                c.db,
		nlb:               c.nlb,
		alb:               c.alb,
		nlbTargetGroup:    c.nlbTargetGroup,
		albTargetGroup:    c.albTargetGroup,
		config:            c.config,
	}

	return clusterSetup, nil
}

// Remotely connect to the given node
func (c *ClusterSetup) ConnectToNode(node *ec2.Instance) *remote.ConnectionArgs {
	conn := &remote.ConnectionArgs{
		User:       vmUsername,
		Host:       node.PublicIp,
		Port:       sshPort,
		PrivateKey: c.config.SSHPrivKey,
	}

	return conn
}

// Utility for other resources that need to perform some action on the leader node
func (c *ClusterSetup) ConnectToLeaderNode() *remote.ConnectionArgs {
	leader := c.controlplaneNodes[0]

	conn := &remote.ConnectionArgs{
		User:       vmUsername,
		Host:       leader.PublicIp,
		Port:       sshPort,
		PrivateKey: c.config.SSHPrivKey,
	}

	return conn
}

// Bootstrap the initial control plane node, which creates the K3S cluster
func (c *ClusterSetup) setupK3SServer(resourceName string, conn *remote.ConnectionArgs, leaderNode *ec2.Instance) (*remote.Command, error) {
	deleteLine := "sudo sed -i -e \"\\$d\" /etc/systemd/system/k3s.service"
	dbConnectURI := pulumi.Sprintf("postgres://%s:%s@%s/k3s", "polyqube", c.config.DatabasePassword, c.db.Endpoint)
	updateService := pulumi.Sprintf("echo \"ExecStart=/usr/local/bin/k3s server --token %s --agent-token %s --node-external-ip=%s --datastore-endpoint='%s' --tls-san %s --tls-san %s --node-taint CriticalAddonsOnly=true:NoExecute --disable=traefik --flannel-backend=wireguard-native --flannel-external-ip\" | sudo tee -a /etc/systemd/system/k3s.service > /dev/null", c.config.Token, c.config.AgentToken, conn.Host, dbConnectURI, c.nlb.DnsName, c.alb.DnsName)
	command := pulumi.Sprintf("(%s); (%s); (%s); (%s)", deleteLine, deleteLine, deleteLine, updateService)
	comm, err := remote.NewCommand(c.ctx, resourceName, &remote.CommandArgs{
		Connection: conn,
		Create:     command,
	}, pulumi.DependsOn([]pulumi.Resource{leaderNode, c.db}))

	if err != nil {
		return nil, err
	}

	return comm, nil

}

// Enable the K3S service so that K3S starts on node reboot
func (c *ClusterSetup) enableK3SServerService(resourceName string, conn *remote.ConnectionArgs, dependsOn []pulumi.Resource) (*remote.Command, error) {
	comm, err := remote.NewCommand(c.ctx, resourceName, &remote.CommandArgs{
		Connection: conn,
		Create:     pulumi.String("sudo systemctl enable k3s.service"),
	}, pulumi.DependsOn(dependsOn))

	if err != nil {
		return nil, err
	}

	return comm, nil
}

// Start the K3S service
func (c *ClusterSetup) startK3SService(resourceName string, conn *remote.ConnectionArgs, dependsOn []pulumi.Resource) (*remote.Command, error) {
	comm, err := remote.NewCommand(c.ctx, resourceName, &remote.CommandArgs{
		Connection: conn,
		Create:     pulumi.String("sudo systemctl start k3s.service"),
	}, pulumi.DependsOn(dependsOn))

	if err != nil {
		return nil, err
	}

	return comm, nil
}

// Join a control plane node to the cluster
func (c *ClusterSetup) controlplaneNodeJoinCluster(resourceName string, conn *remote.ConnectionArgs, dependsOn []pulumi.Resource) (*remote.Command, error) {
	// TODO: Add database username to config
	deleteLine := "sudo sed -i -e \"\\$d\" /etc/systemd/system/k3s.service"
	dbConnectURI := pulumi.Sprintf("postgres://%s:%s@%s/k3s", "polyqube", c.config.DatabasePassword, c.db.Endpoint)
	updateService := pulumi.Sprintf("echo \"ExecStart=/usr/local/bin/k3s server --server https://%s:6443 --token %s --agent-token %s --node-external-ip=%s --datastore-endpoint='%s' --node-taint CriticalAddonsOnly=true:NoExecute --disable=traefik --flannel-backend=wireguard-native --flannel-external-ip\" | sudo tee -a /etc/systemd/system/k3s.service > /dev/null", c.nlb.DnsName, c.config.Token, c.config.AgentToken, conn.Host, dbConnectURI)
	command := pulumi.Sprintf("(%s); (%s); (%s); (%s)", deleteLine, deleteLine, deleteLine, updateService)

	comm, err := remote.NewCommand(c.ctx, resourceName, &remote.CommandArgs{
		Connection: conn,
		Create:     command,
	}, pulumi.DependsOn(dependsOn))

	if err != nil {
		return nil, err
	}

	return comm, nil
}

// Join the worker nodes to the cluster
func (c *ClusterSetup) workerJoinCluster(resourceName string, conn *remote.ConnectionArgs, dependsOn []pulumi.Resource) (*remote.Command, error) {
	deleteLine := "sudo sed -i -e \"\\$d\" /etc/systemd/system/k3s.service"
	updateService := pulumi.Sprintf("echo \"ExecStart=/usr/local/bin/k3s agent --server https://%s:6443 --token %s --node-label cloud-worker=true --node-external-ip=%s\" | sudo tee -a /etc/systemd/system/k3s.service > /dev/null", c.nlb.DnsName, c.config.AgentToken, conn.Host)
	command := pulumi.Sprintf("(%s); (%s); (%s); (%s)", deleteLine, deleteLine, deleteLine, updateService)

	comm, err := remote.NewCommand(c.ctx, resourceName, &remote.CommandArgs{
		Connection: conn,
		Create:     command,
	}, pulumi.DependsOn(dependsOn))

	if err != nil {
		return nil, err
	}

	return comm, nil

}

// Perform the necessary steps to bootstrap the leader node, to create the K3S cluster
func (c *ClusterSetup) setupInititalLeader() (pulumi.StringOutput, *remote.Command, error) {
	leaderNode := c.controlplaneNodes[0]
	conn := c.ConnectToNode(leaderNode)

	setupComm, err := c.setupK3SServer(fmt.Sprintf("%s-setupInititalLeader", c.clusterName), conn, leaderNode)
	if err != nil {
		return pulumi.String("").ToStringOutput(), nil, err
	}

	enableComm, err := c.enableK3SServerService(fmt.Sprintf("%s-enableLeaderK3SService", c.clusterName), conn, []pulumi.Resource{setupComm})
	if err != nil {
		return pulumi.String("").ToStringOutput(), nil, err
	}

	startComm, err := c.startK3SService(fmt.Sprintf("%s-startLeaderK3SService", c.clusterName), conn, []pulumi.Resource{setupComm, enableComm})
	if err != nil {
		return pulumi.String("").ToStringOutput(), nil, err
	}

	kubeconfig, err := c.exportKubeconfig(conn, []pulumi.Resource{startComm})
	if err != nil {
		return pulumi.String("").ToStringOutput(), nil, err
	}
	return kubeconfig, startComm, nil
}

// Join control plane nodes to the cluster
func (c *ClusterSetup) joinControlPlaneNodesToCluster(dependsOn []pulumi.Resource) ([]pulumi.Resource, error) {
	var dependencies []pulumi.Resource
	var resultingDependencies []pulumi.Resource

	leaderNode := c.controlplaneNodes[0]
	dependencies = append(dependencies, []pulumi.Resource{leaderNode, c.db, c.nlb}...)
	dependencies = append(dependencies, dependsOn...)

	if len(c.controlplaneNodes) < 2 {
		return dependencies, ErrInvalidNumberOfControlPlaneNodes
	}

	for index, node := range c.controlplaneNodes[1:] {
		conn := c.ConnectToNode(node)
		resourceName := fmt.Sprintf("%s-%s-%d", c.clusterName, "joinControlNodeToLeader", index)

		setupComm, err := c.controlplaneNodeJoinCluster(resourceName, conn, dependencies)
		if err != nil {
			return dependencies, err
		}

		resourceName = fmt.Sprintf("%s-%s-%d", c.clusterName, "enableControlK3SService", index)
		enableComm, err := c.enableK3SServerService(resourceName, conn, []pulumi.Resource{setupComm})
		if err != nil {
			return dependencies, err
		}

		resourceName = fmt.Sprintf("%s-%s-%d", c.clusterName, "startControlK3SService", index)
		svc, err := c.startK3SService(resourceName, conn, []pulumi.Resource{setupComm, enableComm})
		if err != nil {
			return dependencies, err
		}
		resultingDependencies = append(resultingDependencies, svc)
	}

	return resultingDependencies, nil
}

// Join worker nodes to the cluster
func (c *ClusterSetup) joinWorkerNodesToCluster(dependsOn []pulumi.Resource) ([]pulumi.Resource, error) {
	var dependencies []pulumi.Resource
	var resultingDependencies []pulumi.Resource

	leaderNode := c.controlplaneNodes[0]
	dependencies = append(dependencies, []pulumi.Resource{leaderNode, c.db, c.nlb}...)
	dependencies = append(dependencies, dependsOn...)

	for index, node := range c.workerNodes {
		conn := c.ConnectToNode(node)

		// We don't need to depend on the K3S cluster commands here, since the worker nodes will just
		// keep retrying to join the cluster
		resourceName := fmt.Sprintf("%s-%s-%d", c.clusterName, "joinWorkerToLeader", index)
		setupComm, err := c.workerJoinCluster(resourceName, conn, dependencies)
		if err != nil {
			return dependencies, err
		}

		resourceName = fmt.Sprintf("%s-%s-%d", c.clusterName, "enableWorkerK3SService", index)
		enableComm, err := c.enableK3SServerService(resourceName, conn, []pulumi.Resource{setupComm})
		if err != nil {
			return dependencies, err
		}

		resourceName = fmt.Sprintf("%s-%s-%d", c.clusterName, "startWorkerK3SService", index)
		svc, err := c.startK3SService(resourceName, conn, []pulumi.Resource{setupComm, enableComm})
		if err != nil {
			return dependencies, err
		}
		resultingDependencies = append(resultingDependencies, svc)
	}

	return resultingDependencies, nil
}

// Join the Control Plane leader to the network loadbalancer first, so that other
// control plane nodes that join through the loadbalancer are guranteed to have their
// traffic sent to only the leader node initially
func (c *ClusterSetup) joinControlPlaneLeaderToNetworkLoadBalancer() (*alb.TargetGroupAttachment, error) {
	// We want to make sure the leader is attached to the target group before we forward traffic to
	// the network loadbalancer
	leader := c.controlplaneNodes[0]

	resourceName := fmt.Sprintf("%s-leader-attachment", c.clusterName)
	attachment, err := alb.NewTargetGroupAttachment(c.ctx, resourceName, &alb.TargetGroupAttachmentArgs{
		TargetGroupArn: c.nlbTargetGroup.Arn,
		TargetId:       leader.ID(),
		Port:           pulumi.Int(6443),
	})
	if err != nil {
		return nil, err
	}
	return attachment, nil
}

// Join the K3S Control Plane nodes to the target group of the cluster's loadbalancer
//
// TODO: This will need to be moved outside of internal, since it depends on the loadbalancer type
func (c *ClusterSetup) joinControlPlaneNodesToNetworkLoadBalancer(dependsOn []pulumi.Resource) ([]pulumi.Resource, error) {
	var attachments []pulumi.Resource

	for index, node := range c.controlplaneNodes[1:] {
		resourceName := fmt.Sprintf("%s-%s-%d", c.clusterName, "control-target", index)
		attachment, err := alb.NewTargetGroupAttachment(c.ctx, resourceName, &alb.TargetGroupAttachmentArgs{
			TargetGroupArn: c.nlbTargetGroup.Arn,
			TargetId:       node.ID(),
			Port:           pulumi.Int(6443),
		}, pulumi.DependsOn(dependsOn))
		if err != nil {
			return nil, err
		}
		attachments = append(attachments, attachment)
	}
	return attachments, nil
}

func (c *ClusterSetup) joinWorkerNodesToApplicationLoadBalancer() error {
	for index, node := range c.workerNodes {
		resourceName := fmt.Sprintf("%s-%s-%d", c.clusterName, "worker-target", index)
		if _, err := alb.NewTargetGroupAttachment(c.ctx, resourceName, &alb.TargetGroupAttachmentArgs{
			TargetGroupArn: c.albTargetGroup.Arn,
			TargetId:       node.ID(),
			Port:           pulumi.Int(32080),
		}); err != nil {
			return err
		}
	}
	return nil

}

// Export the cluster's kubeconfig to the pulumi stack output
func (c *ClusterSetup) exportKubeconfig(conn *remote.ConnectionArgs, dependsOn []pulumi.Resource) (pulumi.StringOutput, error) {
	command := "(sleep 5); (sudo cat /etc/rancher/k3s/k3s.yaml);"

	comm, err := remote.NewCommand(c.ctx, fmt.Sprintf("%s-fetchKubeconf", c.clusterName), &remote.CommandArgs{
		Connection: conn,
		Create:     pulumi.String(command),
	}, pulumi.DependsOn(dependsOn))

	if err != nil {
		return pulumi.String("").ToStringOutput(), err
	}

	kubeconfig := comm.Stdout
	lbDnsName := pulumi.Sprintf("https://%s:6443", c.nlb.DnsName)
	combined := pulumi.All(kubeconfig, lbDnsName)

	lbKubeconfig := combined.ApplyT(func(args interface{}) (string, error) {
		tuple := args.([]interface{})
		original := tuple[0].(string)
		replacement := tuple[1].(string)
		return strings.Replace(original, "https://127.0.0.1:6443", replacement, -1), nil
	}).(pulumi.StringOutput)

	c.ctx.Export("kubeconfig", lbKubeconfig)
	return lbKubeconfig, nil
}

// Run the necessary steps to turn the infrastructure into a functioning HA K3S cluster
//
// Involves bootstrapping the initial leader node, joining control plane nodes to the
// leader node, joining workers to the leader node, and joining the control plane nodes
// to the target group for the cluster loadbalancer.
func (c *ClusterSetup) Setup() (pulumi.StringOutput, []pulumi.Resource, error) {
	var dependencies []pulumi.Resource

	kubeconfig, startComm, err := c.setupInititalLeader()
	if err != nil {
		return pulumi.String("").ToStringOutput(), dependencies, err
	}

	leaderAttachment, err := c.joinControlPlaneLeaderToNetworkLoadBalancer()
	if err != nil {
		return pulumi.String("").ToStringOutput(), dependencies, err
	}

	// Only join other control plane nodes when the leader has joined the loadbalancer target group
	// This ensures all cluster join traffic is being sent to the running leader node
	joinedControlPlaneNodes, err := c.joinControlPlaneNodesToCluster([]pulumi.Resource{startComm, leaderAttachment})
	if err != nil {
		return pulumi.String("").ToStringOutput(), dependencies, err
	}

	clusterJoinDependencies := []pulumi.Resource{leaderAttachment}
	clusterJoinDependencies = append(clusterJoinDependencies, joinedControlPlaneNodes...)

	attachment, err := c.joinControlPlaneNodesToNetworkLoadBalancer(clusterJoinDependencies)
	if err != nil {
		return pulumi.String("").ToStringOutput(), dependencies, err
	}

	if err := c.joinWorkerNodesToApplicationLoadBalancer(); err != nil {
		return pulumi.String("").ToStringOutput(), dependencies, err
	}

	joinedWorkerNodes, err := c.joinWorkerNodesToCluster(clusterJoinDependencies)
	if err != nil {
		return pulumi.String("").ToStringOutput(), dependencies, err
	}

	dependencies = append(dependencies, attachment...)
	dependencies = append(dependencies, joinedControlPlaneNodes...)
	dependencies = append(dependencies, joinedWorkerNodes...)

	return kubeconfig, dependencies, err
}

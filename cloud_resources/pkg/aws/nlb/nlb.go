package nlb

import (
	"fmt"

	"polyqube/internal/config"
	"polyqube/pkg/aws/vpc"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/alb"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type LoadBalancer struct {
	ctx         *pulumi.Context
	name        string
	clusterName string
	subnets     pulumi.StringArray
}

// Initialize a new Network LoadBalancer
func NewLoadBalancer(ctx *pulumi.Context, name, clusterName string, targets []*ec2.Instance) *LoadBalancer {
	if len(targets) == 0 {
		panic("Must have at least one target!")
	}

	var subnetIds pulumi.StringArray
	for _, instance := range targets {
		subnetIds = append(subnetIds, instance.SubnetId)
	}

	return &LoadBalancer{
		ctx:         ctx,
		name:        name,
		clusterName: clusterName,
		subnets:     subnetIds,
	}
}

// Create the Network Loadbalancer
func (l *LoadBalancer) Create() (*alb.LoadBalancer, *alb.TargetGroup, error) {
	conf := config.LoadConfig(l.ctx, "aws", l.clusterName)

	defaultVpc, err := vpc.GetDefaultVpc(l.ctx)
	if err != nil {
		return nil, nil, err
	}

	resourceName := fmt.Sprintf("%s-lb", l.name)
	lb, err := alb.NewLoadBalancer(l.ctx, resourceName, &alb.LoadBalancerArgs{
		Subnets:          l.subnets,
		LoadBalancerType: pulumi.String("network"),
	}, pulumi.Protect(conf.ProtectLB))
	if err != nil {
		return nil, nil, err
	}

	resourceName = fmt.Sprintf("%s-tg", l.name)
	targetGroup, err := alb.NewTargetGroup(l.ctx, resourceName, &alb.TargetGroupArgs{
		Port:       pulumi.Int(6443),
		Protocol:   pulumi.String("TCP"),
		VpcId:      pulumi.String(defaultVpc.Id),
		TargetType: pulumi.String("instance"),
	}, pulumi.DependsOn([]pulumi.Resource{lb}))
	if err != nil {
		return nil, nil, err
	}

	resourceName = fmt.Sprintf("%s-listener", l.name)
	if _, err := alb.NewListener(l.ctx, resourceName, &alb.ListenerArgs{
		LoadBalancerArn: lb.Arn,
		Port:            pulumi.Int(6443),
		Protocol:        pulumi.String("TCP"),
		DefaultActions: alb.ListenerDefaultActionArray{
			&alb.ListenerDefaultActionArgs{
				Type:           pulumi.String("forward"),
				TargetGroupArn: targetGroup.Arn,
			},
		},
	}, pulumi.DependsOn([]pulumi.Resource{targetGroup, lb})); err != nil {
		return nil, nil, err
	}

	return lb, targetGroup, nil
}

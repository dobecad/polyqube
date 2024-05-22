package alb

import (
	"fmt"

	"polyqube/pkg/aws/vpc"

	"polyqube/internal/config"

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

// Initialize a new Application LoadBalancer
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

// Create the Application Loadbalancer
func (l *LoadBalancer) Create() (*alb.LoadBalancer, *alb.TargetGroup, error) {
	conf := config.LoadConfig(l.ctx, "aws", l.clusterName)

	defaultVpc, err := vpc.GetDefaultVpc(l.ctx)
	if err != nil {
		return nil, nil, err
	}

	resourceName := fmt.Sprintf("%s-alb", l.name)
	lb, err := alb.NewLoadBalancer(l.ctx, resourceName, &alb.LoadBalancerArgs{
		Subnets:          l.subnets,
		LoadBalancerType: pulumi.String("application"),
	}, pulumi.Protect(conf.ProtectLB))
	if err != nil {
		return nil, nil, err
	}

	resourceName = fmt.Sprintf("%s-workers-tg", l.name)
	targetGroup, err := alb.NewTargetGroup(l.ctx, resourceName, &alb.TargetGroupArgs{
		// For HTTPS
		// Port:       pulumi.Int(443),
		// Protocol:   pulumi.String("HTTPS"),
		Port:       pulumi.Int(32080),
		Protocol:   pulumi.String("HTTP"),
		VpcId:      pulumi.String(defaultVpc.Id),
		TargetType: pulumi.String("instance"),
		HealthCheck: &alb.TargetGroupHealthCheckArgs{
			Enabled:            pulumi.Bool(true),
			Path:               pulumi.String("/ping"),
			Port:               pulumi.String("32080"),
			Protocol:           pulumi.String("HTTP"),
			Timeout:            pulumi.Int(6),
			UnhealthyThreshold: pulumi.Int(4),
		},
	}, pulumi.DependsOn([]pulumi.Resource{lb}))
	if err != nil {
		return nil, nil, err
	}

	resourceName = fmt.Sprintf("%s-workers-listener", l.name)
	if _, err := alb.NewListener(l.ctx, resourceName, &alb.ListenerArgs{
		LoadBalancerArn: lb.Arn,
		// For HTTPS
		// Port:            pulumi.Int(443),
		// Protocol:        pulumi.String("HTTPS"),
		// SslPolicy: pulumi.String("ELBSecurityPolicy-TLS13-1-2-2021-06"),
		Port:     pulumi.Int(80),
		Protocol: pulumi.String("HTTP"),
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

package postgres

import (
	"fmt"
	"math"
	"polyqube/internal/config"
	sanitization "polyqube/pkg/utils"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/rds"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type database struct {
	clusterName          string
	ctx                  *pulumi.Context
	dbSecGroup           *ec2.SecurityGroup
	controlPlaneSecGroup *ec2.SecurityGroup
	storageSize          uint8
	databasePassword     pulumi.StringOutput
}

func NewDatabase(clusterName string, ctx *pulumi.Context, controlPlaneSecGroup *ec2.SecurityGroup, storageSize uint8, databasePassword pulumi.StringOutput) *database {
	storageSize = sanitization.Clamp(storageSize, 20, math.MaxUint8)

	db := &database{
		clusterName:          clusterName,
		ctx:                  ctx,
		dbSecGroup:           nil,
		controlPlaneSecGroup: controlPlaneSecGroup,
		storageSize:          storageSize,
		databasePassword:     databasePassword,
	}
	return db
}

func rdsSecurityGroup(ctx *pulumi.Context, clusterName string) (*ec2.SecurityGroup, error) {
	secGroupName := fmt.Sprintf("%s-rds-sg", clusterName)
	secGroup, err := ec2.NewSecurityGroup(ctx, secGroupName, &ec2.SecurityGroupArgs{
		Description: pulumi.String("Allow access to the PostgreSQL RDS instance from EC2 instances"),
	})

	if err != nil {
		return nil, err
	}

	return secGroup, nil
}

func rdsSecurityGroupRule(ctx *pulumi.Context, clusterName string, rdsSecGroup, controlPlaneSecGroup *ec2.SecurityGroup) error {
	secGroupRuleName := fmt.Sprintf("%s-rds-sgr", clusterName)
	_, err := ec2.NewSecurityGroupRule(ctx, secGroupRuleName, &ec2.SecurityGroupRuleArgs{
		Type:                  pulumi.String("ingress"),
		FromPort:              pulumi.Int(5432),
		ToPort:                pulumi.Int(5432),
		Protocol:              pulumi.String("tcp"),
		SecurityGroupId:       rdsSecGroup.ID(),
		SourceSecurityGroupId: controlPlaneSecGroup.ID(),
	})

	if err != nil {
		return err
	}

	return nil
}

func (d *database) Create() (*rds.Instance, error) {
	conf := config.LoadConfig(d.ctx, "aws", d.clusterName)

	securityGroup, err := rdsSecurityGroup(d.ctx, d.clusterName)
	if err != nil {
		return nil, err
	}

	if err := rdsSecurityGroupRule(d.ctx, d.clusterName, securityGroup, d.controlPlaneSecGroup); err != nil {
		return nil, err
	}

	dbName := fmt.Sprintf("%s-database", d.clusterName)
	db, err := rds.NewInstance(d.ctx, dbName, &rds.InstanceArgs{
		Engine:              pulumi.String("postgres"),
		InstanceClass:       pulumi.String("db.m5.xlarge"),
		AllocatedStorage:    pulumi.Int(20),
		Username:            pulumi.String("polyqube"),
		Password:            d.databasePassword,
		VpcSecurityGroupIds: pulumi.StringArray{d.controlPlaneSecGroup.ID()},
		SkipFinalSnapshot:   pulumi.Bool(true),
	}, pulumi.Protect(conf.ProtectDB))

	if err != nil {
		return nil, err
	}

	return db, nil
}

package vpc

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Load the default VPC for the current AWS region
//
// This exists primarily because we are only using the default
// VPC at the moment. We might not even need to create a VPC for
// the scope of this project.
func GetDefaultVpc(ctx *pulumi.Context) (*ec2.LookupVpcResult, error) {
	defaultVpc, err := ec2.LookupVpc(ctx, &ec2.LookupVpcArgs{
		Default: pulumi.BoolRef(true),
	})
	if err != nil {
		return nil, err
	}

	return defaultVpc, nil
}

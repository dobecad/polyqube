package config

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	pConfig "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

type StackConfig struct {
	Region           string
	SSHPubKey        pulumi.StringOutput
	SSHPrivKey       pulumi.StringOutput
	Token            pulumi.StringOutput
	AgentToken       pulumi.StringOutput
	DatabasePassword pulumi.StringOutput
	ProtectLB        bool
	ProtectDB        bool
}

// Load the necessary values from the Stack config
func LoadConfig(ctx *pulumi.Context, cloudPlatform string, clusterName string) *StackConfig {
	conf := pConfig.New(ctx, cloudPlatform)

	region := conf.Require("region")
	if region == "" {
		panic("Region is empty!")
	}

	conf = pConfig.New(ctx, clusterName)

	sshPubKey := conf.RequireSecret("ssh-public-key")
	if sshPubKey == pulumi.String("").ToStringOutput() {
		panic("SSH Public Key is empty!")
	}

	sshPrivKey := conf.RequireSecret("ssh-private-key")
	if sshPrivKey == pulumi.String("").ToStringOutput() {
		panic("SSH Private Key is empty!")
	}

	token := conf.RequireSecret("token")
	if token == pulumi.String("").ToStringOutput() {
		panic("Server token is empty!")
	}

	agentToken := conf.RequireSecret("agent-token")
	if agentToken == pulumi.String("").ToStringOutput() {
		panic("Agent token is empty")
	}

	dbPass := conf.RequireSecret("db-password")
	if dbPass == pulumi.String("").ToStringOutput() {
		panic("Database Password is empty!")
	}

	protectLB := conf.RequireBool("protect-lb")
	protectDB := conf.RequireBool("protect-db")

	stackConfig := &StackConfig{
		Region:           region,
		SSHPubKey:        sshPubKey,
		SSHPrivKey:       sshPrivKey,
		Token:            token,
		AgentToken:       agentToken,
		DatabasePassword: dbPass,
		ProtectLB:        protectLB,
		ProtectDB:        protectDB,
	}

	return stackConfig
}

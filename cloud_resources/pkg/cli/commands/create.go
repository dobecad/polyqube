package create

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	sanitization "polyqube/pkg/utils"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/spf13/viper"
)

var (
	ErrInvalidPlatform = errors.New("invalid platform")
)

func CreatePulumiStack(platform, clusterName, region string) error {
	ctx := context.Background()

	privateKey, publicKey, err := CreateRsaKey(4096)
	if err != nil {
		return err
	}

	s, err := auto.UpsertStackLocalSource(ctx, fmt.Sprintf("%s_%s", platform, region), ".")
	if err != nil {
		fmt.Println("Failed to create or select stack:", err)
		os.Exit(1)
	}

	err = s.AddEnvironments(ctx, "global")
	if err != nil {
		return err
	}

	err = s.SetConfig(ctx, "aws:region", auto.ConfigValue{Value: region})
	if err != nil {
		return err
	}

	pubKeyContent := string(publicKey)
	err = s.SetConfig(ctx, fmt.Sprintf("%s:ssh-public-key", clusterName), auto.ConfigValue{Value: pubKeyContent, Secret: true})
	if err != nil {
		fmt.Println("Error setting public key for stack config:", err)
		return err
	}

	privKeyContent := string(privateKey)
	err = s.SetConfig(ctx, fmt.Sprintf("%s:ssh-private-key", clusterName), auto.ConfigValue{Value: privKeyContent, Secret: true})
	if err != nil {
		fmt.Println("Error setting private key for stack config:", err)
		return err
	}

	dbPassword, err := CreateRandPassword(24)
	if err != nil {
		return err
	}

	err = s.SetConfig(ctx, fmt.Sprintf("%s:db-password", clusterName), auto.ConfigValue{Value: dbPassword, Secret: true})
	if err != nil {
		fmt.Println("Error setting database password:", err)
		return err
	}

	token, err := CreateRandPassword(24)
	if err != nil {
		return err
	}

	err = s.SetConfig(ctx, fmt.Sprintf("%s:token", clusterName), auto.ConfigValue{Value: token, Secret: true})
	if err != nil {
		fmt.Println("Error setting token:", err)
		return err
	}

	agentToken, err := CreateRandPassword(24)
	if err != nil {
		return err
	}

	err = s.SetConfig(ctx, fmt.Sprintf("%s:agent-token", clusterName), auto.ConfigValue{Value: agentToken, Secret: true})
	if err != nil {
		fmt.Println("Error setting agent-token:", err)
		return err
	}

	err = s.SetConfig(ctx, fmt.Sprintf("%s:protect-lb", clusterName), auto.ConfigValue{Value: "true"})
	if err != nil {
		fmt.Println("Error setting protect-lb:", err)
	}

	err = s.SetConfig(ctx, fmt.Sprintf("%s:protect-db", clusterName), auto.ConfigValue{Value: "true"})
	if err != nil {
		fmt.Println("Error setting protect-lb:", err)
	}

	return nil
}

func CreateRsaKey(keySize int) ([]byte, []byte, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		fmt.Println("Error generating private key:", err)
		return nil, nil, err
	}

	publicKey := &privateKey.PublicKey

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return nil, nil, err
	}

	publicKeyPEM := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}

	pemEncodedPrivateKey := pem.EncodeToMemory(privateKeyPEM)
	pemEncodedPublicKey := pem.EncodeToMemory(publicKeyPEM)

	return pemEncodedPrivateKey, pemEncodedPublicKey, nil
}

func CreateRandPassword(length int8) (string, error) {
	sanitizedLen := sanitization.Clamp(length, 24, 48)
	randomBytes := make([]byte, sanitizedLen)
	_, err := rand.Read(randomBytes)
	if err != nil {
		fmt.Println("Error creating random password:", err)
		return "", err
	}

	randomHex := hex.EncodeToString(randomBytes)
	return randomHex, nil

}

// Initializes viper to load in the correct config
// depending on the given platform
func InitViper(platform string) error {
	viper.SetConfigName("clusters")
	viper.SetConfigType("yaml")
	viper.SetDefault("regions", map[string]RegionConfig{})

	switch platform {
	case "aws":
		viper.AddConfigPath("./clusters/aws/")
	case "aws_dev":
		viper.AddConfigPath("./clusters/aws_dev/")
	case "azure":
		viper.AddConfigPath("./clusters/azure/")
	case "azure_dev":
		viper.AddConfigPath("./clusters/azure_dev/")
	case "gcp":
		viper.AddConfigPath("./clusters/gcp/")
	case "gcp_dev":
		viper.AddConfigPath("./clusters/gcp_dev/")
	default:
		return ErrInvalidPlatform
	}
	return nil

}

// Note: The fields are cluster names are intentionally left lowercase
// Viper does not care about capitalization, so your keys will be
// left as lowercase.
// https://github.com/spf13/viper?tab=readme-ov-file#does-viper-support-case-sensitive-keys
type ClusterDefinition struct {
	WorkerCount       uint8  `mapstructure:"workercount"`
	ControlPlaneCount uint8  `mapstructure:"controlplanecount"`
	TemplateId        string `mapstructure:"templateid"`
}

type RegionConfig struct {
	Clusters map[string]ClusterDefinition `mapstructure:"clusters" yaml:"clusters"`
}

type Config struct {
	Regions map[string]RegionConfig `mapstructure:"regions"`
}

func LoadClusters(platform string) (Config, error) {
	if err := InitViper(platform); err != nil {
		return Config{}, err
	}

	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Error reading config file: %s\n", err)
		return Config{}, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		fmt.Printf("Error unmashalling config: %s\n", err)
		return Config{}, err
	}

	return config, nil
}

func CreateCluster(platform, region, name string, workerNodeCount, controlPlaneNodeCount uint8) error {
	config, err := LoadClusters(platform)
	if err != nil {
		return err
	}

	// Check if region exists. If not, create it
	regionConfig, ok := config.Regions[region]
	if !ok {
		regionConfig = RegionConfig{
			Clusters: make(map[string]ClusterDefinition),
		}
		config.Regions[region] = regionConfig
	}

	newDefinition := ClusterDefinition{
		WorkerCount:       workerNodeCount,
		ControlPlaneCount: controlPlaneNodeCount,
		TemplateId:        "new-template-id",
	}
	regionConfig.Clusters[name] = newDefinition

	viper.Set("regions", config.Regions)
	if err := viper.WriteConfig(); err != nil {
		fmt.Printf("Error writing config file: %s\n", err)
		return err
	}

	if err := CreatePulumiStack(platform, name, region); err != nil {
		return err
	}

	return nil
}

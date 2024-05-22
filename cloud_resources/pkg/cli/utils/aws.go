package utils

import (
	"errors"

	"github.com/manifoldco/promptui"
)

var (
	AWSRegions = []string{"us-east-1", "us-east-2", "us-west-1", "us-west-2"}
)

var (
	ErrInvalidAWSRegion = errors.New("invalid AWS region")
)

func ValidAWSRegion(val string) error {
	if err := IsWithinStringSlice(val, AWSRegions); err != nil {
		return ErrInvalidAWSRegion
	}
	return nil
}

func AWSRegionPrompt() (string, error) {
	prompt := promptui.Select{
		Label: "AWS Region",
		Items: AWSRegions,
	}
	_, result, err := prompt.Run()
	if err != nil {
		return "", ErrInvalidAWSRegion
	}
	return result, nil
}

package utils

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"crypto/rand"
	"math/big"

	"github.com/manifoldco/promptui"
)

const allowedChars = "abcdefghijklmnopqrstuvwxyz0123456789"

var (
	Platforms = []string{"aws", "azure", "gcp", "aws_dev", "azure_dev", "gcp_dev"}
)

var (
	ErrValNotInSlice    = errors.New("val not within slice")
	ErrInvalidPlatform  = errors.New("invalid platform")
	ErrInvalidNodeCount = errors.New("invalid node count")
)

func IsWithinStringSlice(needle string, haystack []string) error {
	for _, val := range haystack {
		if needle == val {
			return nil
		}
	}
	return ErrValNotInSlice
}

func GenerateUniqueString(length int) (string, error) {
	// Determine the number of allowed characters
	numChars := big.NewInt(int64(len(allowedChars)))

	// Generate random indices to select characters from the allowed characters list
	indices := make([]int, length)
	for i := range indices {
		idx, err := rand.Int(rand.Reader, numChars)
		if err != nil {
			return "", err
		}
		indices[i] = int(idx.Int64())
	}

	// Build the string using the selected characters
	str := ""
	for _, idx := range indices {
		str += string(allowedChars[idx])
	}

	return str, nil
}

func PromptRegionBasedOnPlatform(platform string) (string, error) {
	switch platform {
	case "aws":
		return AWSRegionPrompt()
	case "aws_dev":
		return AWSRegionPrompt()
	default:
		fmt.Println("Invalid cloud platform in prompt selection")
		os.Exit(1)
	}

	return "", ErrInvalidPlatform
}

func PromptPlatform() (string, error) {
	prompt := promptui.Select{
		Label: "Cloud platform",
		Items: Platforms,
	}
	_, result, err := prompt.Run()
	if err != nil {
		os.Exit(1)
	}
	return result, nil
}

func PromptNodeCount(nodeType string) (uint8, error) {
	validate := func(input string) error {
		_, err := strconv.ParseUint(input, 10, 8)
		if err != nil {
			return ErrInvalidNodeCount
		}
		return nil
	}

	prompt := promptui.Prompt{
		Label:    fmt.Sprintf("%s node count", nodeType),
		Validate: validate,
	}
	result, err := prompt.Run()
	if err != nil {
		return 0, ErrInvalidNodeCount
	}

	r, err := strconv.ParseUint(result, 10, 8)
	if err != nil {
		return 0, err
	}
	resultCast := uint8(r)
	return resultCast, nil
}

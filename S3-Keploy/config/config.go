// Package config implements the configuration function
package config

import (
	"S3-Keploy/bucket"
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func Configuration() (awsService bucket.Basics) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("ap-south-1"))
	if err != nil {
		fmt.Printf("error: %v", err)
		return
	}

	awsService = bucket.Basics{
		S3Client: s3.NewFromConfig(cfg),
	}

	return awsService
}

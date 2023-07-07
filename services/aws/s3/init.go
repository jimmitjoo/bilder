package s3

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func NewClient(region string) (*s3.Client, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
	)
	if err != nil {
		fmt.Println("Failed to load aws configuration:", err)
		return nil, err
	}

	s3Client := s3.NewFromConfig(cfg)

	return s3Client, nil
}

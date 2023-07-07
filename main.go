package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jimmitjoo/bilder/services/adobe/photoshop"
	sss "github.com/jimmitjoo/bilder/services/aws/s3"
	"github.com/joho/godotenv"
	"log"
	"os"
)

var (
	region              = os.Getenv("AWS_REGION")
	srcKey              = "inputs/asdsad.jpeg"
	destKey             = "outputs/tjenaberra.jpg"
	autotoneURL         = "https://image.adobe.io/lrService/autoTone"
	removeBackgroundURL = "https://image.adobe.io/sensei/cutout"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("err loading: %v", err)
	}

	clientID := os.Getenv("ADOBE_CLIENT_ID")
	token := os.Getenv("ADOBE_ACCESS_TOKEN")

	bucket := os.Getenv("AWS_BUCKET")
	//s3AccessKeyID := os.Getenv("AWS_S3_ACCESS_KEY")
	//s3SecretAccessKey := os.Getenv("AWS_S3_SECRET_ACCESS_KEY")

	s3Client, err := sss.NewClient(region)
	presignClient := s3.NewPresignClient(s3Client)

	presigner := sss.Presigner{
		PresignClient: presignClient,
	}

	inputSrcSigned, _ := presigner.GetObject(bucket, srcKey, 1000)
	outputDestSigned, _ := presigner.PutObject(bucket, destKey, 1000)
	outputLinkSigned, _ := presigner.GetObject(bucket, destKey, 1000)

	// Create HTTP headers
	headers := map[string]string{
		"Authorization": "Bearer " + token,
		"x-api-key":     clientID,
		"Content-Type":  "application/json",
	}

	// Ask user what action to take
	fmt.Println("What action do you want to take?")
	fmt.Println("1. Remove background")
	fmt.Println("2. Auto tone")
	var action int
	fmt.Scanln(&action)

	switch action {
	case 1:
		// Invoke removeBackground API
		resp, err := photoshop.RemoveBackground(removeBackgroundURL, headers, inputSrcSigned, outputDestSigned)
		if err != nil {
			fmt.Println(err)
			return
		}
		// Poll status
		err = photoshop.PollStatus(resp, headers)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println(resp)

	case 2:
		// Invoke autoTone API
		resp, err := photoshop.InvokeAutotone(autotoneURL, headers, inputSrcSigned, outputDestSigned)
		if err != nil {
			fmt.Println(err)
			return
		}
		// Poll status
		err = photoshop.PollStatus(resp, headers)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println(resp)

	default:
		fmt.Println("Invalid action")
	}

	fmt.Println("Output link:", outputLinkSigned.URL)

}

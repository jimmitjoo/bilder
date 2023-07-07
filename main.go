package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jimmitjoo/bilder/actions"
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

	s3Client, err := sss.NewClient(region)
	presignClient := s3.NewPresignClient(s3Client)

	presigner := sss.Presigner{
		PresignClient: presignClient,
	}

	inputSrcSigned, _ := presigner.GetObject(srcKey, 3600)
	outputDestSigned, _ := presigner.PutObject(destKey, 3600)
	outputLinkSigned, _ := presigner.GetObject(destKey, 3600)

	// Ask user what action to take
	fmt.Println("What action do you want to take?")
	fmt.Println("1. Remove background")
	fmt.Println("2. Auto tone")
	var action int
	fmt.Scanln(&action)

	switch action {
	case 1:
		// Invoke removeBackground API
		actions.RemoveBackground(inputSrcSigned, outputDestSigned)

	case 2:
		// Invoke autoTone API
		actions.Autotone(inputSrcSigned, outputDestSigned)

	default:
		fmt.Println("Invalid action")
	}

	fmt.Println("Output link:", outputLinkSigned.URL)

}

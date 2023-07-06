package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jimmitjoo/bilder/actions"
	"log"
	"os"
)

// BucketBasics encapsulates the Amazon Simple Storage Service (Amazon S3) actions
// used in the examples.
// It contains S3Client, an Amazon S3 service client that is used to perform bucket
// and object actions.
type BucketBasics struct {
	S3Client *s3.Client
}

func UploadFile(bucketName string, objectKey string, fileName string) error {
	file, err := os.Open(fileName)
	if err != nil {
		log.Printf("Couldn't open file %v to upload. Here's why: %v\n", fileName, err)
	} else {
		defer file.Close()
		basics := BucketBasics{
			S3Client: s3.NewFromConfig(aws.Config{
				Region: "eu-north-1",
			}),
		}

		_, err := basics.S3Client.PutObject(context.TODO(), &s3.PutObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(objectKey),
			Body:   file,
		})
		if err != nil {
			log.Printf("Couldn't upload file %v to %v:%v. Here's why: %v\n",
				fileName, bucketName, objectKey, err)
		}
	}
	return err
}

func main() {

	presigner := actions.Presigner{
		PresignClient: s3.NewFromConfig(aws.Config{
			Region: "eu-north-1",
		}),
	}

	url, _ := presigner.PutObject("adobe-sign-test", "outputs/lol2.jpg", 1000)

	fmt.Println(url)
	//UploadFile("adobe-sign-test", "outputs/lol.jpg", "/Users/jimmiejohansson/go/jimmitjoo/bilder/src/bil.png")
}

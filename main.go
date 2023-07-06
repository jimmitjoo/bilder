package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/joho/godotenv"
)

var (
	srcKey      = "inputs/asdsad.jpeg"
	destKey     = "hejsan.jpg"
	autotoneURL = "https://image.adobe.io/lrService/autoTone"
)

type CutoutPayload struct {
	Input struct {
		Storage string `json:"storage"`
		Href    string `json:"href"`
	} `json:"input"`
	Output struct {
		Storage string `json:"storage"`
		Href    string `json:"href"`
		Mask    struct {
			Format string `json:"format"`
		} `json:"mask"`
	} `json:"output"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("err loading: %v", err)
	}

	clientID := os.Getenv("ADOBE_CLIENT_ID")
	token := os.Getenv("ADOBE_ACCESS_TOKEN")
	region := os.Getenv("AWS_REGION")
	bucket := os.Getenv("AWS_BUCKET")
	s3AccessKeyID := os.Getenv("AWS_S3_ACCESS_KEY")
	s3SecretAccessKey := os.Getenv("AWS_S3_SECRET_ACCESS_KEY")

	fmt.Println("Client ID:", clientID)
	fmt.Println("Token:", token)
	fmt.Println("Region:", region)
	fmt.Println("Bucket:", bucket)
	fmt.Println("SrcKey:", srcKey)
	fmt.Println("DestKey:", destKey)
	fmt.Println("S3AccessKeyID:", s3AccessKeyID)
	fmt.Println("S3SecretAccessKey:", s3SecretAccessKey)

	/*
		fmt.Println(clientID)
		fmt.Println(token)
		fmt.Println(region)
		fmt.Println(bucket)
		fmt.Println(srcKey)
		fmt.Println(destKey)
		fmt.Println(s3AccessKeyID)
		fmt.Println(s3SecretAccessKey)
	*/

	// Get pre-signed URLs
	inputSrcURL := getPresignedURL(bucket, srcKey, "getObject", 3600, region, s3AccessKeyID, s3SecretAccessKey)
	outputDestURL := getPresignedURL(bucket, destKey, "putObject", 3600, region, s3AccessKeyID, s3SecretAccessKey)

	// fmt.Println(inputSrcURL)
	fmt.Println(outputDestURL)

	// Create HTTP headers
	headers := map[string]string{
		"Authorization": "Bearer " + token,
		"x-api-key":     clientID,
		"Content-Type":  "application/json",
	}

	// Create payload
	payload := map[string]interface{}{
		"inputs": map[string]interface{}{
			"href":    inputSrcURL,
			"storage": "external",
		},
		"outputs": []map[string]interface{}{
			{
				"href":    outputDestURL,
				"storage": "external",
				"type":    "image/jpeg",
			},
		},
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Payload:", string(payloadJSON))

	// Invoke autoTone API
	resp, err := invokeAutotone(autotoneURL, headers, payload)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Poll status
	err = pollStatus(resp, headers)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(resp)
}

func getPresignedURL(bucket, key, operation string, expiresIn int64, region string, s3AccessKeyID string, s3SecretAccessKey string) string {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
		Credentials: credentials.NewStaticCredentials(
			s3AccessKeyID,
			s3SecretAccessKey,
			"",
		),
	}))

	uploader := s3manager.NewUploader(sess)

	if operation == "PutObject" {
		// Generate a pre-signed URL for a PutObject operation
		uploadUrl, _ := uploader.Upload(&s3manager.UploadInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
			ACL:    aws.String("bucket-owner-full-control"),
		})

		return uploadUrl.Location
	}

	// Generate a pre-signed URL for a GetObject operation
	req, _ := uploader.S3.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	getUrl, err := req.Presign(time.Duration(expiresIn) * time.Second)
	if err != nil {
		fmt.Println(err)
		return ""
	}

	return getUrl
}

func invokeAutotone(url string, headers map[string]string, payload map[string]interface{}) (*http.Response, error) {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	return client.Do(req)
}

func getAutotoneStatus(url string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	return client.Do(req)
}

func pollStatus(resp *http.Response, headers map[string]string) error {
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fmt.Println("Response Body:", string(body))

	var tmp map[string]interface{}
	err = json.Unmarshal(body, &tmp)
	if err != nil {
		return err
	}

	if tmp["code"] != nil {
		return fmt.Errorf("API Error: %v", tmp)
	}

	selfLink, ok := tmp["_links"].(map[string]interface{})["self"].(map[string]interface{})
	if !ok || selfLink == nil {
		return fmt.Errorf("self link not found")
	}

	url, ok := selfLink["href"].(string)
	if !ok || url == "" {
		return fmt.Errorf("invalid self link")
	}

	fmt.Println("Self Link:", url)

	for {
		respStatus, err := getAutotoneStatus(url, headers)
		if err != nil {
			return err
		}

		bodyStatus, err := io.ReadAll(respStatus.Body)
		if err != nil {
			return err
		}

		fmt.Println("Status Response Body:", string(bodyStatus))

		var tmpStatus map[string]interface{}
		err = json.Unmarshal(bodyStatus, &tmpStatus)
		if err != nil {
			return err
		}

		if tmpStatus["code"] != nil {
			return fmt.Errorf("API Error: %v", tmpStatus)
		}

		outputs, ok := tmpStatus["outputs"].([]interface{})
		if !ok || len(outputs) == 0 {
			return fmt.Errorf("invalid outputs data")
		}

		output, ok := outputs[0].(map[string]interface{})
		if !ok || output == nil {
			return fmt.Errorf("invalid output data")
		}

		fmt.Println("Output:", output)

		status, ok := output["status"].(string)
		if !ok || status == "" {
			return fmt.Errorf("invalid status data")
		}

		fmt.Println("Status:", status)

		if status == "failed" {
			if _, errorExists := output["error"]; errorExists {
				errorValue, ok := output["error"].(string)
				if !ok {
					return fmt.Errorf("invalid error data")
				}
				fmt.Println("Error:", errorValue)
			}
			return nil
		}

		if status == "succeeded" {
			return nil
		}

		time.Sleep(1 * time.Second)
	}

}

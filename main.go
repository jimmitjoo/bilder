package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jimmitjoo/bilder/actions"
	"github.com/joho/godotenv"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type AdobePayload struct {
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

type RemoveBackgroundPayload struct {
	Input struct {
		Href    string `json:"href"`
		Storage string `json:"storage"`
	} `json:"input"`
	Output struct {
		Href    string `json:"href"`
		Storage string `json:"storage"`
		Mask    struct {
			Format string `json:"format"`
		} `json:"mask"`
	} `json:"output"`
}

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

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
	)
	if err != nil {
		fmt.Println("Failed to load AWS configuration:", err)
		return
	}

	s3Client := s3.NewFromConfig(cfg)
	presignClient := s3.NewPresignClient(s3Client)

	presigner := actions.Presigner{
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
		resp, err := removeBackground(removeBackgroundURL, headers, inputSrcSigned, outputDestSigned)
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

	case 2:
		// Invoke autoTone API
		resp, err := invokeAutotone(autotoneURL, headers, inputSrcSigned, outputDestSigned)
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

	default:
		fmt.Println("Invalid action")
	}

	fmt.Println("Output link:", outputLinkSigned.URL)

}

func removeBackground(url string, headers map[string]string, inputSrcSigned *v4.PresignedHTTPRequest, outputDestSigned *v4.PresignedHTTPRequest) (*http.Response, error) {

	// Create payload
	payload := RemoveBackgroundPayload{
		Input: struct {
			Href    string `json:"href"`
			Storage string `json:"storage"`
		}{
			Href:    inputSrcSigned.URL,
			Storage: "external",
		},
		Output: struct {
			Href    string `json:"href"`
			Storage string `json:"storage"`
			Mask    struct {
				Format string `json:"format"`
			} `json:"mask"`
		}{
			Href:    outputDestSigned.URL,
			Storage: "external",
			Mask: struct {
				Format string `json:"format"`
			}{
				Format: "soft",
			},
		},
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		fmt.Println(err)
		return nil, nil
	}

	fmt.Println("Payload:", string(payloadJSON))

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

func invokeAutotone(url string, headers map[string]string, inputSrcSigned *v4.PresignedHTTPRequest, outputDestSigned *v4.PresignedHTTPRequest) (*http.Response, error) {
	// Create payload based on AdobePayload struct
	payload := map[string]interface{}{
		"inputs": map[string]interface{}{
			"href":    inputSrcSigned.URL,
			"storage": "external",
		},
		"outputs": []map[string]interface{}{
			{
				"href":    outputDestSigned.URL,
				"storage": "external",
				"type":    "image/jpeg",
			},
		},
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		fmt.Println(err)
		return nil, nil
	}

	fmt.Println("Payload:", string(payloadJSON))

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

	fmt.Println("Headers:", resp.Header)
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

		var outputs []interface{}
		var output map[string]interface{}

		if tmpOutputs, ok := tmpStatus["outputs"].([]interface{}); ok && len(tmpOutputs) > 0 {
			// Use tmpOutputs as the value for outputs
			outputs = tmpOutputs
		} else if tmpOutput, ok := tmpStatus["output"].(map[string]interface{}); ok {
			// Use tmpOutput as the value for output
			output = tmpOutput
		} else {
			return fmt.Errorf("invalid outputs data")
		}

		if len(outputs) > 0 {
			// Use outputs[0] as the value for output
			if tmpOutput, ok := outputs[0].(map[string]interface{}); ok {
				output = tmpOutput
			} else {
				return fmt.Errorf("invalid output data")
			}
		}

		fmt.Println("Output:", output)

		if tmpStatus["status"] != nil {
			status, ok := tmpStatus["status"].(string)
			if !ok || status == "" {
				// If status is not found, check for error
				if _, errorExists := tmpStatus["error"]; errorExists {
					errorValue, ok := tmpStatus["error"].(string)
					if !ok {
						return fmt.Errorf("invalid error data")
					}
					fmt.Println("Error:", errorValue)
				}

				return fmt.Errorf("invalid status data")
			}

			fmt.Println("Status:", status)

			if status == "failed" {
				if _, errorExists := tmpStatus["error"]; errorExists {
					errorValue, ok := tmpStatus["error"].(string)
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
		} else {

			status, ok := output["status"].(string)
			if !ok || status == "" {
				// If status is not found, check for error
				if _, errorExists := output["error"]; errorExists {
					errorValue, ok := output["error"].(string)
					if !ok {
						return fmt.Errorf("invalid error data")
					}
					fmt.Println("Error:", errorValue)
				}

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

		}

		time.Sleep(1 * time.Second)
	}

}

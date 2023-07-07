package photoshop

import (
	"bytes"
	"encoding/json"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"net/http"
	"os"
)

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

var removeBackgroundURL = "https://image.adobe.io/sensei/cutout"

func RemoveBackground(inputSrcSigned *v4.PresignedHTTPRequest, outputDestSigned *v4.PresignedHTTPRequest) (*http.Response, error) {

	// Create HTTP headers
	headers := map[string]string{
		"Authorization": "Bearer " + os.Getenv("ADOBE_ACCESS_TOKEN"),
		"x-api-key":     os.Getenv("ADOBE_CLIENT_ID"),
		"Content-Type":  "application/json",
	}

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

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", removeBackgroundURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	return client.Do(req)
}

package photoshop

import (
	"bytes"
	"encoding/json"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"net/http"
	"os"
)

type AutotonePayload struct {
	Inputs struct {
		Href    string `json:"href"`
		Storage string `json:"storage"`
	} `json:"inputs"`
	Outputs []struct {
		Href    string `json:"href"`
		Storage string `json:"storage"`
		Type    string `json:"type"`
	} `json:"outputs"`
}

var autotoneURL = "https://image.adobe.io/lrService/autoTone"

func Autotone(inputSrcSigned *v4.PresignedHTTPRequest, outputDestSigned *v4.PresignedHTTPRequest) (*http.Response, error) {

	// Create HTTP headers
	headers := map[string]string{
		"Authorization": "Bearer " + os.Getenv("ADOBE_ACCESS_TOKEN"),
		"x-api-key":     os.Getenv("ADOBE_CLIENT_ID"),
		"Content-Type":  "application/json",
	}

	// Create payload based on AutotonePayload struct
	payload := AutotonePayload{
		Inputs: struct {
			Href    string `json:"href"`
			Storage string `json:"storage"`
		}{
			Href:    inputSrcSigned.URL,
			Storage: "external",
		},
		Outputs: []struct {
			Href    string `json:"href"`
			Storage string `json:"storage"`
			Type    string `json:"type"`
		}{
			{
				Href:    outputDestSigned.URL,
				Storage: "external",
				Type:    "image/jpeg",
			},
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", autotoneURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	return client.Do(req)
}

package photoshop

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func fetchStatus(url string, headers map[string]string) (*http.Response, error) {
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

func PollStatus(resp *http.Response) error {

	// Create HTTP headers
	headers := map[string]string{
		"Authorization": "Bearer " + os.Getenv("ADOBE_ACCESS_TOKEN"),
		"x-api-key":     os.Getenv("ADOBE_CLIENT_ID"),
		"Content-Type":  "application/json",
	}

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

	for {
		respStatus, err := fetchStatus(url, headers)
		if err != nil {
			return err
		}

		bodyStatus, err := io.ReadAll(respStatus.Body)
		if err != nil {
			return err
		}

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
			message := tmpStatus["message"].(string)
			return fmt.Errorf(message)
		}

		if len(outputs) > 0 {
			// Use outputs[0] as the value for output
			if tmpOutput, ok := outputs[0].(map[string]interface{}); ok {
				output = tmpOutput
			} else {
				return fmt.Errorf("invalid output data")
			}
		}

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

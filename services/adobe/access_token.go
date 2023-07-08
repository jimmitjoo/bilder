package adobe

import (
	"encoding/json"
	"fmt"
	"github.com/kataras/iris/v12"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type AccessToken struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

func CheckAccessToken(ctx iris.Context) {

	accessTokenCookie := ctx.GetCookie("access_token")

	// If access token cookie does not exist, get a new one
	if accessTokenCookie == "" {
		fmt.Println("get new access token")
		accessToken, err := GenerateAccessToken()
		if err != nil {
			fmt.Println(err)
			return
		}
		os.Setenv("ADOBE_ACCESS_TOKEN", accessToken)

		ctx.SetCookieKV("access_token", accessToken, iris.CookieExpires(time.Hour*24))
		return
	}

	var cookie AccessToken
	err := json.Unmarshal([]byte(accessTokenCookie), &cookie)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Check if access token is expired
	expiration := time.Duration(cookie.ExpiresIn) * time.Second
	expiresAt := time.Now().Add(expiration)

	// If access token is expired, get a new one
	if time.Now().After(expiresAt) {
		fmt.Println("get new access token")
		accessToken, err := GenerateAccessToken()
		if err != nil {
			fmt.Println(err)
			return
		}
		os.Setenv("ADOBE_ACCESS_TOKEN", accessToken)

		ctx.SetCookieKV("access_token", accessToken, iris.CookieExpires(time.Hour*24))
		return
	}

	// If access token is not expired, use the existing one
	fmt.Println("use existing access token")
	os.Setenv("ADOBE_ACCESS_TOKEN", cookie.AccessToken)

}

func GenerateAccessToken() (string, error) {
	// Set the endpoint URL
	tokenUrl := "https://ims-na1.adobelogin.com/ims/token/v3"

	// Set the form data
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", os.Getenv("ADOBE_CLIENT_ID"))
	data.Set("client_secret", os.Getenv("ADOBE_CLIENT_SECRET"))
	data.Set("scope", "openid,AdobeID,read_organizations")

	// Create a new HTTP client
	client := &http.Client{}

	// Create a new POST request
	req, err := http.NewRequest("POST", tokenUrl, strings.NewReader(data.Encode()))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return "", err
	}

	// Set the request headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return "", err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return "", err
	}

	// Return the response body
	return string(body), nil
}

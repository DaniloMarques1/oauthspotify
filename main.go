package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type MakeTokenRequest struct {
	Url           string
	Grant_type    string
	Code          string
	Redirect_uri  string
	Client_id     string // header
	Client_secret string // header
}

type TokenResponse struct {
	Access_token  string `json:"access_token"`
	Token_type    string `json:"token_type"`
	Scope         string `json:"scope"`
	Expires_in    int64  `json:"expires_in"`
	Refresh_token string `json:"refresh_token"`
}

type MakeRefreshTokenRequest struct {
	Grant_type    string `json:"grant_type"`
	Refresh_token string `json:"refresh_token"`
	Client_id     string // header
	Client_secret string // header
}

// returns a MakeTokenRequest object
func NewMakeTokenRequest(url, grant_type, code, redirect_uri string) *MakeTokenRequest {
	client_id := os.Getenv("client_id")
	client_secret := os.Getenv("client_secret")
	makeTokenRequest := &MakeTokenRequest{
		url,
		grant_type,
		code,
		redirect_uri,
		client_id,
		client_secret,
	}

	return makeTokenRequest
}

func NewMakeRefreshTokenRequest(grant_type, refresh_token string) *MakeRefreshTokenRequest {
	client_id := os.Getenv("client_id")
	client_secret := os.Getenv("client_secret")
	//os.Setenv("jwt_secret", "ksldksldksd") // TODO what is this
	refreshTokenRequest := MakeRefreshTokenRequest{
		grant_type,
		refresh_token,
		client_id,
		client_secret,
	}
	return &refreshTokenRequest
}

func (mrtr *MakeRefreshTokenRequest) Body() *strings.Reader {
	// query parameters
	body := url.Values{}
	body.Add("grant_type", mrtr.Grant_type)
	body.Add("refresh_token", mrtr.Refresh_token)
	body_reader := strings.NewReader(body.Encode())

	return body_reader
}

func (mtr *MakeTokenRequest) Body() *strings.Reader {
	body := url.Values{}
	body.Add("grant_type", mtr.Grant_type)
	body.Add("code", mtr.Code)
	body.Add("redirect_uri", mtr.Redirect_uri)
	body_reader := strings.NewReader(body.Encode())

	return body_reader
}

func (tr *TokenResponse) SaveToken(filename string) {
	b_token, err := json.Marshal(tr)
	if err != nil {
		log.Fatal("Error parsing json to save")
	}
	file, err := os.Create(filename)
	if err != nil {
		log.Fatal("Error creating token file")
	}
	defer file.Close()
	_, err = file.WriteString(string(b_token))
	if err != nil {
		log.Fatal("Error saving token")
	}
}

func getTokenFromFile(filename string) (*TokenResponse, error) {
	b, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var tokenResponse TokenResponse
	if err := json.Unmarshal(b, &tokenResponse); err != nil {
		return nil, err
	}

	return &tokenResponse, nil
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("error reading enviroment variables")
	}
	http.HandleFunc("/", Index)
	http.HandleFunc("/redirect", RedirectUri)

	http.ListenAndServe(":8080", nil)
}

// checks if there is a token already, if not opens up a browser
// to make the authorization code request to be exchanged for a
// access token in the redirect endpoint
func Index(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not suported", http.StatusMethodNotAllowed)
		return
	}
	tokenResponse, err := getTokenFromFile(".token")
	if err != nil {
		// this will open the authorization server so you can provide your credentials in order to
		// be able to access the resources you want.
		// user-read-recently-played is the scope for this operation
		// foobar is the state, it can be anything
		// code is the response_type, which means we want the response_code
		codeRequestUrl := buildGetCodeRequestUrl("https://accounts.spotify.com/authorize", "user-read-recently-played", "http://127.0.0.1:8080/redirect", "foobar", "code")
		exec.Command("firefox", codeRequestUrl).Start()
	} else {
		now := time.Now().Unix()
		if isTokenExpired(now, tokenResponse.Expires_in) {
			refreshTokenRequest := NewMakeRefreshTokenRequest("refresh_token", tokenResponse.Refresh_token)
			refreshToken(refreshTokenRequest)
		} else {
			requestData(tokenResponse)
		}
	}
}

func buildGetCodeRequestUrl(baseUrl, scope, redirectUri, state, responseType string) string {
	clientId := os.Getenv("client_id")
	return fmt.Sprintf("%v?client_id=%v&redirect_uri=%v&response_type=%v&state=%v&scope=%v",
		baseUrl, clientId, redirectUri, responseType, state, scope)
}

func isTokenExpired(nowMillis, tokenMillis int64) bool {
	return nowMillis > tokenMillis
}

// after the user accept the usage of his data it will get the
// authorization code and then exchange it for a access token
func RedirectUri(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not suported", http.StatusMethodNotAllowed)
		return
	}
	client := &http.Client{}
	authorization_code := r.FormValue("code")
	makeTokenRequest := NewMakeTokenRequest("https://accounts.spotify.com/api/token", "authorization_code", authorization_code, "http://127.0.0.1:8080/redirect")
	req, err := http.NewRequest(http.MethodPost, makeTokenRequest.Url, makeTokenRequest.Body())
	if err != nil {
		log.Fatalf("Error creating the request %v", err)
	}
	header := getAuthorizationHeader(makeTokenRequest.Client_id, makeTokenRequest.Client_secret)
	req.Header.Set("Authorization", "Basic "+header)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := client.Do(req)
	if err != nil {
		log.Fatal("Error performing request")
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal("Error reading body")
	}
	var tokenResponse TokenResponse
	err = json.Unmarshal(body, &tokenResponse)
	if err != nil {
		log.Fatal("Error getting the token struct")
	}
	fmt.Fprintf(w, "Token obtained. Check your terminal.")
	now := time.Now().Unix()
	tokenResponse.Expires_in += now
	tokenResponse.SaveToken(".token")
	requestData(&tokenResponse)
}

// makes use of the access token to request the recently played tracks
func requestData(tokenResponse *TokenResponse) {
	client := http.Client{}
	spotify_url := "https://api.spotify.com/v1/me/player/recently-played"
	req, err := http.NewRequest(http.MethodGet, spotify_url, nil)
	if err != nil {
		log.Fatalf("Error creating the request %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+tokenResponse.Access_token)
	response, err := client.Do(req)
	if err != nil || response.StatusCode != 200 {
		log.Fatalf("Error making request %v %v", response.StatusCode, err)
	}
	defer response.Body.Close()
	bytes, err := io.ReadAll(response.Body)
	showTracks(bytes)
}

// request a new token using the refresh token
func refreshToken(refreshNewToken *MakeRefreshTokenRequest) {
	client := http.Client{}
	baseUrl := "https://accounts.spotify.com/api/token"
	req, err := http.NewRequest(http.MethodPost, baseUrl, refreshNewToken.Body())
	if err != nil {
		log.Fatalf("Error creating request %v\n", err)
	}
	header := getAuthorizationHeader(refreshNewToken.Client_id, refreshNewToken.Client_secret)
	req.Header.Set("Authorization", "Basic "+header)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := client.Do(req)
	if err != nil || response.StatusCode != 200 {
		log.Fatalf("error making request %v %v\n", response.StatusCode, err)
	}
	defer response.Body.Close()
	b, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatalf("Error parsing response %v\n", err)
	}
	var tokenResponse TokenResponse
	json.Unmarshal(b, &tokenResponse)
	tokenResponse.Refresh_token = refreshNewToken.Refresh_token
	now := time.Now().Unix()
	tokenResponse.Expires_in += now
	tokenResponse.SaveToken(".token")
	requestData(&tokenResponse)
}

// returns the base64 encoded client_id:client_secret
func getAuthorizationHeader(client_id, client_secret string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(client_id + ":" + client_secret))
	return header
}

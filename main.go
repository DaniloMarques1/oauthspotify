package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/joho/godotenv"
)

//var client_secret string

type MakeCodeRequest struct {
	Url           string
	Scopes        string
	Redirect_uri  string
	State         string
	Response_type string
	Client_id     string
}

type MakeTokenRequest struct {
	Url           string
	Grant_type    string
	Code          string
	Redirect_uri  string
	Client_id     string // header
	Client_secret string // header
}

type TokenResponse struct {
	access_token  string
	token_type    string
	scope         string
	expires_in    int64
	refresh_token string
}

// returns a MakeCodeRequest object
func NewMakeCodeRequest(url, scopes, redirect_uri, state, response_type string) *MakeCodeRequest {
	client_id := os.Getenv("client_id")
	makeCodeRequest := &MakeCodeRequest{
		url,
		scopes,
		redirect_uri,
		state,
		response_type,
		client_id,
	}

	return makeCodeRequest
}

// return the url that will be used to make the request
// for the authorization code
func (mcr *MakeCodeRequest) RequestUrl() string {
	return fmt.Sprintf("%v?response_type=code&client_id=%v&redirect_uri=%v&state=%v&scope=%v",
		mcr.Url, mcr.Client_id, url.QueryEscape(mcr.Redirect_uri), mcr.State, url.QueryEscape(mcr.Scopes))
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

func (mtr *MakeTokenRequest) Body() *strings.Reader {
	body := url.Values{}
	body.Add("grant_type", mtr.Grant_type)
	body.Add("code", mtr.Code)
	body.Add("redirect_uri", mtr.Redirect_uri)
	body_reader := strings.NewReader(body.Encode())

	return body_reader
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("error reading enviroment variables")
	}
	http.HandleFunc("/", Index)
	http.HandleFunc("/redirect", RedirectUri)

	http.ListenAndServe(":8080", nil)
}

func Index(w http.ResponseWriter, r *http.Request) {
	mcr := NewMakeCodeRequest("https://accounts.spotify.com/authorize", "user-read-recently-played",
		"http://127.0.0.1:8080/redirect", "foobar", "code")
	exec.Command("firefox", mcr.RequestUrl()).Start()
	fmt.Fprintf(w, "Ola")
}

func RedirectUri(w http.ResponseWriter, r *http.Request) {
	client := &http.Client{}
	authorization_code := r.FormValue("code")
	//fmt.Println(authorization_code)
	fmt.Fprintf(w, "Opa\n")
	makeTokenRequest := NewMakeTokenRequest("https://accounts.spotify.com/api/token", "authorization_code", authorization_code, "http://127.0.0.1:8080/redirect")
	req, err := http.NewRequest(http.MethodPost, makeTokenRequest.Url, makeTokenRequest.Body())
	if err != nil {
		log.Fatalf("Error creating the request %v", err)
	}
	header := base64.RawURLEncoding.EncodeToString([]byte(makeTokenRequest.Client_id + ":" + makeTokenRequest.Client_secret))
	fmt.Println(header)
	req.Header.Set("Authorization", "Basic "+header)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := client.Do(req)
	if err != nil {
		log.Fatal("Error performing request")
	}
	defer response.Body.Close()
	fmt.Fprint(w, "Performed request")
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal("Error reading body")
	}
	fmt.Println(string(body))
}

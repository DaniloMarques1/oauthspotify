package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

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

func NewMakeRefreshTokenRequest(grant_type, refresh_token string) *MakeRefreshTokenRequest {
	client_id := os.Getenv("client_id")
	client_secret := os.Getenv("client_secret")
	refreshTokenRequest := MakeRefreshTokenRequest{
		grant_type,
		refresh_token,
		client_id,
		client_secret,
	}
	return &refreshTokenRequest
}

func (mrtr *MakeRefreshTokenRequest) Body() *strings.Reader {
	body := url.Values{}
	body.Add("grant_type", mrtr.Grant_type)
	body.Add("refresh_token", mrtr.Refresh_token)
	body_reader := strings.NewReader(body.Encode())

	return body_reader
}

// recently played tracks response
type Response struct {
	Items []Item `json:"items"`
}

type Item struct {
	Track     Track  `json:"track"`
	Played_at string `json:"played_at"`
}

type Track struct {
	Album Album  `json:"album"`
	Name  string `json:"name"`
}

type Album struct {
	Artists []Artist `json:"artists"`
	Name    string   `json:"name"`
}

type Artist struct {
	Name string `json:"name"`
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

func GetTokenFromFile(filename string) (*TokenResponse, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var tokenResponse TokenResponse
	json.Unmarshal(b, &tokenResponse)

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
	tokenResponse, err := GetTokenFromFile(".token")
	if err != nil {
		mcr := NewMakeCodeRequest("https://accounts.spotify.com/authorize", "user-read-recently-played",
			"http://127.0.0.1:8080/redirect", "foobar", "code")
		exec.Command("firefox", mcr.RequestUrl()).Start()
	} else {
		now := time.Now().Unix()
		if now < tokenResponse.Expires_in {
			// TODO need a new token
			refreshTokenRequest := NewMakeRefreshTokenRequest("refresh_token", tokenResponse.Refresh_token)
			RequestNewToken(refreshTokenRequest)
		} else {
			MakingDataRequest(tokenResponse)
		}
	}
}

// after the user accept the usage of his data it will get the
// authorization code and then exchange it for a access token
func RedirectUri(w http.ResponseWriter, r *http.Request) {
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
	body, err := ioutil.ReadAll(response.Body)
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
	MakingDataRequest(&tokenResponse)
}

// makes use of the access token to request the recently played tracks
func MakingDataRequest(tokenResponse *TokenResponse) {
	client := http.Client{}
	spotify_url := "https://api.spotify.com/v1/me/player/recently-played"
	req, err := http.NewRequest(http.MethodGet, spotify_url, nil)
	if err != nil {
		log.Fatalf("Error creating the request %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+tokenResponse.Access_token)
	response, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error making request %v", err)
	}
	if response.StatusCode != 200 {
		//TODO fix
		fmt.Println("calling new token")
		refreshTokenRequest := NewMakeRefreshTokenRequest("refresh_token", tokenResponse.Refresh_token)
		RequestNewToken(refreshTokenRequest)
	}
	body, err := ioutil.ReadAll(response.Body)
	var data Response
	err = json.Unmarshal(body, &data)
	if err != nil {
		log.Fatalf("error parsing json %v", err)
	}

	showTracks(data)
}

// request a new token using the refresh token
func RequestNewToken(refreshNewToken *MakeRefreshTokenRequest) {
	client := http.Client{}
	url_refresh_token := "https://accounts.spotify.com/api/token"
	req, err := http.NewRequest(http.MethodPost, url_refresh_token, refreshNewToken.Body())
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
	b, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatalf("Error parsing response %v\n", err)
	}
	var tokenResponse TokenResponse
	json.Unmarshal(b, &tokenResponse)
	tokenResponse.Refresh_token = refreshNewToken.Refresh_token // TODO can i do this?
	tokenResponse.SaveToken(".token")
	MakingDataRequest(&tokenResponse)
}

// returns the base64 encoded client_id:client_secret
func getAuthorizationHeader(client_id, client_secret string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(client_id + ":" + client_secret))
	return header
}

// loops over the response from spotify and shows
// recently played tracks
func showTracks(data Response) {
	itens := data.Items
	var size int
	if len(itens) < 10 {
		size = len(itens)
	} else {
		size = 10
	}
	for i := 0; i < size; i++ {
		item := itens[i]
		dt, err := time.Parse("2006-01-02T15:04:05.000Z", item.Played_at)
		loc, err := time.LoadLocation("America/Recife")
		if err != nil {
			log.Fatalf("Error loading location %v", err)
		}
		fmt.Printf("Music name: %s\n", item.Track.Name)
		fmt.Printf("Album name: %s\n", item.Track.Album.Name)
		if err == nil {
			dtStr := dt.In(loc).Format(time.UnixDate)
			fmt.Printf("Played at: %v\n", dtStr)
		}
		fmt.Println("Artits (just the first 2):")
		artits := item.Track.Album.Artists
		var artistsSize int
		if len(artits) < 2 {
			artistsSize = len(artits)
		} else {
			artistsSize = 2
		}
		for j := 0; j < artistsSize; j++ {
			fmt.Printf("   %d- %v\n", j+1, artits[j].Name)
		}
		fmt.Println("==================")
		fmt.Println()
	}
}

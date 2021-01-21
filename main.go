package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"

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

// TODO sturct to be used to request token
type MakeTokenRequest struct {
}

// populates the object with the given values
func (mcr *MakeCodeRequest) New(url, scopes, redirect_uri, state, response_type string) {
	mcr.Client_id = os.Getenv("client_id")
	mcr.Url = url
	mcr.Scopes = scopes
	mcr.Redirect_uri = redirect_uri
	mcr.State = state
	mcr.Response_type = response_type
}

// return the url that will be used to make the request
// for the authorization code
func (mcr *MakeCodeRequest) RequestUrl() string {
	return fmt.Sprintf("%v?response_type=code&client_id=%v&redirect_uri=%v&state=%v&scope=%v",
		mcr.Url, mcr.Client_id, url.QueryEscape(mcr.Redirect_uri), mcr.State, url.QueryEscape(mcr.Scopes))
}

var client_secret string // TODO remove
func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("error reading enviroment variables")
	}
	client_secret = os.Getenv("client_secret")

	http.HandleFunc("/", Index)
	http.HandleFunc("/redirect", RedirectUri)

	http.ListenAndServe(":8080", nil)
}

func Index(w http.ResponseWriter, r *http.Request) {
	var mcr MakeCodeRequest
	mcr.New("https://accounts.spotify.com/authorize", "user-read-recently-played", "http://127.0.0.1:8080/redirect", "foobar", "code")
	exec.Command("firefox", mcr.RequestUrl()).Start()
	fmt.Fprintf(w, "Ola")
}

func RedirectUri(w http.ResponseWriter, r *http.Request) {
	// it will come with the authorization code
	fmt.Println("Opa")
	//code := r.Form["code"]
	authorization_code := r.FormValue("code")
	fmt.Println(authorization_code)
	fmt.Fprintf(w, "Opa\n")
}

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

const (
	url_code      = "url"
	url_token     = "url"
	scopes        = "scopes"
	redirect_uri  = "redirect_uri"
	state         = "state"
	response_type = "response_type"
	grant_type    = "grand_type"
	code          = "code"
)

func Test_NewMakeCodeRequest(t *testing.T) {
	makeCodeRequest := getMakeCodeRequest(t)
	if makeCodeRequest.Client_id != os.Getenv("client_id") || makeCodeRequest.Redirect_uri != redirect_uri ||
		makeCodeRequest.Response_type != response_type || makeCodeRequest.Scopes != scopes ||
		makeCodeRequest.State != state || makeCodeRequest.Url != url_code {
		t.Fatal("Values differ in the NewMakeCodeRequest")
	}
}

func Test_NewMakeTokenRequest(t *testing.T) {
	makeTokenRequest := getMakeTokenRequest(t)
	if makeTokenRequest.Client_id != os.Getenv("client_id") || makeTokenRequest.Client_secret != os.Getenv("client_secret") ||
		makeTokenRequest.Code != code || makeTokenRequest.Grant_type != grant_type || makeTokenRequest.Url != url_token ||
		makeTokenRequest.Redirect_uri != redirect_uri {
		t.Fatal("Values differ in the NewMakeTokenRequest")
	}
}

func Test_RequestCodeUrl(t *testing.T) {
	makeCodeRequest := getMakeCodeRequest(t)
	expect := fmt.Sprintf("url?response_type=code&client_id=%v&redirect_uri=redirect_uri&state=state&scope=scopes", makeCodeRequest.Client_id)
	if makeCodeRequest.RequestUrl() != expect {
		t.Fatal("Wrong generated url")
	}
}

func Test_RequestTokenBody(t *testing.T) {
	makeTokenRequest := getMakeTokenRequest(t)
	b, _ := ioutil.ReadAll(makeTokenRequest.Body())
	expect := "code=code&grant_type=grand_type&redirect_uri=redirect_uri"
	if string(b) != expect {
		t.Fatal("Body to request token is incorrect")
	}
}

func loadEnv(t *testing.T) {
	if err := godotenv.Load(); err != nil {
		t.Fatal("Error loading env vars")
	}
}

func getMakeCodeRequest(t *testing.T) *MakeCodeRequest {
	loadEnv(t)
	makeCodeRequest := NewMakeCodeRequest(url_code, scopes, redirect_uri, state, response_type)
	return makeCodeRequest
}

func getMakeTokenRequest(t *testing.T) *MakeTokenRequest {
	loadEnv(t)
	makeTokenRequest := NewMakeTokenRequest(url_token, grant_type, code, redirect_uri)
	return makeTokenRequest
}

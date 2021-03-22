// Copyright 2021 MIMIRO AS
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package login

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/viper"
	bolt "go.etcd.io/bbolt"
)

type Response struct {
	Message string `json:"message"`
}

type tokenRequest struct {
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Audience     string `json:"audience"`
	GrantType    string `json:"grant_type"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
	ExpiresIn    int64  `json:"expires_in,omitempty"`
	TokenType    string `json:"token_type"`
}

func ResolveCredentials() (string, string, error) {
	alias := viper.GetString("activelogin")
	if alias != "" {
		payload, err := getLoginAlias(alias)
		if err != nil {
			return "", "", err
		}

		if payload.ClientId == "" {
			return payload.Server, payload.Token, nil
		}

		token, err := exchangeToken(payload)
		if err != nil {
			return "", "", err
		}
		return payload.Server, token.AccessToken, nil
	} else {
		server := viper.GetString("server")
		token := viper.GetString("token")

		return server, token, nil
	}
}

func exchangeToken(config *payload) (*tokenResponse, error) {
	request := tokenRequest{
		ClientId:     config.ClientId,
		ClientSecret: config.ClientSecret,
		Audience:     config.Audience,
		GrantType:    "app_credentials",
	}

	if request.Audience == "" {
		request.Audience = config.Server
	}

	content, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", config.Authorizer, bytes.NewBuffer(content))
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusOK {
		token := &tokenResponse{}
		err = json.Unmarshal(bodyBytes, token)
		if err != nil {
			return nil, err
		}
		return token, nil
	} else if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("the combination of id and secred did not match")
	} else {
		message := &Response{}
		err = json.Unmarshal(bodyBytes, message)
		utils.HandleError(err)
		return nil, errors.New("Got http status " + resp.Status + ": " + message.Message)
	}
}

func getLoginAlias(alias string) (*payload, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	db, err := bolt.Open(home+"/.mim/conf.db", 0666, &bolt.Options{ReadOnly: true})
	defer db.Close()

	data := &payload{}
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("logins"))
		res := b.Get([]byte(alias))
		if res == nil {
			return errors.New("alias not found")
		}
		err := json.Unmarshal(res, data)
		if err != nil {
			data = nil
			return err
		}
		return nil
	})

	return data, err
}

// attemptLogin takes the token and server configuration it is given, and tries to call the /jobs endpoint on the server
// If it gets a 200 OK, it assumes login is fine, if not, it returns an error
func AttemptLogin(server string, token string) error {
	if server == "" {
		return errors.New("server is missing")
	}

	// we check to see if we can get the jobs list
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/datasets", server), nil)
	utils.HandleError(err)

	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}
	pterm.Println()
	introSpinner := pterm.DefaultSpinner.WithRemoveWhenDone(true).Start("Login in to: " + server)

	resp, err := http.DefaultClient.Do(req)
	utils.HandleError(err)

	time.Sleep(500 * time.Millisecond) // add some time to let the user feel like he is doing something

	introSpinner.Stop()

	if resp.StatusCode != http.StatusOK {
		// should have a response message
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		utils.HandleError(err)
		message := &Response{}
		err = json.Unmarshal(bodyBytes, message)
		return errors.New(message.Message)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	pterm.Success.Println("Logged in to " + server)

	return nil
}

func UpdateConfig(alias string) {
	viper.Set("activeLogin", alias)

	err := viper.SafeWriteConfig()
	if err != nil { // if file exist, we try again
		err = viper.WriteConfig()
		utils.HandleError(err)
	}

	pterm.Success.Println("Updated config file")
}

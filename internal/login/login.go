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
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mimiro-io/datahub-cli/internal/config"
	"github.com/mimiro-io/datahub-cli/internal/web"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/viper"
)

type Response struct {
	Message string `json:"message"`
}

// ResolveCredentials is deprecated, and you should realy use the web.ResolveCredentials() instead.
func ResolveCredentials() (string, string, error) {
	alias := viper.GetString("activelogin")
	if alias != "" {

		payload, err := getLoginAlias(alias)
		if err != nil {
			return "", "", err
		}
		tkn, err := web.ResolveCredentials()
		if err != nil {
			return "", "", err
		}
		return payload.Server, tkn.AccessToken, nil
	} else {
		server := viper.GetString("server")
		token := viper.GetString("token")

		return server, token, nil
	}
}

func getLoginAlias(alias string) (*config.Config, error) {
	data := &config.Config{}
	if err := config.Load(alias, data); err != nil {
		return nil, err
	}
	return data, nil
}

// AttemptLogin takes the token and server configuration it is given, and tries to call the /jobs endpoint on the server
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
	introSpinner, err := pterm.DefaultSpinner.WithRemoveWhenDone(true).Start("Login in to: " + server)
	utils.HandleError(err)

	resp, err := http.DefaultClient.Do(req)
	utils.HandleError(err)

	time.Sleep(500 * time.Millisecond) // add some time to let the user feel like he is doing something

	err = introSpinner.Stop()
	utils.HandleError(err)

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

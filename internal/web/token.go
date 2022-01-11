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

package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/mimiro-io/datahub-cli/internal/config"
	"github.com/rotisserie/eris"
	"github.com/spf13/viper"
	"io/ioutil"
	"net/http"
	"time"
)

func GetServer() string {
	alias := viper.GetString("activelogin")
	if alias != "" {
		if p, err := getLoginAlias(alias); err != nil {
			return ""
		} else {
			return p.Server
		}

	} else {
		return viper.GetString("Server")
	}
}

func ResolveCredentials() (*config.SignedToken, error) {
	alias := viper.GetString("activelogin")
	if alias != "" {
		cfg, err := getLoginAlias(alias)
		if err != nil {
			return nil, err
		}

		loginType := cfg.Type
		if cfg.Type == "" {
			if cfg.ClientId == "" {
				loginType = "token"
			} else {
				loginType = "client"
			}
		}

		switch loginType {
		case "client":
			return exchangeToken(cfg)
		case "user":
			tkn, err := GetValidToken(cfg)
			if err != nil {
				return nil, err
			}
			cfg.SignedToken = tkn
			_ = config.Store(alias, cfg)
			return tkn, nil
		default:
			return &config.SignedToken{AccessToken: cfg.Token}, nil
		}
	}

	token := viper.GetString("token")
	return &config.SignedToken{AccessToken: token}, nil
}

func GetValidToken(cfg *config.Config) (*config.SignedToken, error) {
	tkn := cfg.SignedToken
	if tkn == nil {
		return nil, eris.New("token is missing, please login first")
	}

	claims, err := tkn.Unpack()
	if err != nil {
		return nil, eris.New("failed to unpack the token")
	}
	now := time.Now()
	valid := claims.VerifyExpiresAt(now.Unix(), true)

	if !valid {
		tkn2, err := RefreshToken(claims.Subject, tkn.RefreshToken, cfg)
		if err != nil {
			return nil, eris.Wrap(err, "failed to refresh token")
		}
		return tkn2, nil
	}
	return tkn, nil
}

func exchangeToken(cfg *config.Config) (*config.SignedToken, error) {
	// so, depending on the type, it behaves differently

	request := tokenRequest{
		ClientId:     cfg.ClientId,
		ClientSecret: cfg.ClientSecret,
		Audience:     cfg.Audience,
		GrantType:    "app_credentials",
	}

	if request.Audience == "" {
		request.Audience = cfg.Server
	}
	client := &Client{Server: cfg.Authorizer}
	response := &config.SignedToken{}
	if err := client.doMutate(cfg.Authorizer, "POST", nil, request, response); err != nil {
		return nil, err
	}
	return response, nil
}

func getLoginAlias(alias string) (*config.Config, error) {
	data := &config.Config{}
	if err := config.Load(alias, data); err != nil {
		return nil, err
	}
	return data, nil
}

func RefreshToken(clientId, refreshToken string, cfg *config.Config) (*config.SignedToken, error) {
	tkn := &config.SignedToken{}
	request := tokenRequest{
		ClientId:     clientId,
		GrantType:    "refresh_token",
		RefreshToken: refreshToken,
	}
	if err := doMutate(fmt.Sprintf("%s/oauth/token", cfg.Authorizer), "POST", nil, request, tkn); err != nil {
		return nil, err
	}
	return tkn, nil
}

// Copied from the client to avoid refresh loop issue causing stack overflow. Might want to optimize in the future.
func doMutate(url string, method string, token *config.SignedToken, request interface{}, response interface{}) error {

	content, err := json.Marshal(request)
	if err != nil {
		return ErrFailedToMarshal
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(content))
	if err != nil {
		return eris.Wrap(err, "failed creating http request for some reason")
	}

	if token != nil {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return eris.Wrap(err, "failed to call endpoint")
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return eris.Wrap(err, "impossible to read the result")
	}
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		if response != nil {
			err := json.Unmarshal(bodyBytes, response)
			if err != nil {
				return eris.Wrap(err, "failed to unmarshal response")
			}
		}
	} else {
		// so, we might get back a message object, so lets attempt to parse that
		msg := make(map[string]interface{})
		err = json.Unmarshal(bodyBytes, &msg)
		if err != nil {
			return eris.New("Got http status " + resp.Status)
		}
		if m, ok := msg["message"]; ok {
			return eris.New(fmt.Sprintf("%v: %s", resp.StatusCode, m))
		}
		return eris.New("Got http status " + resp.Status)
	}

	return nil
}

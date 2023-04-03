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
	"github.com/gofrs/uuid"
	"github.com/rotisserie/eris"
	"golang.org/x/oauth2"
	"io"
	"net/http"
	"strings"
)

type tokenRequest struct {
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Audience     string `json:"audience"`
	GrantType    string `json:"grant_type"`
	Code         string `json:"code"`
	RefreshToken string `json:"refresh_token"`
}

type Client struct {
	Server string
}

var (
	ErrNotLoggedIn     = eris.New("user is not logged in")
	ErrFailedToMarshal = eris.New("failed to marshal the payload")
)

// IdResponse can be used as a default response object where only an id is returned
type IdResponse struct {
	Id uuid.UUID `json:"id"`
}

type Header struct {
	Header string
	Value  string
}

func NewClient(server string) (*Client, error) {
	return &Client{
		Server: server,
	}, nil
}

func (c *Client) PostRaw(path string, content []byte) ([]byte, error) {
	tkn, err := c.getValidToken()
	if err != nil {
		return nil, err
	}
	return PostRequest(c.Server, tkn.AccessToken, path, content)
}

func (c *Client) GetRaw(path string) ([]byte, error) {
	tkn, err := c.getValidToken()
	if err != nil {
		return nil, err
	}
	return GetRequest(c.Server, tkn.AccessToken, path)
}

func (c *Client) PutRaw(path string) ([]byte, error) {
	tkn, err := c.getValidToken()
	if err != nil {
		return nil, err
	}
	return PutRequest(c.Server, tkn.AccessToken, path)
}
func (c *Client) DeleteRaw(path string) error {
	tkn, err := c.getValidToken()
	if err != nil {
		return err
	}
	return DeleteRequest(c.Server, tkn.AccessToken, path)
}

func (c *Client) getValidToken() (*oauth2.Token, error) {
	return ResolveCredentials()
}

func (c *Client) Delete(endpoint string) error {
	// check for valid token, and fetch if needed
	tkn, err := c.getValidToken()
	if err != nil {
		return ErrNotLoggedIn
	}
	return c.doDelete(endpoint, tkn)
}

func (c *Client) doDelete(endpoint string, token *oauth2.Token) error {
	url := fmt.Sprintf("%s%s", c.Server, endpoint)
	if strings.HasPrefix(endpoint, "http") { // this is a full url
		url = endpoint
	}

	req, err := http.NewRequest("DELETE", url, nil)
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

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return eris.Wrap(err, "impossible to read the result")
	}

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusAccepted {
		return nil
	} else {
		// so, we might get back a message object, so lets attempt to parse that
		msg := make(map[string]interface{})
		err = json.Unmarshal(bodyBytes, &msg)
		if err != nil {
			return eris.New("Got http status " + resp.Status)
		}
		return eris.New(fmt.Sprintf("%s", msg["message"]))
	}
}

func (c *Client) Get(endpoint string, response interface{}, headers ...Header) error {
	// check for valid token, and fetch if needed
	tkn, err := c.getValidToken()
	if err != nil {
		return ErrNotLoggedIn
	}

	return c.doGet(endpoint, tkn, response, headers...)
}

func (c *Client) doGet(endpoint string, token *oauth2.Token, response interface{}, headers ...Header) error {
	url := fmt.Sprintf("%s%s", c.Server, endpoint)
	if strings.HasPrefix(endpoint, "http") { // this is a full url
		url = endpoint
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return eris.Wrap(err, "failed creating http request for some reason")
	}

	if token != nil {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
	}
	req.Header.Set("Content-Type", "application/json")
	for _, header := range headers {
		req.Header.Set(header.Header, header.Value)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return eris.Wrap(err, "failed to call endpoint")
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return eris.Wrap(err, "impossible to read the result")
	}
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusAccepted {
		err = json.Unmarshal(bodyBytes, response)
		if err != nil {
			return eris.Wrap(err, "failed to unmarshal response")
		}
	} else {
		// so, we might get back a message object, so lets attempt to parse that
		msg := make(map[string]interface{})
		err = json.Unmarshal(bodyBytes, &msg)
		if err != nil {
			return eris.New("Got http status " + resp.Status)
		}

		return eris.New(fmt.Sprintf("%v: %s", resp.StatusCode, msg["message"]))
	}
	return nil
}

func (c *Client) Post(endpoint string, request interface{}, response interface{}) error {
	// check for valid token, and fetch if needed
	tkn, err := c.getValidToken()
	if err != nil {
		return ErrNotLoggedIn
	}

	return c.doMutate(endpoint, "POST", tkn, request, response)
}
func (c *Client) Put(endpoint string, request interface{}, response interface{}) error {
	// check for valid token, and fetch if needed
	tkn, err := c.getValidToken()
	if err != nil {
		return ErrNotLoggedIn
	}

	return c.doMutate(endpoint, "PUT", tkn, request, response)
}

func (c *Client) doMutate(endpoint string, method string, token *oauth2.Token, request interface{}, response interface{}) error {
	url := fmt.Sprintf("%s%s", c.Server, endpoint)
	if strings.HasPrefix(endpoint, "http") { // this is a full url
		url = endpoint
	}

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

	bodyBytes, err := io.ReadAll(resp.Body)
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

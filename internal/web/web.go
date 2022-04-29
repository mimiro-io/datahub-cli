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
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

func PostRequest(server string, token string, path string, content []byte) ([]byte, error) {
	req, err := http.NewRequest("POST", fmt.Sprintf("%s%s", server, path), bytes.NewBuffer(content))

	if err != nil {
		return nil, err
	}

	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	req.Header.Set("Content-Type", "application/json")

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

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		return bodyBytes, nil
	} else {
		// so, we might get back a message object, so lets attempt to parse that
		msg := make(map[string]interface{})
		err = json.Unmarshal(bodyBytes, &msg)
		if err != nil {
			return nil, errors.New("Got http status " + resp.Status)
		}
		return nil, errors.New(fmt.Sprintf("%s", msg["message"]))
	}

}

func DeleteRequest(server string, token string, path string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s%s", server, path), nil)
	if err != nil {
		return err
	}

	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusOK {
		return nil
	} else {
		// so, we might get back a message object, so lets attempt to parse that
		msg := make(map[string]interface{})
		err = json.Unmarshal(bodyBytes, &msg)
		if err != nil {
			return errors.New("Got http status " + resp.Status)
		}
		return errors.New(fmt.Sprintf("%s", msg["message"]))
	}
}

func GetRequest(server string, token string, path string) ([]byte, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s%s", server, path), nil)
	if err != nil {
		return nil, err
	}

	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return bodyBytes, nil
	} else {
		return nil, errors.New("Got http status " + resp.Status)
	}

}

func PutRequest(server string, token string, path string) ([]byte, error) {
	req, err := http.NewRequest("PUT", fmt.Sprintf("%s%s", server, path), nil)
	if err != nil {
		return nil, err
	}

	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	resp, err := http.DefaultClient.Do(req)
	defer func() {
		_ = resp.Body.Close()
	}()
	if err != nil {
		return nil, err
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusOK {
		return bodyBytes, nil
	} else {
		// so, we might get back a message object, so lets attempt to parse that
		msg := make(map[string]interface{})
		err = json.Unmarshal(bodyBytes, &msg)
		if err != nil {
			return nil, errors.New("Got http status " + resp.Status)
		}
		return nil, errors.New(fmt.Sprintf("Http %s - %s", resp.Status, msg["message"]))
	}
}

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
	"fmt"
	"github.com/mimiro-io/datahub-cli/internal/config"
	"github.com/mimiro-io/datahub-cli/internal/web"
	"github.com/skratchdot/open-golang/open"
	"net"
	"net/http"
	"net/url"
)

type UserLogin struct {
}

type AuthCode struct {
	Code     string `json:"code"`
	ClientId string `json:"clientId"`
}

func NewUserLogin() *UserLogin {
	return &UserLogin{}
}

func (l *UserLogin) Login(targetUrl string) (*config.SignedToken, error) {
	var code *AuthCode

	tokenChannel := make(chan *AuthCode)
	errorChannel := make(chan error)
	portChannel := make(chan string)

	go listenForTokenCallback(tokenChannel, errorChannel, portChannel, targetUrl)

	port := <-portChannel

	u, err := url.Parse(fmt.Sprintf("%s/login", targetUrl))
	if err != nil {
		return nil, err
	}
	q, _ := url.ParseQuery("")
	q.Add("redirect_uri", fmt.Sprintf("http://localhost:%v/callback", port))
	u.RawQuery = q.Encode()

	fmt.Println("navigate to the following URL in your browser:\r")
	fmt.Println("\r")

	fmt.Printf("  %s\r\n", u.String())

	_ = open.Start(u.String())

	var err2 error

	select {
	case codeMsg := <-tokenChannel:
		code = codeMsg
	case errorMsg := <-errorChannel:
		err2 = errorMsg
	}

	if err2 != nil {
		return nil, err2
	}

	// have a code, exchange it for a token
	client := &web.Client{Server: targetUrl}
	tkn, err := client.FetchRefreshToken(code.ClientId, code.Code)
	if err != nil {
		return nil, err
	}

	return tkn, nil
}

func listenForTokenCallback(tokenChannel chan *AuthCode, errorChannel chan error, portChannel chan string, targetUrl string) {
	s := &http.Server{
		Addr: "127.0.0.1:0",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", targetUrl)
			code := r.URL.Query().Get("code")
			clientId := r.URL.Query().Get("clientId")

			ac := &AuthCode{
				Code:     code,
				ClientId: clientId,
			}
			tokenChannel <- ac
			if r.Header.Get("Upgrade-Insecure-Requests") != "" {
				http.Redirect(w, r, fmt.Sprintf("%s/auth/success", targetUrl), http.StatusFound)
			}
		}),
	}

	err := listenAndServeWithPort(s, portChannel)

	if err != nil {
		errorChannel <- err
	}
}

func listenAndServeWithPort(srv *http.Server, portChannel chan string) error {
	addr := srv.Addr
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		return err
	}

	portChannel <- port

	return srv.Serve(tcpKeepAliveListener{ln.(*net.TCPListener)})
}

type tcpKeepAliveListener struct {
	*net.TCPListener
}

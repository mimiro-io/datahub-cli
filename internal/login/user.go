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
	"context"
	_ "embed"
	"fmt"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"github.com/mimiro-io/datahub-cli/internal/config"
	pkce "github.com/nirasan/go-oauth-pkce-code-verifier"
	"github.com/rotisserie/eris"
	"github.com/skratchdot/open-golang/open"
	"golang.org/x/oauth2"
	"io"
	"net"
	"net/http"
)

type UserLogin struct {
}

type AuthCode struct {
	Code     string `json:"code"`
	ClientId string `json:"clientId"`
}

//go:embed success.html
var successHtml string

func NewUserLogin() *UserLogin {
	return &UserLogin{}
}

func (l *UserLogin) Login(cfg *config.Config) (*oauth2.Token, error) {
	var code string

	codeChannel := make(chan string)
	errorChannel := make(chan error)
	portChannel := make(chan string)

	go listenForTokenCallback(codeChannel, errorChannel, portChannel, cfg.Authorizer)
	port := <-portChannel

	if cfg.ClientId == "" {
		return nil, eris.New("Login missing configured clientId")
	}

	ctx := oidc.InsecureIssuerURLContext(context.Background(), cfg.Authorizer)
	provider, err := oidc.NewProvider(ctx, cfg.Authorizer)
	if err != nil {
		return nil, err
	}

	oauthCfg := &oauth2.Config{
		ClientID:    cfg.ClientId,
		RedirectURL: fmt.Sprintf("http://localhost:%v/callback", port),
		Endpoint:    provider.Endpoint(),
		Scopes:      []string{"openid", "offline"},
	}

	v, err := pkce.CreateCodeVerifier()
	if err != nil {
		return nil, err
	}

	cc := oauth2.SetAuthURLParam("code_challenge", v.CodeChallengeS256())
	ccm := oauth2.SetAuthURLParam("code_challenge_method", "S256")
	state := uuid.New().String()
	if cfg.Audience == "" {
		cfg.Audience = cfg.Server
	}
	audience := oauth2.SetAuthURLParam("audience", cfg.Audience)
	authURL := oauthCfg.AuthCodeURL(state, cc, ccm, audience)

	fmt.Println("navigate to the following URL in your browser:\r")
	fmt.Println("\r")

	fmt.Printf("  %s\r\n", authURL)

	_ = open.Start(authURL)

	var err2 error

	select {
	case codeMsg := <-codeChannel:
		code = codeMsg
	case errorMsg := <-errorChannel:
		err2 = errorMsg
	}

	if err2 != nil {
		return nil, err2
	}

	// have a code, exchange it for a token
	cv := oauth2.SetAuthURLParam("code_verifier", v.String())
	tkn, err := oauthCfg.Exchange(context.Background(), code, cv)
	if err != nil {
		return nil, err
	}
	cfg.OauthConfig = oauthCfg
	cfg.OauthToken = tkn
	return tkn, nil
}

func listenForTokenCallback(codeChannel chan string, errorChannel chan error, portChannel chan string, targetUrl string) {
	s := &http.Server{
		Addr: "127.0.0.1:31337",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", targetUrl)
			code := r.URL.Query().Get("code")

			codeChannel <- code
			if r.Header.Get("Upgrade-Insecure-Requests") != "" {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_, _ = io.WriteString(w, successHtml)
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

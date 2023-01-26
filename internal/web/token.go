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
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/mimiro-io/datahub-cli/internal/config"
	"github.com/pkg/errors"
	"github.com/rotisserie/eris"
	"github.com/spf13/viper"
)

func GetServer() string {
	alias := viper.GetString("activelogin")
	return GetServerFromAlias(alias)
}

func GetServerFromAlias(alias string) string {
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

func ResolveCredentialsFromAlias(alias string) (*config.SignedToken, error) {
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
		case "admin":
			tkn, err := GetValidToken(cfg)
			if err != nil {
				return nil, err
			}
			cfg.SignedToken = tkn
			_ = config.Store(alias, cfg)
			return tkn, nil
		case "user":
			tkn, err := GetValidToken(cfg)
			if err != nil {
				return nil, err
			}
			cfg.SignedToken = tkn
			_ = config.Store(alias, cfg)
			return tkn, nil
		case "cert":
			tkn, err := GetValidToken(cfg)
			if err != nil {
				return nil, err
			}
			cfg.SignedToken = tkn
			_ = config.Store(alias, cfg)
			return tkn, nil
		case "unsecured":
			// this can be improved.
			return &config.SignedToken{AccessToken: cfg.Token}, nil
		case "token":
			return &config.SignedToken{AccessToken: cfg.Token}, nil
		default:
			return nil, errors.New("unrecognised auth type")
		}
	}

	token := viper.GetString("token")
	return &config.SignedToken{AccessToken: token}, nil
}

func ResolveCredentials() (*config.SignedToken, error) {
	alias := viper.GetString("activelogin")
	return ResolveCredentialsFromAlias(alias)
}

func createJWTForTokenRequest(subject string, audience string, privateKey *rsa.PrivateKey) (string, error) {
	uniqueId := uuid.New()

	claims := jwt.StandardClaims{
		ExpiresAt: time.Now().Add(time.Minute * 1).Unix(),
		Id:        uniqueId.String(),
		Subject:   subject,
		Audience:  audience,
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(privateKey)
	if err != nil {
		return "", err
	}
	return token, nil
}

func GetTokenWithClientCert(cfg *config.Config) (*config.SignedToken, error) {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_assertion_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")

	pem1, err := createJWTForTokenRequest(cfg.ClientId, cfg.Audience, ClientKeyPair.PrivateKey)
	data.Set("client_assertion", pem1)

	reqUrl := cfg.Server + "/security/token"
	res, err := http.PostForm(reqUrl, data)
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(res.Body)
	response := make(map[string]interface{})
	err = decoder.Decode(&response)
	accessToken := response["access_token"].(string)
	token := &config.SignedToken{}
	token.AccessToken = accessToken
	cfg.SignedToken = token
	return token, nil
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
		if cfg.Type == "cert" {

		} else if cfg.Type == "admin" {
			tkn2, err := DoAdminLogin(cfg)
			if err != nil {
				return nil, eris.Wrap(err, "failed to refresh token")
			}
			return tkn2, nil
		} else {
			tkn2, err := RefreshToken(claims.Subject, tkn.RefreshToken, cfg)
			if err != nil {
				return nil, eris.Wrap(err, "failed to refresh token")
			}
			return tkn2, nil
		}
	}
	return tkn, nil
}

func DoAdminLogin(cfg *config.Config) (*config.SignedToken, error) {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", cfg.ClientId)
	data.Set("client_secret", cfg.ClientSecret)

	reqUrl := cfg.Server + "/security/token"
	res, err := http.PostForm(reqUrl, data)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	statusOK := res.StatusCode >= 200 && res.StatusCode < 300
	if !statusOK {
		bodyBytes, _ := ioutil.ReadAll(res.Body)
		return nil, fmt.Errorf("%d: %s", res.StatusCode, string(bodyBytes))
	}

	decoder := json.NewDecoder(res.Body)
	response := make(map[string]interface{})
	err = decoder.Decode(&response)
	accessToken := response["access_token"].(string)
	token := &config.SignedToken{}
	token.AccessToken = accessToken
	cfg.SignedToken = token
	return token, nil
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

func GenerateRsaKeyPair() (*rsa.PrivateKey, *rsa.PublicKey) {
	key, _ := rsa.GenerateKey(rand.Reader, 4096)
	return key, &key.PublicKey
}

func ExportRsaPrivateKeyAsPem(key *rsa.PrivateKey) (string, error) {
	bytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return "", err
	}
	pemBytes := pem.EncodeToMemory(
		&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: bytes,
		},
	)
	return string(pemBytes), nil
}

func ParseRsaPrivateKeyFromPem(pemValue []byte) (*rsa.PrivateKey, error) {
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(pemValue)
	if err != nil {
		return nil, err
	}
	return privateKey, nil
}

func ExportRsaPublicKeyAsPem(key *rsa.PublicKey) (string, error) {
	bytes, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return "", err
	}

	pemBytes := pem.EncodeToMemory(
		&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: bytes,
		},
	)

	return string(pemBytes), nil
}

func ParseRsaPublicKeyFromPem(pemValue []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pemValue)
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	switch pub := pub.(type) {
	case *rsa.PublicKey:
		return pub, nil
	default:
		break // fall through
	}
	return nil, errors.New("Key type is not RSA")
}

type KeyPair struct {
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
	Active     bool
	Expires    uint64
}

func NewKeyPair(privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey, active bool) *KeyPair {
	keyPair := &KeyPair{}
	keyPair.PrivateKey = privateKey
	keyPair.PublicKey = publicKey
	keyPair.Active = active
	return keyPair
}

var ClientKeyPair *KeyPair

func InitialiseClientKeys() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return eris.Wrap(err, "home directory is missing")
	}
	if _, err := os.Stat(home + "/.mim"); os.IsNotExist(err) { // create dir if not exists
		err = os.Mkdir(home+"/.mim", os.ModePerm)
		if err != nil {
			return eris.Wrap(err, "failed creating the .mim dir")
		}
	}
	err = ensureKeys(home + "/.mim")
	if err != nil {
		return eris.Wrap(err, "failed creating client certificate")
	}
	return nil
	// return bolt.Open(home+"/.mim/conf.db", 0666, &bolt.Options{Timeout: 1 * time.Second})
}

func ensureKeys(location string) error {
	// try load private and public key
	fileinfo, err := os.Stat(location + string(os.PathSeparator) + "node_key")
	if err == nil {
		// load data for private key
		content, err := ioutil.ReadFile(location + string(os.PathSeparator) + fileinfo.Name())
		if err != nil {
			return err
		}

		privateKey, err := ParseRsaPrivateKeyFromPem(content)
		if err != nil {
			return err
		}

		// public key
		content, err = ioutil.ReadFile(location + string(os.PathSeparator) + "node_key.pub")
		if err != nil {
			return err
		}

		publicKey, err := ParseRsaPublicKeyFromPem(content)
		if err != nil {
			return err
		}

		ClientKeyPair = NewKeyPair(privateKey, publicKey, true)
	} else {
		// generate files
		privateKey, publicKey := GenerateRsaKeyPair()
		privateKeyPem, err := ExportRsaPrivateKeyAsPem(privateKey)
		if err != nil {
			return err
		}
		publicKeyPem, err := ExportRsaPublicKeyAsPem(publicKey)
		if err != nil {
			return err
		}

		// write keys to files
		err = ioutil.WriteFile(location+string(os.PathSeparator)+"node_key", []byte(privateKeyPem), 0600)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(location+string(os.PathSeparator)+"node_key.pub", []byte(publicKeyPem), 0600)
		if err != nil {
			return err
		}

		ClientKeyPair = NewKeyPair(privateKey, publicKey, true)
	}

	return nil
}

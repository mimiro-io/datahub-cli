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
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v4"
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

func ResolveCredentialsFromAlias(alias string) (*oauth2.Token, error) {
	if alias != "" {
		cfg, err := getLoginAlias(alias)
		if err != nil {
			return nil, err
		}

		if cfg.Type == "" {
			if cfg.ClientId == "" {
				cfg.Type = "token"
			} else {
				cfg.Type = "client"
			}
		}

		tkn, err := GetValidToken(cfg)
		if err != nil {
			return nil, err
		}
		_ = config.Store(alias, cfg)
		return tkn, nil
	}

	token := viper.GetString("token")
	return &oauth2.Token{AccessToken: token}, nil
}

func ResolveCredentials() (*oauth2.Token, error) {
	alias := viper.GetString("activelogin")
	return ResolveCredentialsFromAlias(alias)
}

func createJWTForTokenRequest(subject string, audience string, privateKey *rsa.PrivateKey) (string, error) {
	uniqueId := uuid.New()

	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 1)),
		ID:        uniqueId.String(),
		Subject:   subject,
		Audience:  jwt.ClaimStrings{audience},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(privateKey)
	if err != nil {
		return "", err
	}
	return token, nil
}

func GetTokenWithClientCert(cfg *config.Config) (*oauth2.Token, error) {
	// This may be simplified when rfc7523 private_key_jwt in client credentials flow is supported by golang/oauth2
	// See https://github.com/golang/oauth2/pull/450

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

	return &oauth2.Token{
		AccessToken: accessToken,
	}, nil
}

func GetValidToken(cfg *config.Config) (*oauth2.Token, error) {
	if cfg.OauthToken == nil {
		return nil, eris.New("token is missing, please login first")
	}

	switch cfg.Type {
	case "cert":
		if cfg.OauthToken.Valid() {
			return cfg.OauthToken, nil
		}
		return GetTokenWithClientCert(cfg)
	case "admin":
		fallthrough
	case "client":
		token, err := oauth2.ReuseTokenSource(cfg.OauthToken, cfg.ClientCredentialsConfig.TokenSource(context.Background())).Token()
		if err != nil {
			return nil, err
		}
		cfg.OauthToken = token
		return token, err
	case "user":
		token, err := cfg.OauthConfig.TokenSource(context.Background(), cfg.OauthToken).Token()
		if err != nil {
			return nil, err
		}
		cfg.OauthToken = token
		return token, err
	case "token":
		fallthrough
	case "unsecured":
		return &oauth2.Token{
			AccessToken: cfg.Token,
		}, nil
	default:
		return nil, eris.New("unrecognized auth type")
	}
}

func DoAdminLogin(cfg *config.Config) (*oauth2.Token, error) {
	if cfg.ClientCredentialsConfig == nil {
		// might need to set AuthStyle to params here
		cfg.ClientCredentialsConfig = &clientcredentials.Config{
			ClientID:     cfg.ClientId,
			ClientSecret: cfg.ClientSecret,
			TokenURL:     cfg.Server + "/security/token",
		}
	}

	return cfg.ClientCredentialsConfig.Token(context.Background())
}

func DoClientLogin(cfg *config.Config) (*oauth2.Token, error) {
	ctx := oidc.InsecureIssuerURLContext(context.Background(), cfg.Authorizer)
	provider, err := oidc.NewProvider(ctx, cfg.Authorizer)
	if err != nil {
		return nil, err
	}

	params := url.Values{"audience": []string{cfg.Audience}}
	cc := &clientcredentials.Config{
		ClientID:       cfg.ClientId,
		ClientSecret:   cfg.ClientSecret,
		TokenURL:       provider.Endpoint().TokenURL,
		EndpointParams: params,
	}
	cfg.ClientCredentialsConfig = cc

	return cc.Token(ctx)
}

func getLoginAlias(alias string) (*config.Config, error) {
	data := &config.Config{}
	if err := config.Load(alias, data); err != nil {
		return nil, err
	}
	return data, nil
}

func GenerateRsaKeyPair() (*rsa.PrivateKey, *rsa.PublicKey) {
	key, _ := rsa.GenerateKey(rand.Reader, 4096)
	return key, &key.PublicKey
}

func ExportRsaPrivateKeyAsPem(key *rsa.PrivateKey) (string, error) {
	b, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return "", err
	}
	pemBytes := pem.EncodeToMemory(
		&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: b,
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
	b, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return "", err
	}

	pemBytes := pem.EncodeToMemory(
		&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: b,
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
		content, err := os.ReadFile(location + string(os.PathSeparator) + fileinfo.Name())
		if err != nil {
			return err
		}

		privateKey, err := ParseRsaPrivateKeyFromPem(content)
		if err != nil {
			return err
		}

		// public key
		content, err = os.ReadFile(location + string(os.PathSeparator) + "node_key.pub")
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
		err = os.WriteFile(location+string(os.PathSeparator)+"node_key", []byte(privateKeyPem), 0600)
		if err != nil {
			return err
		}
		err = os.WriteFile(location+string(os.PathSeparator)+"node_key.pub", []byte(publicKeyPem), 0600)
		if err != nil {
			return err
		}

		ClientKeyPair = NewKeyPair(privateKey, publicKey, true)
	}

	return nil
}

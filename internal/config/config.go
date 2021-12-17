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

package config

import (
	"encoding/json"
	"github.com/golang-jwt/jwt"
	"github.com/mimiro-io/datahub-cli/internal/display"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
	bolt "go.etcd.io/bbolt"
	"os"
	"time"
)

var ErrValueNotFound = eris.New("value not found for key")

type SignedToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
	ExpiresIn    int64  `json:"expires_in,omitempty"`
	TokenType    string `json:"token_type"`
}

func (tkn *SignedToken) isValid() bool {
	// we dont care about token validity, we only care about the time
	parser := jwt.Parser{
		SkipClaimsValidation: true,
	}
	claims := &jwt.StandardClaims{}

	_, _, err := parser.ParseUnverified(tkn.AccessToken, claims)
	if err != nil {
		return false
	}
	return claims.VerifyExpiresAt(claims.ExpiresAt, true)
}

func (tkn *SignedToken) Unpack() (*jwt.StandardClaims, error) {
	// we dont care about token validity, we only care about the time
	parser := jwt.Parser{
		SkipClaimsValidation: true,
	}
	claims := &jwt.StandardClaims{}

	_, _, err := parser.ParseUnverified(tkn.AccessToken, claims)
	if err != nil {
		return nil, err
	}
	return claims, nil
}

type Config struct {
	Server       string       `json:"server"`
	Token        string       `json:"token"`
	ClientId     string       `json:"client_id"`
	ClientSecret string       `json:"client_secret"`
	Authorizer   string       `json:"authorizer"`
	Audience     string       `json:"audience"`
	Type         string       `json:"type"`
	SignedToken  *SignedToken `json:"signed_token"`
}

const bucket = "logins"
const (
	AuthenticationServer = "AuthenticationServer"
	Eana360Server        = "Eana360Endpoint"
	Eana360FarmId        = "CurrentFarm"
)

var expectedConfs = []string{
	AuthenticationServer,
}

func PreVerify(cmd *cobra.Command) {
	driver := display.ResolveDriver(cmd)

	failed, err := verifyConfig()
	driver.RenderError(err, true)

	if failed != nil && len(failed) > 0 {
		driver.Msg("")
		driver.RenderError(eris.New("The following config keys are missing:"), false)
		driver.Msg(failed...)
		driver.Msg("")
		os.Exit(1)
	}
}

func verifyConfig() ([]string, error) {
	failed := make([]string, 0)
	db, err := ensureDb()
	defer func() {
		_ = db.Close()
	}()
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			return err
		}
		return nil
	})
	return failed, nil
}

func Dump() (map[string][]byte, error) {
	db, err := ensureDb()
	if err != nil {
		return nil, err
	}

	items := make(map[string][]byte)

	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			// because we close the db, there is some sort of ref to the original
			// byte slice that gets lost, so need to copy it to get it out properly
			dst := make([]byte, len(v))
			copy(dst, v)
			items[string(k)] = dst
		}
		return nil
	})

	return items, err
}

func Store(key string, payload interface{}) error {
	value, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return WriteValue(key, value)
}

func WriteValue(key string, value []byte) error {
	db, err := ensureDb()
	if err != nil {
		return err
	}
	defer func() {
		_ = db.Close()
	}()
	return db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			return err
		}
		return b.Put([]byte(key), value)
	})

}

func Load(key string, response interface{}) error {
	if v, err := GetValue(key); err != nil {
		return err
	} else {
		if v == nil {
			return ErrValueNotFound
		}
		return json.Unmarshal(v, response)
	}
}

// Must return a string or fails gracefully
func Must(key string, driver display.Driver) string {
	v, err := GetValue(key)
	driver.RenderError(err, true)
	return string(v)
}

func GetValue(key string) ([]byte, error) {
	db, err := ensureDb()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = db.Close()
	}()

	var res []byte
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		res = b.Get([]byte(key))
		return nil
	})

	// because we close the db, there is some sort of ref to the original
	// byte slice that gets lost, so need to copy it to get it out properly
	dst := make([]byte, len(res))
	copy(dst, res)

	return dst, nil
}

func Delete(key string) error {
	db, err := ensureDb()
	if err != nil {
		return err
	}
	defer func() {
		_ = db.Close()
	}()
	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		return b.Delete([]byte(key))
	})
}

func ensureDb() (*bolt.DB, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, eris.Wrap(err, "your home is missing, are you ok?")
	}
	if _, err := os.Stat(home + "/.mim"); os.IsNotExist(err) { // create dir if not exists
		err = os.Mkdir(home+"/.mim", os.ModePerm)
		if err != nil {
			return nil, eris.Wrap(err, "failed creating the .moo dir")
		}
	}
	return bolt.Open(home+"/.mim/conf.db", 0666, &bolt.Options{Timeout: 1 * time.Second})
}

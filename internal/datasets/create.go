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

package datasets

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"strings"

	"github.com/mimiro-io/datahub-cli/internal/login"
	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/mimiro-io/datahub-cli/internal/web"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// CreateCmd represents the delete command
func CreateCmd() *cobra.Command {
	var (
		name              []string
		publicNamespaces  []string
		proxy             bool
		virtual           bool
		virtualTransform  string
		proxyRemoteUrl    string
		proxyAuthProvider string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create dataset with given name",
		Long: `Create a dataset with given name, For example:
mim dataset create --name <name>
or
mim dataset create <name>

Optionally, you can specify public namespaces for the dataset:
	mim dataset create <name> --publicNamespaces <namespace1,namespace2>

Optionally, you can specify that the dataset is a proxy dataset:
	mim dataset create <name> --proxy --proxyRemoteUrl <url> --proxyAuthProvider <authProviderName>

Optionally, you can specify that the dataset is a virtual dataset:
	mim dataset create <name> --virtual --transform <BASE64_ENCODED_TRANSFORM>
`,
		Run: func(cmd *cobra.Command, args []string) {
			server, token, err := login.ResolveCredentials()
			utils.HandleError(err)

			pterm.EnableDebugMessages()

			// verify that name is present
			if len(name) == 0 && len(args) > 0 {
				name = strings.Split(args[0], ",")
			}
			if len(name) == 0 {
				pterm.Warning.Println("You must provide a dataset name")
				pterm.Println()
				os.Exit(1)
			}

			createDatasetConfig := &CreateDatasetConfig{}
			createDatasetConfig.PublicNamespaces = publicNamespaces

			if proxy {
				createDatasetConfig.ProxyDatasetConfig = &ProxyDatasetConfig{}
				if proxyRemoteUrl == "" {
					utils.HandleError(errors.New("proxyRemoteUrl required when proxy=true"))
				}
				createDatasetConfig.ProxyDatasetConfig.RemoteUrl = proxyRemoteUrl
				createDatasetConfig.ProxyDatasetConfig.AuthProviderName = proxyAuthProvider
			}
			if virtual {
				createDatasetConfig.VirtualDatasetConfig = &VirtualDatasetConfig{}
				if virtualTransform == "" {
					utils.HandleError(errors.New("transform required when virtual=true"))
				}
				// try to decode transfarm from base64 and return error if it fails
				_, err = base64.StdEncoding.Decode(
					make([]byte, len([]byte(virtualTransform))),
					[]byte(virtualTransform),
				)
				if err != nil {
					utils.HandleError(errors.New("transform is not valid base64"))
				}

				createDatasetConfig.VirtualDatasetConfig.Transform = virtualTransform
			}

			for _, i := range name {
				err = updateDataset(server, token, i, createDatasetConfig)
				if len(name) == 1 {
					utils.HandleError(err)
				}
				if err != nil {
					pterm.Error.Println(err.Error())
					continue
				}
				pterm.Success.Printf("Dataset '%s' has been created", i)
				pterm.Println()
			}
		},
		TraverseChildren: true,
	}
	cmd.Flags().StringSliceVar(&name, "name", nil, "The dataset to create. ")
	cmd.Flags().StringSliceVar(&publicNamespaces, "publicNamespaces", nil, "list of public namespaces for dataset")
	cmd.Flags().BoolVar(&proxy, "proxy", false, "flag dataset as proxy dataset")
	cmd.Flags().StringVar(&proxyRemoteUrl, "proxyRemoteUrl", "", "url of proxied remote dataset")
	cmd.Flags().StringVar(&proxyAuthProvider, "proxyAuthProvider", "", "name of token provider to be used with requests against remote")
	cmd.Flags().BoolVar(&virtual, "virtual", false, "flag dataset as virtual dataset")
	cmd.Flags().StringVar(&virtualTransform, "transform", "", "base64 encoded transform define virtual dataset")

	return cmd
}

type ProxyDatasetConfig struct {
	RemoteUrl           string `json:"remoteUrl"`
	UpstreamTransform   string `json:"upstreamTransform"`
	DownstreamTransform string `json:"downstreamTransform"`
	AuthProviderName    string `json:"authProviderName"`
}

type VirtualDatasetConfig struct {
	Transform string `json:"transform"`
}

type CreateDatasetConfig struct {
	ProxyDatasetConfig   *ProxyDatasetConfig   `json:"ProxyDatasetConfig"`
	VirtualDatasetConfig *VirtualDatasetConfig `json:"virtualDatasetConfig"`
	PublicNamespaces     []string              `json:"publicNamespaces"`
}

func updateDataset(server string, token string, name string, conf *CreateDatasetConfig) error {
	var b []byte
	var err error

	if len(conf.PublicNamespaces) > 0 || conf.ProxyDatasetConfig != nil || conf.VirtualDatasetConfig != nil {
		b, err = json.Marshal(conf)
		if err != nil {
			return err
		}
	}
	path := "/datasets/" + name
	if conf.ProxyDatasetConfig != nil {
		path = path + "?proxy=true"
	}
	_, err = web.PostRequest(server, token, path, b)
	if err != nil {
		return err
	}

	return nil
}

package api

import (
	"encoding/json"

	"github.com/mimiro-io/datahub-cli/internal/web"
)

type SecurityManager struct {
	server string
	token  string
}

func NewSecurityManager(server string, token string) *SecurityManager {
	return &SecurityManager{
		server: server,
		token:  token,
	}
}

type AccessControl struct {
	Resource string
	Action   string
	Deny     bool
}

type ClientInfo struct {
	ClientId  string
	PublicKey []byte
	Deleted   bool
}

func (secManager *SecurityManager) ListClients() (map[string]ClientInfo, error) {
	client, err := web.NewClient(secManager.server)
	if err != nil {
		return nil, err
	}
	clients := make(map[string]ClientInfo)
	if err := client.Get("/security/clients", &clients); err != nil {
		return nil, err
	}
	return clients, nil
}

func (secManager *SecurityManager) AddClient(id string, key []byte) error {
	clientInfo := &ClientInfo{}
	clientInfo.ClientId = id
	clientInfo.PublicKey = key
	clientJSON, err := json.Marshal(clientInfo)
	if err != nil {
		return err
	}

	client, err := web.NewClient(secManager.server)
	if err != nil {
		return err
	}

	_, err = client.PostRaw("/security/clients", clientJSON)
	if err != nil {
		return err
	}

	return nil
}

func (secManager *SecurityManager) DeleteClient(id string) error {
	clientInfo := &ClientInfo{}
	clientInfo.ClientId = id
	clientInfo.Deleted = true
	clientJSON, err := json.Marshal(clientInfo)
	if err != nil {
		return err
	}

	client, err := web.NewClient(secManager.server)
	if err != nil {
		return err
	}

	_, err = client.PostRaw("/security/clients", clientJSON)
	if err != nil {
		return err
	}

	return nil
}

func (secManager *SecurityManager) AddClientAcl(id string, acls []byte) error {
	client, err := web.NewClient(secManager.server)
	if err != nil {
		return err
	}

	_, err = client.PostRaw("/security/clients/"+id+"/acl", acls)
	if err != nil {
		return err
	}

	return nil
}

func (secManager *SecurityManager) GetClientAcl(id string) ([]AccessControl, error) {
	client, err := web.NewClient(secManager.server)
	if err != nil {
		return nil, err
	}

	clientAcls := make([]AccessControl, 0)
	if err := client.Get("/security/clients/"+id+"/acl", &clientAcls); err != nil {
		return nil, err
	}
	return clientAcls, nil
}

type ValueReader struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type ProviderConfig struct {
	Name         string       `json:"name"`
	Type         string       `json:"type"`
	User         *ValueReader `json:"user,omitempty"`
	Password     *ValueReader `json:"password,omitempty"`
	ClientId     *ValueReader `json:"key,omitempty"`
	ClientSecret *ValueReader `json:"secret,omitempty"`
	Audience     *ValueReader `json:"audience,omitempty"`
	GrantType    *ValueReader `json:"grantType,omitempty"`
	Endpoint     *ValueReader `json:"endpoint,omitempty"`
}

func (secManager *SecurityManager) AddTokenProvider(tokenProviderConfig []byte) error {
	client, err := web.NewClient(secManager.server)
	if err != nil {
		return err
	}

	_, err = client.PostRaw("/provider/logins", tokenProviderConfig)
	if err != nil {
		return err
	}

	return nil
}

func (secManager *SecurityManager) ListTokenProviders() ([]ProviderConfig, error) {
	client, err := web.NewClient(secManager.server)
	if err != nil {
		return nil, err
	}
	providers := make([]ProviderConfig, 0)
	if err := client.Get("/provider/logins", &providers); err != nil {
		return nil, err
	}
	return providers, nil
}

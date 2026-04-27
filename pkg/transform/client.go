// Copyright 2026 MIMIRO AS
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0

package transform

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/mimiro-io/datahub-cli/pkg/api"
)

type queryClient struct {
	baseURL string
	bearer  string
	client  *http.Client
}

func newQueryClient(baseURL, bearer string) *queryClient {
	return &queryClient{
		baseURL: baseURL,
		bearer:  bearer,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// queryResultPart decodes the [uri, predicateUri, entity] tuple from /query.
type queryResultPart struct {
	Uri          string
	PredicateUri string
	Entity       *api.Entity
}

func (p *queryResultPart) UnmarshalJSON(data []byte) error {
	var v []json.RawMessage
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	if len(v) < 3 {
		return errors.New("query result tuple too short")
	}
	if err := json.Unmarshal(v[0], &p.Uri); err != nil {
		return err
	}
	if err := json.Unmarshal(v[1], &p.PredicateUri); err != nil {
		return err
	}
	if string(v[2]) == "null" {
		p.Entity = nil
		return nil
	}
	ent := &api.Entity{}
	if err := json.Unmarshal(v[2], ent); err != nil {
		return err
	}
	p.Entity = ent
	return nil
}

func (c *queryClient) Query(startingEntities []string, predicate string, inverse bool, datasets []string) ([]queryResultPart, error) {
	if predicate == "" {
		predicate = "*"
	}
	body := map[string]interface{}{
		"startingEntities": startingEntities,
		"predicate":        predicate,
		"inverse":          inverse,
		"datasets":         datasets,
	}

	raw, err := c.postJSON("/query", body)
	if err != nil {
		return nil, err
	}

	var envelope []json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("decode /query envelope: %w", err)
	}
	if len(envelope) < 2 {
		return nil, errors.New("/query: empty response")
	}

	var parts []queryResultPart
	if err := json.Unmarshal(envelope[1], &parts); err != nil {
		return nil, fmt.Errorf("decode /query tuples: %w", err)
	}
	return parts, nil
}

func (c *queryClient) QuerySingle(entityId string, datasets []string) (*api.Entity, error) {
	body := map[string]interface{}{
		"entityId": entityId,
		"datasets": datasets,
	}

	raw, err := c.postJSON("/query", body)
	if err != nil {
		return nil, err
	}

	entities := make([]api.Entity, 0)
	if err := json.Unmarshal(raw, &entities); err != nil {
		return nil, fmt.Errorf("decode /query single: %w", err)
	}
	if len(entities) < 2 {
		return nil, errors.New("/query single: unexpected response")
	}
	return &entities[1], nil
}

type namespaceLookup struct {
	Prefix    string `json:"prefix"`
	Expansion string `json:"expansion"`
}

func (c *queryClient) GetNamespacePrefix(urlExpansion string) (string, error) {
	path := fmt.Sprintf("/query/namespace?expansion=%s", url.QueryEscape(urlExpansion))
	raw, err := c.get(path)
	if err != nil {
		return "", err
	}
	n := &namespaceLookup{}
	if err := json.Unmarshal(raw, n); err != nil {
		return "", err
	}
	return n.Prefix, nil
}

// Used at startup so AssertNamespacePrefix reuses existing prefixes rather than minting ns* names.
func fetchNamespaces(baseURL, bearer string) (map[string]string, error) {
	c := newQueryClient(baseURL, bearer)
	raw, err := c.get("/namespaces")
	if err != nil {
		return nil, err
	}
	out := make(map[string]string)
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) postJSON(path string, body interface{}) ([]byte, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return c.do("POST", path, bytes.NewReader(payload), "application/json")
}

func (c *queryClient) get(path string) ([]byte, error) {
	return c.do("GET", path, nil, "")
}

func (c *queryClient) do(method, path string, body io.Reader, contentType string) ([]byte, error) {
	if c.baseURL == "" {
		return nil, errors.New("transform: HubURL is required for hub-bound helpers")
	}
	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if c.bearer != "" {
		req.Header.Set("Authorization", "Bearer "+c.bearer)
	}
	req.Header.Set("User-Agent", "datahub-pkg-transform/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		return raw, nil
	}
	msg := map[string]interface{}{}
	if json.Unmarshal(raw, &msg) == nil {
		if m, ok := msg["message"].(string); ok && m != "" {
			return nil, fmt.Errorf("hub %s: %s", resp.Status, m)
		}
	}
	return nil, fmt.Errorf("hub %s", resp.Status)
}

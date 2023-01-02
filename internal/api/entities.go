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

package api

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"time"

	"github.com/pterm/pterm"
	"github.com/tidwall/pretty"
)

type Entity struct {
	ID         string                 `json:"id"`
	Recorded   uint64                 `json:"recorded"`
	IsDeleted  bool                   `json:"deleted"`
	References map[string]interface{} `json:"refs"`
	Properties map[string]interface{} `json:"props"`
}

// NewEntity Create a new entity with global uri and internal resource id
func NewEntity(ID string) *Entity {
	e := Entity{}
	e.ID = ID
	e.Properties = make(map[string]interface{})
	e.References = make(map[string]interface{})
	return &e
}

func NewContext() *Entity {
	e := NewEntity("@context")
	e.Properties["id"] = "@context"
	e.Properties["namespaces"] = make(map[string]interface{})

	return e
}

func NewEntityFromMap(data map[string]interface{}) *Entity {
	e := Entity{}
	e.ID = data["id"].(string)
	e.Properties = data["props"].(map[string]interface{})
	e.References = data["refs"].(map[string]interface{})
	return &e
}

// GetStringProperty returns the string value of the requested property
func (e *Entity) GetStringProperty(propName string) string {
	val := e.Properties[propName]
	if val == nil {
		return ""
	}

	switch v := val.(type) {
	case string:
		return v
	default:
		return ""
	}
}

// GetProperty returns the value of the named property as an interface
func (e *Entity) GetProperty(propName string) interface{} {
	prop := e.Properties[propName]
	switch prop.(type) {
	case map[string]interface{}:
		return NewEntityFromMap(prop.(map[string]interface{}))
	default:
		return prop
	}
}

type EntityManager struct {
	server      string
	token       string
	ctx         context.Context
	datasetType DatasetType
}

type Source interface {
	readEntities(since string, batchSize int, processEntities func([]*Entity) error) error
}

type EntityListDatasource struct {
	Entities []*Entity
}

func (s *EntityListDatasource) readEntities(since string, batchSize int, processEntities func([]*Entity) error) error {
	err := processEntities(s.Entities)
	if err != nil {
		return err
	}
	return nil
}

type StdinDatasetSource struct{}

func (s *StdinDatasetSource) readEntities(since string, batchSize int, processEntities func([]*Entity) error) error {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return err
	}
	if fi.Mode()&os.ModeNamedPipe == 0 {
		return errors.New("no file provided and no stdin pipe")
	} else {
		reader := bufio.NewReader(os.Stdin)
		// var output []byte

		/*for {
			input, err := reader.ReadByte()

			if err != nil && err == io.EOF {
				break
			}
			output = append(output, input)
		}

		return output, nil*/
		read := 0
		entities := make([]*Entity, 0)
		esp := NewEntityStreamParser()
		err = esp.ParseStream(reader, func(entity *Entity) error {
			entities = append(entities, entity)
			read++
			if read == batchSize+2 { // need to account for @context and @continuation
				read = 0
				err := processEntities(entities)
				if err != nil {
					return err
				}
				entities = make([]*Entity, 0)
			}
			return nil
		})

		if err != nil {
			return err
		}

		if read > 0 {
			err = processEntities(entities)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

type httpDatasetSource struct {
	Endpoint       string
	Token          string
	SinceParamName string
}

func (httpDatasetSource *httpDatasetSource) readEntities(since string, batchSize int, processEntities func([]*Entity) error) error {
	// create headers if needed
	endpoint, err := url.Parse(httpDatasetSource.Endpoint)
	if err != nil {
		return err
	}
	if since != "" {
		if httpDatasetSource.SinceParamName == "" {
			httpDatasetSource.SinceParamName = "since"
		}
		q, _ := url.ParseQuery(endpoint.RawQuery)
		q.Add(httpDatasetSource.SinceParamName, since)
		endpoint.RawQuery = q.Encode()
	}

	// set up our request
	req, err := http.NewRequest("GET", endpoint.String(), nil) //
	if err != nil {
		return err
	}

	// we add a cancellable context, and makes sure it gets cancelled when we exit
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// set up a transport with sane defaults, but with a default content timeout of 0 (infinite)
	netTransport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 5 * time.Second,
	}
	netClient := &http.Client{
		Transport: netTransport,
	}

	// we set up a cancel timer, this will cancel the connection after max 30 seconds
	timer := time.AfterFunc(30*time.Second, func() {
		pterm.Warning.Println("Shutting down http connection because of timeout")
		cancel()
	})

	defer timer.Stop()

	if httpDatasetSource.Token != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", httpDatasetSource.Token))
	}

	// do get
	res, err := netClient.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		return handleHttpError(res)
	}

	// set default batch size if not specified
	if batchSize <= 0 {
		batchSize = 1000
	}

	read := 0
	entities := make([]*Entity, 0)
	esp := NewEntityStreamParser()
	err = esp.ParseStream(res.Body, func(entity *Entity) error {
		timer.Reset(2 * time.Second) // we reset this everytime we get data, if we dont get anything more for 2 seconds, we cancel
		entities = append(entities, entity)
		read++
		if read == batchSize+2 { // need to account for @context and @continuation
			read = 0
			err := processEntities(entities)
			if err != nil {
				return err
			}
			entities = make([]*Entity, 0)
		}
		return nil
	})
	if err != nil {
		return err
	}

	if read > 0 {
		err = processEntities(entities)
		if err != nil {
			return err
		}
	}

	return nil
}

type Sink interface {
	ProcessEntities(entities []*Entity) error
	Start()
	End()
}

type RawSink struct {
	header       bool
	footer       bool
	continuation *Entity
	isFirst      bool
}

func (s *RawSink) Start() {
	s.isFirst = true
	_, _ = os.Stdout.Write([]byte("["))
}

func (s *RawSink) End() {
	if s.continuation != nil {
		c := s.continuation.Properties
		c["id"] = "@continuation"
		out, err := json.Marshal(c)
		if err == nil {
			_, _ = os.Stdout.Write(out)
		}
	}
	_, _ = os.Stdout.Write([]byte("]\n"))
}

func (s *RawSink) ProcessEntities(entities []*Entity) error {
	for _, e := range entities {
		if s.isFirst {
			s.isFirst = false
		} else {
			_, _ = os.Stdout.Write([]byte(","))
		}
		var layer []byte
		var err error
		if e.ID == "@context" && !s.header {
			layer, err = json.Marshal(e.Properties)
			s.header = true
		} else if e.ID == "@continuation" {
			s.continuation = e
		} else {
			layer, err = json.Marshal(e)
		}

		if err != nil {
			return err
		}
		if layer != nil {
			_, _ = os.Stdout.Write(layer)
		}
	}
	return nil
}

type PrettySink struct{}

func (s *PrettySink) Start() {}
func (s *PrettySink) End()   {}

func (s *PrettySink) ProcessEntities(entities []*Entity) error {
	for i, e := range entities {
		separator := ","
		if i == len(entities)-1 {
			separator = ""
		}

		var layer []byte
		var err error
		if e.ID == "@context" {
			layer, err = json.Marshal(e.Properties)
		} else if e.ID == "@continuation" {
			c := e.Properties
			c["id"] = "@continuation"
			layer, err = json.Marshal(c)
		} else {
			layer, err = json.Marshal(e)
		}

		if err != nil {
			return err
		}

		f := pretty.Pretty(layer)
		result := pretty.Color(f, nil)

		fmt.Printf("%s%s", string(result), separator)
	}
	return nil
}

type ConsoleSink struct {
	out [][]string
}

func (s *ConsoleSink) ProcessEntities(entities []*Entity) error {
	for _, e := range entities {
		if e == nil {
			continue
		}
		if e.ID == "@context" {
			s.prettyContext(e.Properties)
		} else if e.ID == "@continuation" {
			pterm.DefaultSection.Println(fmt.Sprintf("Continuation token: %s", e.Properties["token"]))
		} else {
			s.out = append(s.out, []string{
				e.ID,
				fmt.Sprintf("%d", e.Recorded),
				fmt.Sprintf("%v", e.IsDeleted),
				fmt.Sprintf("%v", e.Properties),
				fmt.Sprintf("%v", e.References),
			})
		}
	}

	return nil
}

func (s *ConsoleSink) Start() {
	s.out = make([][]string, 0)
	s.out = append(s.out, []string{"Id", "Recorded", "Deleted", "Props", "Refs"})
}

func (s *ConsoleSink) End() {
	pterm.DefaultTable.WithHasHeader().WithData(s.out).Render()
	pterm.Println()
}

func (s *ConsoleSink) prettyContext(context map[string]interface{}) {
	pterm.DefaultSection.Println("Namespaces:")
	namespaces := context["namespaces"].(map[string]interface{})

	keys := make([]string, 0, len(namespaces))
	for k := range namespaces {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make([][]string, 0)
	out = append(out, []string{"#", "Namespace"})
	for _, k := range keys {
		out = append(out, []string{
			k,
			fmt.Sprintf("%s", namespaces[k]),
		})
	}

	pterm.DefaultTable.WithHasHeader().WithData(out).Render()
}

type CollectorSink struct {
	Entities          []*Entity
	ContinuationToken string
	Context           *Entity
}

func (s *CollectorSink) Start() {}
func (s *CollectorSink) End()   {}

func (s *CollectorSink) ProcessEntities(entities []*Entity) error {
	es := make([]*Entity, 0)
	for _, e := range entities {
		if e.ID != "@continuation" && e.ID != "@context" {
			es = append(es, e)
		} else if e.ID == "@continuation" {
			s.ContinuationToken = e.Properties["token"].(string)
		}
	}
	s.Entities = es
	return nil
}

type Pipeline struct {
	source Source
	sink   Sink
}

func NewPipeline(source Source, sink Sink) *Pipeline {
	return &Pipeline{
		source: source,
		sink:   sink,
	}
}

func (pipeline *Pipeline) Sync(ctx context.Context, since string, limit int) error {
	keepReading := true

	pipeline.sink.Start()
	defer pipeline.sink.End()
	total := 0
	for keepReading {
		err := pipeline.source.readEntities(since, limit, func(entities []*Entity) error {
			select {
			// if the cancellable context is cancelled, ctx.Done will trigger, and it will break out. The only way I
			// found to do so, was to trigger an error, and then check for that in the jobs.Runner.
			case <-ctx.Done():
				keepReading = false
				return errors.New("got job interrupt")
			default:
				var err error
				incomingEntityCount := len(entities)

				if incomingEntityCount > 0 {
					// write to sink
					err = pipeline.sink.ProcessEntities(entities)
					if err != nil {
						return err
					}
				}
				total += incomingEntityCount
				if total >= limit {
					keepReading = false
				}
				if incomingEntityCount < limit { // not enough data
					keepReading = false
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

type DatasetType string

const (
	Entities DatasetType = "entities"
	Changes  DatasetType = "changes"
)

func NewEntityManager(server string, token string, ctx context.Context, dsType DatasetType) *EntityManager {
	return &EntityManager{
		server:      server,
		token:       token,
		ctx:         ctx,
		datasetType: dsType,
	}
}

func (em *EntityManager) Read(dataset string, since string, limit int, reverse bool, sink Sink) error {
	endpoint, err := em.buildUrl(em.server, dataset, em.datasetType, limit, reverse)
	if err != nil {
		return err
	}

	source := &httpDatasetSource{
		Endpoint:       endpoint.String(),
		Token:          em.token,
		SinceParamName: "from",
	}

	pipeline := NewPipeline(source, sink)

	return pipeline.Sync(em.ctx, since, limit)
}

func (em *EntityManager) buildUrl(server string, dataset string, t DatasetType, limit int, reverse bool) (*url.URL, error) {
	endpoint, err := url.Parse(fmt.Sprintf("%s/datasets/%s/%s", server, dataset, t))
	if err != nil {
		return nil, err
	}
	q, _ := url.ParseQuery(endpoint.RawQuery)
	if limit > 0 {
		q.Add("limit", fmt.Sprintf("%d", limit))
	}
	if reverse {
		q.Add("reverse", "true")
	}

	endpoint.RawQuery = q.Encode()
	return endpoint, nil
}

func handleHttpError(response *http.Response) error {
	if response.StatusCode == 404 {
		return errors.New("not found. dataset URL returned 404")
	} else if response.StatusCode == 500 {
		return errors.New("server error")
	} else if response.StatusCode == 403 {
		return errors.New("not authorised")
	} else {
		return errors.New(fmt.Sprintf("some other http error code (%d)", response.StatusCode))
	}
}

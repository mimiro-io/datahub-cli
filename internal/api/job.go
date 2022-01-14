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
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mimiro-io/datahub-cli/internal/web"
	"strings"
	"time"

	"github.com/mimiro-io/datahub-cli/internal/login"
	"github.com/mimiro-io/datahub-cli/internal/utils"
)

type JobTrigger struct {
	TriggerType      string `json:"triggerType"`
	JobType          string `json:"jobType"`
	Schedule         string `json:"schedule"`
	MonitoredDataset string `json:"monitoredDataset"`
}
type Job struct {
	Title       string                 `json:"title"`
	Id          string                 `json:"id"`
	Description string                 `json:"description"`
	Tags        []string               `json:"tags"`
	Source      map[string]interface{} `json:"source"`
	Sink        map[string]interface{} `json:"sink"`
	Transform   map[string]interface{} `json:"transform"`
	Triggers    []JobTrigger           `json:"triggers"`
	Paused      bool                   `json:"paused"`
	BatchSize   int                    `json:"batchSize"`
}

type JobHistory struct {
	Id        string    `json:"id"`
	Title     string    `json:"title"`
	Start     time.Time `json:"start"`
	End       time.Time `json:"end"`
	LastError string    `json:"lastError"`
}

type JobOutput struct {
	Job     Job
	History *JobHistory
}

type JobManager struct {
	server string
	token  string
}

func NewJobManager(server string, token string) *JobManager {
	return &JobManager{
		server: server,
		token:  token,
	}
}

// GetJob gets a job given its id, or error if not found.
func (jm *JobManager) GetJob(jobId string) (*Job, error) {
	res, err := web.GetRequest(jm.server, jm.token, fmt.Sprintf("/jobs/%s", jobId))
	if err != nil {
		if strings.Index(err.Error(), "404") > -1 {
			return nil, errors.New(fmt.Sprintf("No job for job id - %s found on server %s", jobId, jm.server))
		}
		return nil, err
	}

	job := &Job{}
	err = json.Unmarshal(res, job)
	if err != nil {
		return nil, err
	}

	return job, nil
}

// UpdateJob updates a job
func (jm *JobManager) UpdateJob(job *Job) (*Job, error) {
	data, err := json.Marshal(job)
	if err != nil {
		return nil, err
	}

	_, err = web.PostRequest(jm.server, jm.token, "/jobs", data)
	if err != nil {
		return nil, err
	}
	return job, nil
}

// AddJob adds a new job to the scheduler
func (jm *JobManager) AddJob(config []byte) (*Job, error) {
	job := &Job{}
	err := json.Unmarshal(config, job)
	if err != nil {
		return nil, err
	}

	_, err = web.PostRequest(jm.server, jm.token, "/jobs", config)
	if err != nil {
		return nil, err
	}
	return job, err
}

// AddTransform adds a transform to an existing job and updates the job on the server
func (jm *JobManager) AddTransform(job *Job, transform string) (*Job, error) {
	// attach the transform
	tfr := make(map[string]interface{})
	tfr["Type"] = "JavascriptTransform"
	tfr["Code"] = transform

	job.Transform = tfr

	return jm.UpdateJob(job)
}

func GetJobsCompletion(pattern string) []string {
	server, token, err := login.ResolveCredentials()
	utils.HandleError(err)

	jobs, err := web.GetRequest(server, token, "/jobs")
	utils.HandleError(err)

	joblist := make([]Job, 0)
	err = json.Unmarshal(jobs, &joblist)
	utils.HandleError(err)

	var jobIds []string

	for _, job := range joblist {
		if strings.HasPrefix(strings.ToLower(job.Title), strings.ToLower(pattern)) {
			jobIds = append(jobIds, job.Title)
		}
	}
	return jobIds
}

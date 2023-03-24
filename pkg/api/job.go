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

type JobStatus struct {
	JobId    string    `json:"jobId"`
	JobTitle string    `json:"jobTitle"`
	Started  time.Time `json:"started"`
}

type JobOutput struct {
	Job     Job
	History *JobHistory
}

type JobOutputViewItem struct {
	Job         Job          `json:"job"`
	History     *JobHistory  `json:"history"`
	HistoryView *HistoryView `json:"historyView"`
}

type HistoryView struct {
	LastRun      string
	LastDuration string
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

// DeleteJob deletes a job
func (jm *JobManager) DeleteJob(jobId string) error {
	err := web.DeleteRequest(jm.server, jm.token, fmt.Sprintf("/jobs/%s", jobId))
	if err != nil {
		return err
	}
	return nil
}

// GetJobStatus get the status for a given id or all running jobs
func (jm *JobManager) GetJobStatus(jobId string) ([]JobStatus, error) {
	endpoint := "/jobs/_/status"
	if jobId != "" {
		endpoint = fmt.Sprintf("/job/%s/status", jobId)
	}

	data, err := web.GetRequest(jm.server, jm.token, endpoint)
	if err != nil {
		return nil, err
	}

	jobs := make([]JobStatus, 0)
	err = json.Unmarshal(data, &jobs)
	if err != nil {
		return nil, err
	}
	return jobs, nil
}

func (jm *JobManager) GetJobHistories() []JobHistory {

	body, err := web.GetRequest(jm.server, jm.token, "/jobs/_/history")
	utils.HandleError(err)

	histories := make([]JobHistory, 0)
	err = json.Unmarshal(body, &histories)
	utils.HandleError(err)

	return histories
}

func (jm *JobManager) GetJobHistoryForId(id string) (JobHistory, error) {
	histories := jm.GetJobHistories()

	for _, hist := range histories {
		if hist.Id == id {
			return hist, nil
		}
	}
	return JobHistory{}, errors.New(fmt.Sprintf("No history found for job %s", id))
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

func (jm *JobManager) ResolveId(title string) string {
	jobList := jm.GetJobs()

	for _, job := range jobList {
		if job.Title == title {
			return job.Id
		}
	}
	return title
}

func (jm *JobManager) GetJobs() []Job {
	allJobs, err := web.GetRequest(jm.server, jm.token, "/jobs")
	utils.HandleError(err)

	jobList := make([]Job, 0)
	err = json.Unmarshal(allJobs, &jobList)
	utils.HandleError(err)

	return jobList
}

func (jm *JobManager) GetJobListWithHistory() []JobOutputViewItem {
	histories := jm.GetJobHistories()
	jobs := jm.GetJobs()

	historyMap := make(map[string]JobHistory)
	for _, jh := range histories {
		historyMap[jh.Id] = jh
	}

	output := make([]JobOutputViewItem, 0)
	for _, job := range jobs {
		out := JobOutputViewItem{
			Job: job,
		}
		if h, ok := historyMap[job.Id]; ok {
			out.History = &h
		}
		if h, ok := historyMap[job.Id+"_temp"]; ok {
			if out.History == nil || h.Start.After(out.History.Start) {
				out.History = &h
			}
		}
		if out.History != nil {
			timed := out.History.End.Sub(out.History.Start)
			hv := HistoryView{
				LastDuration: fmt.Sprintf("%s", timed),
				LastRun:      out.History.Start.Format(time.RFC3339),
			}
			out.HistoryView = &hv
		}
		output = append(output, out)
	}
	return output
}

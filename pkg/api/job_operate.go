package api

import (
	"context"
	"fmt"
	"github.com/mimiro-io/datahub-cli/internal/web"
)

type JobOperation struct {
	server string
	token  string
}

type JobOperationResponse struct {
	JobId string `json:"jobId"`
}

// NewJobOperation will return a new JobOperation service. If you are already using the JobManager, then
// this will have been initiated for you already, and you can call jm.Operate.XX.
// Note that all methods take an ignored context, these should be refactored later together with a rework of the
// http clients, as they are a bit all over the place.
func NewJobOperation(server string, token string) *JobOperation {
	return &JobOperation{server: server, token: token}
}

// Pause will pause the operation of a job
func (o *JobOperation) Pause(_ context.Context, jobId string) (JobOperationResponse, error) {
	return web.Put[JobOperationResponse](o.server, o.token, fmt.Sprintf("/job/%s/pause", jobId))
}

// Resume will resume a paused job
func (o *JobOperation) Resume(_ context.Context, jobId string) (JobOperationResponse, error) {
	return web.Put[JobOperationResponse](o.server, o.token, fmt.Sprintf("/job/%s/resume", jobId))
}

// Reset will reset the since tokens on a job, running the job from the start of the dataset
func (o *JobOperation) Reset(_ context.Context, jobId string, since string) (JobOperationResponse, error) {
	endpoint := fmt.Sprintf("/job/%s/reset", jobId)
	if since != "" {
		endpoint = "?since=" + since
	}

	return web.Put[JobOperationResponse](o.server, o.token, endpoint)
}

// Kill will attempt to stop an already running job
func (o *JobOperation) Kill(_ context.Context, jobId string) (JobOperationResponse, error) {
	return web.Put[JobOperationResponse](o.server, o.token, fmt.Sprintf("/job/%s/kill", jobId))
}

// Run will attempt to manually start a job
func (o *JobOperation) Run(_ context.Context, jobId string, jobType string) (JobOperationResponse, error) {
	endpoint := fmt.Sprintf("/job/%s/run", jobId)
	if jobType != "" {
		endpoint = endpoint + "?jobType=" + jobType
	}
	return web.Put[JobOperationResponse](o.server, o.token, endpoint)
}

// Test is just for testing and will always fail
func (o *JobOperation) Test(_ context.Context, jobId string) (JobOperationResponse, error) {
	return web.Put[JobOperationResponse](o.server, o.token, fmt.Sprintf("/job/%s/testx", jobId))
}

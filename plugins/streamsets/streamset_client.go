package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	url2 "net/url"
	"os"
	"time"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/serr"
)

// Custom error types
type StreamSetsError struct {
	Type    string
	Message string
}

func (e StreamSetsError) Error() string {
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// JobData represents the StreamSets job structure
type JobData struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	RuntimeParameters string `json:"runtimeParameters"`
	// Add other fields as needed based on the actual API response
}

// JobStatus represents the job status response
type JobStatus struct {
	Status       string `json:"status"`
	Color        string `json:"color"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

// StreamSetsClient handles all API interactions
type StreamSetsClient struct {
	baseURL    string
	credID     string
	token      string
	httpClient *http.Client
}

// NewStreamSetsClient creates a new client instance
func NewStreamSetsClient() (*StreamSetsClient, error) {
	credID := os.Getenv("CRED_ID")
	if credID == "" {
		credID = "myId" // Default value for testing
	}

	token := os.Getenv("CRED_TOKEN")
	if token == "" {
		token = "MyToken" // Default value for testing
	}

	// Validate credentials
	if credID == "myId" || token == "MyToken" {
		logger.Warn("Using default credentials - ensure CRED_ID and CRED_TOKEN environment variables are set for production")
	}

	return &StreamSetsClient{
		baseURL: ControlHubAPIBaseURL,
		credID:  credID,
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// getHeaders returns the standard headers for API requests
func (c *StreamSetsClient) getHeaders() map[string]string {
	return map[string]string{
		"Content-Type":          "application/json",
		"X-Requested-By":        "go-streamsets-trigger",
		"X-SS-REST-CALL":        "true",
		"X-SS-App-Auth-Token":   c.token,
		"X-SS-App-Component-Id": c.credID,
	}
}

// makeRequest performs an HTTP request with proper headers
func (c *StreamSetsClient) makeRequest(method, url string, body interface{}) (*http.Response, error) {
	var reqBody []byte
	var err error

	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, serr.Wrap(err, "Failed to marshal request body")
		}
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, serr.Wrap(err, "Failed to create request")
	}

	// Add headers
	for k, v := range c.getHeaders() {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, serr.Wrap(err, "Request failed")
	}

	return resp, nil
}

// getJob retrieves job information
func (c *StreamSetsClient) getJob(jobID string) (*JobData, error) {
	logger.Info("Retrieving job information from Control Hub")

	url := fmt.Sprintf("%s%s/%s", c.baseURL, JobRunnerAPIPath, jobID)
	resp, err := c.makeRequest("GET", url, nil)
	if err != nil {
		return nil, serr.Wrap(err, "Failed to get job")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, StreamSetsError{
			Type:    "APIError",
			Message: fmt.Sprintf("Job with ID '%s' not found: HTTP %d", jobID, resp.StatusCode),
		}
	}

	var jobData JobData
	if err := json.NewDecoder(resp.Body).Decode(&jobData); err != nil {
		return nil, serr.Wrap(err, "Failed to decode job response")
	}

	logger.Info(fmt.Sprintf("Found Job with name '%s'", jobData.Name))
	return &jobData, nil
}

// JobsList represents a list of jobs
type JobsList struct {
	Jobs []JobData `json:"jobs"`
}

// findJobByName searches for a job by name and returns its ID
func (c *StreamSetsClient) findJobByName(jobName string) (string, error) {
	logger.Info(fmt.Sprintf("Searching for job with name '%s'", jobName))

	// Get all jobs
	url := fmt.Sprintf("%s%s", c.baseURL, "/jobrunner/rest/v1/saql/jobs/search?search=name=="+url2.QueryEscape(JobName))
	fmt.Println("**-> url", url)

	// bod := map[string]string{
	// 	"query": fmt.Sprintf("name == %s", jobName),
	// }

	resp, err := c.makeRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", serr.Wrap(err, "Failed to get jobs list")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", StreamSetsError{
			Type:    "APIError",
			Message: fmt.Sprintf("Failed to get jobs list: HTTP %d", resp.StatusCode),
		}
	}

	type JobResp struct {
		Id           string `json:"id"`
		Name         string `json:"name"`
		PipelineName string `json:"pipelineName"`
		// Updated int64 `json:"lastModifiedOn"`
	}

	type searchResponse struct {
		Data []JobResp `json:"data"`
	}

	var searchResp searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return "", serr.Wrap(err, "Failed to decode jobs list response")
	}

	// Find job by name
	for _, job := range searchResp.Data {
		if job.Name == jobName {
			// Extract jobID from the job data
			// This assumes the job ID is available in the response
			// You might need to adjust this based on the actual API response structure
			jobID := job.Id
			logger.Info(fmt.Sprintf("Found job with name '%s', ID: %s, Pipeline: %s", jobName, jobID, job.PipelineName))
			return jobID, nil
		}
	}

	return "", StreamSetsError{
		Type:    "JobNotFoundError",
		Message: fmt.Sprintf("Job with name '%s' not found", jobName),
	}
}

// getJobStatus retrieves the current job status
func (c *StreamSetsClient) getJobStatus(jobID string) (*JobStatus, error) {
	url := fmt.Sprintf("%s%s/%s/currentStatus", c.baseURL, JobRunnerAPIPath, jobID)
	resp, err := c.makeRequest("GET", url, nil)
	if err != nil {
		return nil, serr.Wrap(err, "Failed to get job status")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, StreamSetsError{
			Type:    "APIError",
			Message: fmt.Sprintf("Failed to get job status: HTTP %d", resp.StatusCode),
		}
	}

	var status JobStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, serr.Wrap(err, "Failed to decode status response")
	}

	return &status, nil
}

// updateJobParameters updates the job's runtime parameters
func (c *StreamSetsClient) updateJobParameters(jobID string, job *JobData) error {
	logger.Info("Setting Job parameters...")

	// Convert runtime parameters to JSON string
	paramsJSON, err := json.Marshal(RuntimeParameters)
	if err != nil {
		return serr.Wrap(err, "Failed to marshal runtime parameters")
	}
	job.RuntimeParameters = string(paramsJSON)

	url := fmt.Sprintf("%s%s/%s", c.baseURL, JobRunnerAPIPath, jobID)
	resp, err := c.makeRequest("POST", url, job)
	if err != nil {
		return serr.Wrap(err, "Failed to update job parameters")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return StreamSetsError{
			Type:    "APIError",
			Message: fmt.Sprintf("Failed to update job parameters: HTTP %d", resp.StatusCode),
		}
	}

	return nil
}

// startJob starts the StreamSets job
func (c *StreamSetsClient) startJob(jobID string) error {
	logger.Info("Starting Job...")

	url := fmt.Sprintf("%s%s/%s/start", c.baseURL, JobRunnerAPIPath, jobID)
	resp, err := c.makeRequest("POST", url, map[string]interface{}{})
	if err != nil {
		return serr.Wrap(err, "Failed to start job")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return StreamSetsError{
			Type:    "APIError",
			Message: fmt.Sprintf("Failed to start job: HTTP %d", resp.StatusCode),
		}
	}

	return nil
}

// waitForJobActive waits for the job to become active
func (c *StreamSetsClient) waitForJobActive(jobID string) error {
	logger.Info("Waiting for Job to become ACTIVE...")

	waitSeconds := 0
	ticker := time.NewTicker(PollingFrequencySeconds * time.Second)
	defer ticker.Stop()

	for waitSeconds <= MaxWaitSecondsForJobToBeActive {
		status, err := c.getJobStatus(jobID)
		if err != nil {
			logger.Warn(fmt.Sprintf("Warning: Error checking job status: %v", err))
			// Continue polling instead of failing immediately
		} else {
			logger.Info(fmt.Sprintf("Current job status: %s", status.Status))

			if status.Status == "ACTIVE" {
				logger.Info("Job status is ACTIVE")
				return nil
			}

			if status.Status == "DEACTIVATING" {
				logger.Info("Job may have completed already and the pipeline is now deactivating")
				return nil
			}

			if status.Status == "INACTIVE_ERROR" || status.Status == "ACTIVATION_ERROR" {
				return StreamSetsError{
					Type:    "JobActivationError",
					Message: fmt.Sprintf("Job activation failed with status %s: %s", status.Status, status.ErrorMessage),
				}
			}
		}

		<-ticker.C
		waitSeconds += PollingFrequencySeconds
	}

	return StreamSetsError{
		Type:    "JobTimeoutError",
		Message: fmt.Sprintf("Job activation timeout after %d seconds", MaxWaitSecondsForJobToBeActive),
	}
}

// monitorJobCompletion monitors the job until completion or timeout
func (c *StreamSetsClient) monitorJobCompletion(jobID string) (*JobStatus, error) {
	finalStatuses := map[string]bool{
		"INACTIVE":             true,
		"FINISHED":             true,
		"FINISHING":            true,
		"FINISHED_WITH_ERRORS": true,
		"FAILED":               true,
		"ERROR":                true,
		"INACTIVE_ERROR":       true,
	}

	logger.Info(fmt.Sprintf("Monitoring job for completion (timeout: %d seconds)...", MaxWaitSecondsForJobCompletion))

	waitSeconds := 0
	ticker := time.NewTicker(PollingFrequencySeconds * time.Second)
	defer ticker.Stop()

	var lastStatus *JobStatus

	for waitSeconds <= MaxWaitSecondsForJobCompletion {
		status, err := c.getJobStatus(jobID)
		if err != nil {
			logger.Warn(fmt.Sprintf("Warning: Error checking job status: %v", err))
			// Continue polling instead of failing immediately
		} else {
			lastStatus = status
			logger.Info(fmt.Sprintf("Job status: %s, color: %s", status.Status, status.Color))

			if finalStatuses[status.Status] {
				return status, nil
			}
		}

		<-ticker.C
		waitSeconds += PollingFrequencySeconds
	}

	// Timeout case
	if lastStatus != nil && !finalStatuses[lastStatus.Status] {
		return lastStatus, StreamSetsError{
			Type: "JobTimeoutError",
			Message: fmt.Sprintf("Job execution exceeded %d second timeout. Current status: %s, color: %s. "+
				"The job is still running - check StreamSets Control Hub console.",
				MaxWaitSecondsForJobCompletion, lastStatus.Status, lastStatus.Color),
		}
	}

	return lastStatus, nil
}

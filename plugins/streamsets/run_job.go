package streamsets

import (
	"fmt"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/serr"
)

// Configuration constants
const (
	JobName                        = "CarInfo" // Replace with your actual job name
	PollingFrequencySeconds        = 10
	MaxWaitSecondsForJobToBeActive = 120
	MaxWaitSecondsForJobCompletion = 300
	ControlHubAPIBaseURL           = "https://na01.hub.streamsets.com"
	JobRunnerAPIPath               = "/jobrunner/rest/v1/job"
)

// Runtime parameters for the job
var RuntimeParameters = map[string]string{
	// "p_query_interval": "15",
}

/*func main() {
	logger.Info("Starting StreamSets Job")

	// Run the pipeline
	if err := RunJob(); err != nil {
		logger.LogErr(err)
		return // serr.Wrap(err, "Failed to trigger StreamSets pipeline")
	}

	logger.Info("StreamSets Job completed successfully")
}
*/

// RunJob executes the complete pipeline trigger and monitoring flow
func RunJob() error {
	// Initialize client
	client, err := NewStreamSetsClient()
	if err != nil {
		return serr.Wrap(err, "Failed to initialize StreamSets client")
	}

	// Step 1: Find job by name
	logger.Info(fmt.Sprintf("Searching for job with name: %s", JobName))
	jobID, err := client.findJobByName(JobName)
	if err != nil {
		return serr.Wrap(err, fmt.Sprintf("Failed to find job with name '%s'", JobName))
	}
	logger.Info(fmt.Sprintf("Found job with Id: %s", jobID))

	// Step 2: Get job information
	job, err := client.getJob(jobID)
	if err != nil {
		return serr.Wrap(err)
	}
	_ = job

	// Step 3: Get job status
	status, err := client.getJobStatus(jobID)
	if err != nil {
		return serr.Wrap(err)
	}

	logger.Info(fmt.Sprintf("Job status is '%s'", status.Status))

	// Ensure job is inactive before starting
	if status.Status != "INACTIVE" {
		return StreamSetsError{
			Type:    "ConfigurationError",
			Message: fmt.Sprintf("Job must have status 'INACTIVE' to be started, current status: %s", status.Status),
		}
	}

	/*	// Step 4: Update runtime parameters
		if err := client.updateJobParameters(jobID, job); err != nil {
			return serr.Wrap(err)
		}
	*/
	// Step 5: Start the job
	if err := client.startJob(jobID); err != nil {
		return serr.Wrap(err)
	}

	// Step 6: Wait for job to become active
	if err := client.waitForJobActive(jobID); err != nil {
		return serr.Wrap(err)
	}

	// Step 7: Monitor job completion
	finalStatus, err := client.monitorJobCompletion(jobID)
	if err != nil {
		return serr.Wrap(err)
	}

	// Step 8: Check final status
	logger.Info(fmt.Sprintf("Job completed with status: %s, color: %s", finalStatus.Status, finalStatus.Color))

	// Success path
	if finalStatus.Status == "INACTIVE" && (finalStatus.Color == "GRAY" || finalStatus.Color == "GREEN") {
		logger.Info("Job completed successfully")
		return nil
	}

	// Failure path
	failureStatuses := []string{"INACTIVE", "FINISHED_WITH_ERRORS", "FAILED", "ERROR", "INACTIVE_ERROR"}
	for _, fs := range failureStatuses {
		if finalStatus.Status == fs && finalStatus.Color != "GREEN" && finalStatus.Color != "GRAY" {
			msg := fmt.Sprintf("Job failed with status %s, color %s", finalStatus.Status, finalStatus.Color)
			if finalStatus.ErrorMessage != "" {
				msg += fmt.Sprintf(", error: %s", finalStatus.ErrorMessage)
			}
			return StreamSetsError{
				Type:    "JobFailureError",
				Message: msg,
			}
		}
	}

	// Unexpected status
	return StreamSetsError{
		Type:    "JobError",
		Message: fmt.Sprintf("Job completed with unexpected status %s, color %s", finalStatus.Status, finalStatus.Color),
	}
}

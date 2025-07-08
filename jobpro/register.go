package jobpro

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/serr"
)

const backendPort = "8080" // how to get to the backend container

type JobConfig struct {
	Id         string
	Name       string
	IsPeriodic bool
	// The schedule "*/30 * * * * *" translates to: Every 30 seconds.
	// Fields:
	// First */30 (Seconds field): This indicates that the job should run every 30 seconds. The * means "every possible value" for that field, and /30 means "every 30 units" within that range.
	// Second * (Minutes field): Every minute.
	// Third * (Hours field): Every hour.
	// Fourth * (Day of Month field): Every day of the month.
	// Fifth * (Month field): Every month.
	// Sixth * (Day of Week field): Every day of the week.
	Schedule   string
	Priority   int // Priority is not yet supported
	MaxRunTime int
	RetryCount int  // RetryCount is not yet supported
	AutoStart  bool // Whether to automatically start the job after creation (default: true)
	// We can use either the TriggerEndpoint or the JobFunction.
	TriggerEndpoint string
	JobFunction     func() error // no longer used
}

var jobCfgs = &jobConfigs{}

type jobConfigs struct {
	jobCfgs []JobConfig
	mu      sync.RWMutex
}

// RegisterJob adds a new job configuration to the app
// Example job configurations
//
//	RegisterJob(JobConfig{
//		Id:         "job1",
//		Name:       "Example Job 1",
//		IsPeriodic: true,
//		Schedule:   "0 0 * * * *", // Every hour
//		JobFunction: func() error { fmt.Println("doing work"); return nil }, // Replace with actual job function
//	})
//
//	RegisterJob(JobConfig{ // One-time job
//		Id:         "job2",
//		Name:       "Example Job 2",
//		IsPeriodic: false,
//		Schedule:   "<time.Time>",
//		JobFunction: func() error {  fmt.Println("doing work"); return nil }, // Replace with actual job function
//		MaxRunTime: 300,
//	})
func RegisterJob(cfg JobConfig) {
	jobCfgs.register(cfg)
}

func (jc *jobConfigs) register(cfg JobConfig) {
	jc.mu.Lock()
	defer jc.mu.Unlock()
	jc.jobCfgs = append(jc.jobCfgs, cfg)
}

func (jc *jobConfigs) getJobConfigs() []JobConfig {
	jc.mu.RLock()
	defer jc.mu.RUnlock()
	return jc.jobCfgs
}

type JobsResponse struct {
	Success bool        `json:"success"`
	Error   string      `json:"error"`
	Jobs    []JobConfig `json:"jobs"`
}

// FetchJobConfigs fetches job configurations from the specified endpoint
func FetchJobConfigs(endpoint string) ([]JobConfig, error) {
	resp, err := http.Get(endpoint)
	if err != nil {
		return nil, serr.Wrap(err, "Failed to fetch job configs")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, serr.New("Failed to fetch job configs", "status", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, serr.Wrap(err, "Failed to read job configs response")
	}

	var results JobsResponse
	logger.Debug("Jobs response from Backend:", string(body))

	if err := json.Unmarshal(body, &results); err != nil {
		return nil, serr.Wrap(err, "Failed to parse job configs")
	}

	if !results.Success {
		return nil, serr.New("Fetch job configs returned failure", "error", results.Error)
	}

	return results.Jobs, nil
}

func LoadJobs(mgr JobMgr) error {
	jcfgs := jobCfgs.getJobConfigs()
	for _, jcfg := range jcfgs {
		err := setupJob(mgr, jcfg)
		if err != nil {
			return serr.Wrap(err, "Failed to register job")
		}
	}
	return nil
}

// setupJob adds registered jobs into the manager
func setupJob(mgr JobMgr, jc JobConfig) error {
	job := NewScheduledJob(jc)

	jobID, err := mgr.SetupJob(job, jc.Schedule)
	if err != nil {
		return serr.Wrap(err, "failed to load job")
	}
	log.Printf("Load job: %v\n", jc)

	// Start job automatically if AutoStart is true
	if jc.AutoStart {
		if err := mgr.StartJob(jobID); err != nil {
			logger.LogErr(serr.Wrap(err, "Failed to start job"))
		}
	}

	return nil
}

// TriggerRemoteJob will trigger the job endpoint given by the JobConfig
func TriggerRemoteJob(jc JobConfig) error {
	if jc.TriggerEndpoint == "" {
		return serr.New("Trigger endpoint is empty")
	}

	endpoint := BackendURLWoPath() + jc.TriggerEndpoint

	resp, err := http.Get(endpoint)
	if err != nil {
		return serr.Wrap(err, "Failed to trigger remote job")
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return serr.F("Failed to trigger remote job. Bad status %s", resp.Status)
	}
	// We don't care about the response body

	return nil
}

func BackendURLWoPath() (urlWoPath string) {
	return fmt.Sprintf("http://localhost:%s", backendPort)
}

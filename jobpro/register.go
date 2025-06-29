package jobpro

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/serr"
)

type JobConfig struct {
	ID          string
	Name        string
	IsPeriodic  bool
	Schedule    string
	Priority    int // Priority is not yet supported
	MaxRunTime  int
	RetryCount  int  // RetryCount is not yet supported
	AutoStart   bool // Whether to automatically start the job after creation (default: true)
	JobFunction func() error
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
//		ID:         "job1",
//		Name:       "Example Job 1",
//		IsPeriodic: true,
//		Schedule:   "0 0 * * * *", // Every hour
//		JobFunction: func() error { fmt.Println("doing work"); return nil }, // Replace with actual job function
//	})
//
//	RegisterJob(JobConfig{ // One-time job
//		ID:         "job2",
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

	var configs []JobConfig
	if err := json.Unmarshal(body, &configs); err != nil {
		return nil, serr.Wrap(err, "Failed to parse job configs")
	}

	return configs, nil
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

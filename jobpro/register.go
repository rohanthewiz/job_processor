package jobpro

import (
	"log"
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
	RetryCount  int // RetryCount is not yet supported
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
//	RegisterJob(JobConfig{ // One-time job -- coming soon!
//		ID:         "job2",
//		Name:       "Example Job 2",
//		IsPeriodic: false,
//		Schedule:   "",
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
	var job Job
	if jc.IsPeriodic {
		job = NewPeriodicJob(jc)
	} else {
		// TODO - handle one-time jobs here
		return serr.New("One-time jobs are not supported yet")
	}

	periodicID, err := mgr.SetupJob(job, jc.Schedule)
	if err != nil {
		return serr.Wrap(err, "failed to create job")
	}
	log.Printf("Created job: %v\n", jc)

	// Start the periodic job
	if err := mgr.StartJob(periodicID); err != nil {
		logger.LogErr(serr.Wrap(err, "Failed to start periodic job"))
	}

	return nil
}

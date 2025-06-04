package jobpro

import (
	"context"
	"time"
)

// Stats represents runtime metrics for a job execution
type Stats struct {
	StartTimeUTC time.Time     // When the job started
	Duration     time.Duration // How long the job ran
	SuccessMsg   string        // Message on success
	ErrorTrace   error         // Error details, nil on success
}

// JobStatus represents the current state of a job
type JobStatus string

const (
	StatusCreated  JobStatus = "created"
	StatusRunning  JobStatus = "running"
	StatusPaused   JobStatus = "paused"
	StatusStopped  JobStatus = "stopped"
	StatusComplete JobStatus = "complete"
	StatusFailed   JobStatus = "failed"
)

// FreqType defines whether a job runs once or periodically
type FreqType string

const (
	OneTime  FreqType = "onetime"
	Periodic FreqType = "periodic"
)

// Job defines the interface that all jobs must implement
type Job interface {
	// Run executes the job and returns stats and any error
	Run(ctx context.Context) (Stats, error)
	// ID returns the unique identifier for this job
	ID() string
	// Name returns a human-readable name for this job
	Name() string
	// Type returns the schedule type (onetime or periodic)
	Type() FreqType
}

// JobDef contains metadata about a job
type JobDef struct {
	JobID     string   // Unique identifier
	JobName   string   // Human-readable name
	SchedType FreqType // Type of schedule
	// Cron expression for periodic jobs or time.Time to run for one-time jobs
	Schedule    string
	NextRunTime time.Time // When to next run this job
	Status      JobStatus // Current status
	CreatedAt   time.Time // When the job was created
	UpdatedAt   time.Time // When the job was last updated
}

// JobResult contains the outcome of a job execution
type JobResult struct {
	JobID      string        // ID of the job
	StartTime  time.Time     // When the job started
	EndTime    time.Time     // When the job completed
	Duration   time.Duration // How long it took
	Status     JobStatus     // Outcome status
	SuccessMsg string        // Success message if any
	ErrorMsg   string        // Error message if any
}

// JobStore defines the interface for job persistence
type JobStore interface {
	// SaveJob persists a job definition
	SaveJob(job JobDef) error
	// GetJob retrieves a job definition by ID
	GetJob(id string) (JobDef, error)
	// ListJobs retrieves all job definitions with optional filters
	ListJobs(status JobStatus, freqType FreqType) ([]JobDef, error)
	// UpdateJobStatus updates the status of a job
	UpdateJobStatus(id string, status JobStatus) error
	// UpdateNextRunTime updates when a job should next run
	UpdateNextRunTime(id string, nextRun time.Time) error
	// DeleteJob removes a job definition
	DeleteJob(id string) error
	// RecordJobResult stores the outcome of a job execution
	RecordJobResult(result JobResult) error
	// GetJobResults retrieves historical results for a job
	GetJobResults(jobID string, limit int) ([]JobResult, error)
	// GetJobRuns retrieves historical runs for all jobs
	GetJobRuns(limit int) ([]JobRun, error)
	// CleanupOldJobResults deletes job results older than the specified duration
	CleanupJobResults(olderThan time.Duration) error
	// Close closes the database connection
	Close() error
}

// JobMgr handles the lifecycle of jobs
type JobMgr interface {
	// CreateJob adds a new job to the system
	SetupJob(job Job, schedule string) (string, error)
	// StartJob begins execution of a job
	StartJob(id string) error
	// StopJob halts execution of a job
	StopJob(id string) error
	// PauseJob temporarily suspends a job
	PauseJob(id string) error
	// ResumeJob continues execution of a paused job
	ResumeJob(id string) error
	// DeleteJob removes a job from the system
	DeleteJob(id string) error
	// ListJobs lists all jobs
	ListJobs() ([]JobRun, error)
	// GetJobStatus retrieves the current status of a job
	GetJobStatus(id string) (JobStatus, error)
	// GetJobsUpdatedChan returns a channel that signals when jobs are updated in the manager.
	GetJobsUpdatedChan() <-chan any
	// Shutdown gracefully stops all running jobs
	Shutdown(timeout time.Duration) error
}

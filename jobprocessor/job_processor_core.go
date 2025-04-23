package jobprocessor

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

// ScheduleType defines whether a job runs once or periodically
type ScheduleType string

const (
	ScheduleOneTime  ScheduleType = "onetime"
	SchedulePeriodic ScheduleType = "periodic"
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
	Type() ScheduleType
}

// JobDefinition contains metadata about a job
type JobDefinition struct {
	JobID       string       // Unique identifier
	JobName     string       // Human-readable name
	SchedType   ScheduleType // Type of schedule
	Schedule    string       // Cron expression for periodic jobs
	NextRunTime time.Time    // When to next run this job
	Status      JobStatus    // Current status
	CreatedAt   time.Time    // When the job was created
	UpdatedAt   time.Time    // When the job was last updated
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
	SaveJob(job JobDefinition) error
	// GetJob retrieves a job definition by ID
	GetJob(id string) (JobDefinition, error)
	// ListJobs retrieves all job definitions with optional filters
	ListJobs(status JobStatus, schedType ScheduleType) ([]JobDefinition, error)
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
}

// JobManager handles the lifecycle of jobs
type JobManager interface {
	// CreateJob adds a new job to the system
	CreateJob(job Job, schedule string) (string, error)
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
	// GetJobStatus retrieves the current status of a job
	GetJobStatus(id string) (JobStatus, error)
	// Shutdown gracefully stops all running jobs
	Shutdown(timeout time.Duration) error
}

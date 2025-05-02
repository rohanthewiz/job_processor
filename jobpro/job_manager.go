package jobpro

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/rohanthewiz/serr"
)

// DefaultJobManager implements the JobMgr interface
type DefaultJobManager struct {
	store       JobStore
	cron        *cron.Cron
	jobs        map[string]Job
	cronEntries map[string]cron.EntryID
	runningJobs map[string]context.CancelFunc
	mu          sync.RWMutex
	wg          sync.WaitGroup
	results     chan JobResult
	shutdown    bool
}

// NewJobManager creates a new job manager with the provided store
func NewJobManager(store JobStore) *DefaultJobManager {
	cronParser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	cronScheduler := cron.New(cron.WithParser(cronParser), cron.WithChain())

	mgr := &DefaultJobManager{
		store:       store,
		cron:        cronScheduler,
		jobs:        make(map[string]Job),
		cronEntries: make(map[string]cron.EntryID),
		runningJobs: make(map[string]context.CancelFunc),
		results:     make(chan JobResult, 100), // Buffer for job results
	}

	// Start the results processor
	go mgr.processResults()

	// Start the cron scheduler
	cronScheduler.Start()

	return mgr
}

// processResults handles job completion results
func (m *DefaultJobManager) processResults() {
	for result := range m.results {
		// Store the job result
		if err := m.store.RecordJobResult(result); err != nil {
			log.Printf("Error recording job result for %s: %v", result.JobID, err)
		}

		// Update job status in store if job was successful or failed (not if stopped)
		if result.Status == StatusComplete || result.Status == StatusFailed {
			if err := m.store.UpdateJobStatus(result.JobID, result.Status); err != nil {
				log.Printf("Error updating job status for %s: %v", result.JobID, err)
			}

			// For periodic jobs that completed, update next run time if not already scheduled via cron
			// Are we to assume that if the job is already scheduled in cron, we cannot update the next run time?
			jobDef, err := m.store.GetJob(result.JobID)
			if err == nil && jobDef.SchedType == Periodic {
				m.mu.RLock()
				_, inCron := m.cronEntries[result.JobID]
				m.mu.RUnlock()

				if !inCron { // ~ this is a really weird case as why would the job have a schedule and not be in cron?
					// If not scheduled via cron (e.g., a manually triggered run), calculate next run
					scheduler, err := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom |
						cron.Month | cron.Dow).Parse(jobDef.Schedule)
					if err == nil {
						nextRun := scheduler.Next(time.Now())
						if err := m.store.UpdateNextRunTime(result.JobID, nextRun); err != nil {
							log.Printf("Error updating next run time for %s: %v", result.JobID, err)
						}
						// TODO what about actually executing the job again in cron?
					}
				}
			}
		}

		// Remove from running jobs map
		m.mu.Lock()
		delete(m.runningJobs, result.JobID)
		m.mu.Unlock()

		m.wg.Done() // Mark this job as done
	}
}

// CreateJob adds a new job to the system
func (m *DefaultJobManager) RegisterJob(job Job, schedule string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shutdown {
		return "", fmt.Errorf("job manager is shutting down")
	}

	jobID := job.ID()
	if jobID == "" {
		// Generate a UUID if the job doesn't provide one
		jobID = uuid.New().String()
	}

	// Check if job already exists
	if _, exists := m.jobs[jobID]; exists {
		return "", serr.F("job with ID %s already exists", jobID)
	}

	// Determine next run time
	var nextRun time.Time
	if job.Type() == Periodic && schedule != "" {
		// Parse the cron schedule
		scheduler, err := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow).Parse(schedule)
		if err != nil {
			return "", serr.F("unable to parse schedule: %w", err)
		}
		nextRun = scheduler.Next(time.Now())

	} else if job.Type() == OneTime {
		// For one-time jobs, use the current time if no specific time is provided
		if schedule == "" {
			nextRun = time.Now()
		} else {
			// Parse the schedule as a specific time
			parsedTime, err := time.Parse(time.RFC3339, schedule)
			if err != nil {
				return "", serr.F("invalid time format for one-time job (should be time.RFC3339): %w", err)
			}
			nextRun = parsedTime
		}
	}

	// Create job definition
	jobDef := JobDef{
		JobID:       jobID,
		JobName:     job.Name(),
		SchedType:   job.Type(),
		Schedule:    schedule,
		NextRunTime: nextRun,
		Status:      StatusCreated,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	// Save to store
	if err := m.store.SaveJob(jobDef); err != nil {
		return "", fmt.Errorf("failed to save job: %w", err)
	}

	// Add to in-memory map
	m.jobs[jobID] = job

	return jobID, nil
}

// StartJob begins execution of a job
func (m *DefaultJobManager) StartJob(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shutdown {
		return fmt.Errorf("job manager is shutting down")
	}

	job, exists := m.jobs[id]
	if !exists {
		// Try to load from store
		_, err := m.store.GetJob(id)
		if err != nil {
			return serr.Wrap(err, "job not found")
		}

		// We can't start a job that's not loaded into memory
		return serr.F("job %s exists in store but not in memory, cannot start", id)
	}

	// Check if already running
	if _, running := m.runningJobs[id]; running {
		return fmt.Errorf("job %s is already running", id)
	}

	// Update job status
	if err := m.store.UpdateJobStatus(id, StatusRunning); err != nil {
		return serr.Wrap(err, "failed to update job status")
	}

	// If it's a periodic job, schedule it with cron
	if job.Type() == Periodic {
		jobDef, err := m.store.GetJob(id)
		if err != nil {
			return serr.Wrap(err, "failed to get job details")
		}

		// Schedule with cron if not already scheduled
		if _, exists := m.cronEntries[id]; !exists {
			entryID, err := m.cron.AddFunc(jobDef.Schedule, func() {
				m.executeJob(id)
			})
			if err != nil {
				return serr.Wrap(err, "failed to schedule job")
			}
			m.cronEntries[id] = entryID
		}
	} else {
		// For one-time jobs, just execute it once
		go m.executeJob(id)
	}

	return nil
}

// executeJob runs a job and processes its result
func (m *DefaultJobManager) executeJob(id string) {
	m.mu.Lock()
	if m.shutdown {
		m.mu.Unlock()
		return
	}

	job, exists := m.jobs[id]
	if !exists {
		m.mu.Unlock()
		log.Printf("Job %s not found for execution", id)
		return
	}

	// Create a context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	m.runningJobs[id] = cancel
	m.wg.Add(1) // Track this running job
	m.mu.Unlock()

	// Execute the job
	startTime := time.Now().UTC()
	stats, err := job.Run(ctx)
	endTime := time.Now().UTC()
	duration := endTime.Sub(startTime)

	// Prepare result
	result := JobResult{
		JobID:      id,
		StartTime:  startTime,
		EndTime:    endTime,
		Duration:   duration,
		SuccessMsg: stats.SuccessMsg,
	}

	if err != nil {
		result.Status = StatusFailed
		result.ErrorMsg = err.Error()
	} else {
		result.Status = StatusComplete
	}

	// Send result for processing
	select {
	case m.results <- result:
		// Result queued for processing
	default: // How does default work in a select statement
		// Results channel is full, log and continue
		log.Printf("Results channel full, dropping result for job %s", id)
		m.wg.Done() // Still mark as done even if we couldn't queue the result
	}
}

// StopJob halts execution of a job
func (m *DefaultJobManager) StopJob(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if job exists
	if _, exists := m.jobs[id]; !exists {
		return fmt.Errorf("job %s not found", id)
	}

	// If it's a periodic job, remove from cron
	if entryID, exists := m.cronEntries[id]; exists {
		m.cron.Remove(entryID)
		delete(m.cronEntries, id)
	}

	// If it's running, cancel its context
	if cancel, running := m.runningJobs[id]; running {
		cancel() // This signals the job to stop
		delete(m.runningJobs, id)
		// Note: The job will complete and call wg.Done() when it processes the cancellation
	}

	// Update job status
	if err := m.store.UpdateJobStatus(id, StatusStopped); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	return nil
}

// PauseJob temporarily suspends a job
func (m *DefaultJobManager) PauseJob(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if job exists
	if _, exists := m.jobs[id]; !exists {
		return fmt.Errorf("job %s not found", id)
	}

	// If it's a periodic job, remove from cron but keep the entry ID
	if entryID, exists := m.cronEntries[id]; exists {
		m.cron.Remove(entryID)
		// We keep the entry in the m.cronEntries map to remember it was scheduled
	}

	// If it's running, we can't pause it mid-execution
	if _, running := m.runningJobs[id]; running {
		return fmt.Errorf("job %s is currently running and cannot be paused", id)
	}

	// Update job status
	if err := m.store.UpdateJobStatus(id, StatusPaused); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	return nil
}

// ResumeJob continues execution of a paused job
func (m *DefaultJobManager) ResumeJob(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if job exists
	job, exists := m.jobs[id]
	if !exists {
		return fmt.Errorf("job %s not found", id)
	}

	// Get current status
	jobDef, err := m.store.GetJob(id)
	if err != nil {
		return fmt.Errorf("failed to get job details: %w", err)
	}

	if jobDef.Status != StatusPaused {
		return fmt.Errorf("job %s is not paused", id)
	}

	// Update job status
	if err := m.store.UpdateJobStatus(id, StatusRunning); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// If it's a periodic job, reschedule with cron
	if job.Type() == Periodic {
		_, wasScheduled := m.cronEntries[id]
		if wasScheduled {
			entryID, err := m.cron.AddFunc(jobDef.Schedule, func() {
				m.executeJob(id)
			})
			if err != nil {
				return fmt.Errorf("failed to reschedule job: %w", err)
			}
			m.cronEntries[id] = entryID
		}
	}

	return nil
}

// DeleteJob removes a job from the system
func (m *DefaultJobManager) DeleteJob(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if job exists
	if _, exists := m.jobs[id]; !exists {
		return fmt.Errorf("job %s not found", id)
	}

	// If it's a periodic job, remove from cron
	if entryID, exists := m.cronEntries[id]; exists {
		m.cron.Remove(entryID)
		delete(m.cronEntries, id)
	}

	// If it's running, cancel it
	if cancel, running := m.runningJobs[id]; running {
		cancel()
		delete(m.runningJobs, id)
	}

	// Remove from maps
	delete(m.jobs, id)

	// Delete from store
	if err := m.store.DeleteJob(id); err != nil {
		return fmt.Errorf("failed to delete job from store: %w", err)
	}

	return nil
}

// GetJobStatus retrieves the current status of a job
func (m *DefaultJobManager) GetJobStatus(id string) (JobStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if job exists in memory
	if _, exists := m.jobs[id]; !exists {
		// Try to get from store
		jobDef, err := m.store.GetJob(id)
		if err != nil {
			return "", fmt.Errorf("job not found: %w", err)
		}
		return jobDef.Status, nil
	}

	// Get from store to ensure we have the latest status
	jobDef, err := m.store.GetJob(id)
	if err != nil {
		return "", fmt.Errorf("failed to get job status: %w", err)
	}

	// Override with running status if it's actually running
	if _, running := m.runningJobs[id]; running {
		return StatusRunning, nil
	}

	return jobDef.Status, nil
}

// LoadJobs loads jobs from store
func (m *DefaultJobManager) LoadJobs() error {
	jobs, err := m.store.ListJobs("", "")
	if err != nil {
		return fmt.Errorf("failed to list jobs: %w", err)
	}

	log.Printf("Loaded %d jobs from store", len(jobs))
	return nil
}

// ListJobs in the store
func (m *DefaultJobManager) ListJobs() (jobs []DisplayResults, err error) {
	jobs, err = m.store.GetDisplayResults(100)
	if err != nil {
		return jobs, serr.Wrap(err, "error listing jobs")
	}

	log.Printf("Loaded %d jobs results from store", len(jobs))

	return
}

// Shutdown gracefully stops all running jobs
func (m *DefaultJobManager) Shutdown(timeout time.Duration) error {
	m.mu.Lock()
	m.shutdown = true

	// Stop the cron scheduler
	cronContext := m.cron.Stop()

	// Cancel all running jobs
	for id, cancel := range m.runningJobs {
		log.Printf("Cancelling job %s during shutdown", id)
		cancel()
	}
	m.mu.Unlock()

	// Create a channel to signal timeout
	done := make(chan struct{})

	// Wait for all jobs to complete or timeout
	go func() {
		m.wg.Wait()
		close(done)
	}()

	// Wait for completion or timeout
	select {
	case <-done:
		log.Println("All jobs completed gracefully")
	case <-time.After(timeout):
		log.Println("Shutdown timed out, some jobs may not have completed")
	}

	// Close the results channel to stop the processor
	close(m.results)

	// Wait for cron context to be done
	<-cronContext.Done()

	return nil

}

package jobpro

import (
	"context"
	"fmt"
	"job_processor/util"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/serr"
)

// DefaultJobManager implements the JobMgr interface
type DefaultJobManager struct {
	store         JobStore
	cron          *cron.Cron
	jobs          map[string]Job                // keep track of active jobs
	cronEntries   map[string]cron.EntryID       // keep track of jobs scheduled with cron
	runningJobs   map[string]context.CancelFunc // keep track of running jobs and a cancel function to stop each
	scheduledJobs map[string]*time.Timer        // keep track of scheduled one-time jobs for cancellation
	mu            sync.RWMutex
	wg            sync.WaitGroup
	results       chan JobResult
	jobsUpdated   chan any // Channel to signal that there has been at least one job update
	shutdown      bool
}

// NewJobManager creates a new job manager with the provided store
func NewJobManager(store JobStore) *DefaultJobManager {
	cronParser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	cronScheduler := cron.New(cron.WithParser(cronParser), cron.WithChain())

	mgr := &DefaultJobManager{
		store:         store,
		cron:          cronScheduler,
		jobs:          make(map[string]Job),
		cronEntries:   make(map[string]cron.EntryID),
		runningJobs:   make(map[string]context.CancelFunc),
		scheduledJobs: make(map[string]*time.Timer),
		results:       make(chan JobResult, 256), // Buffer for job results - perhaps make this configurable
		jobsUpdated:   make(chan any, 1),
	}

	// Start the results processor
	go mgr.processResults()

	// Start the cron scheduler
	cronScheduler.Start()

	// Start the job results cleanup goroutine
	go func() {
		logger.Info("Launching cleanup goroutine")
		// Clean up job results older than one week
		oneWeek := 7 * 24 * time.Hour
		ticker := time.NewTicker(1 * time.Hour)

		defer ticker.Stop()

		// Run cleanup immediately on startup
		if err := store.CleanupJobResults(oneWeek); err != nil {
			logger.F("Error cleaning up old job results: %v", err)
		}

		// Then run every hour
		for {
			select {
			case <-ticker.C:
				if mgr.shutdown {
					return
				}
				if err := store.CleanupJobResults(oneWeek); err != nil {
					logger.F("Error cleaning up old job results: %v", err)
				}
			}
		}
	}()

	return mgr
}

// GetJobsUpdatedChan returns a channel that signals when jobs have been updated
func (m *DefaultJobManager) GetJobsUpdatedChan() <-chan any {
	return m.jobsUpdated
}

// processResults handles job completion results
func (m *DefaultJobManager) processResults() {
	for result := range m.results { // range over the results channel
		// Store the job result
		if err := m.store.RecordJobResult(result); err != nil {
			log.Printf("Error recording job result for %s: %v", result.JobID, err)
		}

		// Update job status in store if job was successful or failed (not if stopped)
		if result.Status == StatusComplete || result.Status == StatusFailed {
			fmt.Println("Job completed - updating job status in store")

			jobDef, err := m.store.GetJob(result.JobID)
			if err != nil {
				log.Printf("Error getting job definition for %s: %v", result.JobID, err)
				continue
			}
			isPeriodic := jobDef.SchedType == Periodic

			// Periodic jobs should not be updated here
			if !isPeriodic {
				if err := m.store.UpdateJobStatus(result.JobID, result.Status); err != nil {
					log.Printf("Error updating job status for %s: %v", result.JobID, err)
				}
			} /* we will only run periodic jobs with cron so ignore this block
				// else { // For periodic jobs that completed, update next run time if not already scheduled via cron
				m.mu.RLock()
				_, inCron := m.cronEntries[result.JobID]
				m.mu.RUnlock()

				if !inCron { // ~ why would the job have a schedule and not be in cron?
					// If not scheduled via cron (e.g., a manually triggered run), calculate next run
					scheduler, err := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom |
						cron.Month | cron.Dow).Parse(jobDef.Schedule)
					if err == nil {
						nextRun := scheduler.Next(time.Now())
						if err := m.store.UpdateNextRunTime(result.JobID, nextRun); err != nil {
							log.Printf("Error updating next run time for %s: %v", result.JobID, err)
						}
						// what about actually executing the job again in cron?
					}
				}
			}*/

			// Let the system know that jobs have been updated
			select {
			case m.jobsUpdated <- "updated":
				fmt.Println("Job update notification sent")
			default: // Non-blocking send to avoid blocking if no one is listening
				// If the channel is full, we don't want to block
			}
		}

		// Remove from running jobs map
		m.mu.Lock()
		delete(m.runningJobs, result.JobID)
		m.mu.Unlock()

		m.wg.Done() // Mark this job as done
	}
}

// SetupJob adds a new job to the system
func (m *DefaultJobManager) SetupJob(job Job, schedule string) (string, error) {
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
			// Parse the schedule using our flexible parser
			parsedTime, err := util.ParseSchedule(schedule)
			if err != nil {
				return "", serr.F("invalid time format for one-time job: %w", err)
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
// WIP
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

	jobDef, err := m.store.GetJob(id)
	if err != nil {
		return serr.Wrap(err, "failed to get job details")
	}

	// If it's a periodic job, schedule it with cron
	if job.Type() == Periodic {

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
	} else { // For one-time jobs
		// For manual start jobs (no schedule), execute immediately
		if jobDef.Schedule == "" {
			go m.executeJob(id)
		} else {
			// For scheduled one-time jobs, check if we need to schedule or execute
			if jobDef.NextRunTime.After(time.Now()) {
				// Update status to scheduled for future one-time jobs
				if err := m.store.UpdateJobStatus(id, StatusScheduled); err != nil {
					return serr.Wrap(err, "failed to update job status to scheduled")
				}

				// Schedule the job in a goroutine and store the timer
				go func() {
					timer := RunAt(jobDef.NextRunTime, func() {
						// Remove the timer reference when the job starts
						m.mu.Lock()
						delete(m.scheduledJobs, id)
						m.mu.Unlock()

						m.executeJob(id)
					})

					// Store the timer reference
					m.mu.Lock()
					m.scheduledJobs[id] = timer
					m.mu.Unlock()
				}()
			} else {
				// If the scheduled time has passed, execute immediately
				go m.executeJob(id)
			}
		}
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

	startTime := time.Now().UTC()

	// EXECUTE the job
	stats, err := job.Run(ctx) // DoIt
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
	default:
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
	job, exists := m.jobs[id]
	if !exists {
		return fmt.Errorf("job %s not found", id)
	}

	// Get current job status to determine the appropriate action
	jobDef, err := m.store.GetJob(id)
	if err != nil {
		return fmt.Errorf("failed to get job status: %w", err)
	}

	var finalStatus JobStatus

	// If it's a periodic job, remove from cron
	if entryID, exists := m.cronEntries[id]; exists {
		m.cron.Remove(entryID)
		delete(m.cronEntries, id)
		finalStatus = StatusStopped
	}

	// If it's running, cancel its context
	if cancel, running := m.runningJobs[id]; running {
		cancel() // This signals the job to stop
		delete(m.runningJobs, id)
		finalStatus = StatusStopped
		// Note: The job will complete and call wg.Done() when it processes the cancellation
	} else if job.Type() == OneTime {
		// For one-time jobs that are scheduled but not running
		if timer, scheduled := m.scheduledJobs[id]; scheduled {
			// Cancel the scheduled execution
			timer.Stop()
			delete(m.scheduledJobs, id)
			finalStatus = StatusCancelled
		} else if jobDef.Status == StatusScheduled {
			// Job was scheduled but timer already fired or was removed
			finalStatus = StatusCancelled
		} else {
			finalStatus = StatusStopped
		}
	} else {
		finalStatus = StatusStopped
	}

	// Update job status
	if err := m.store.UpdateJobStatus(id, finalStatus); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// Send notification that job has stopped
	select {
	case m.jobsUpdated <- "updated":
		fmt.Println("Job update (stopped/cancelled) notification sent")
	default: // Non-blocking send to avoid blocking if no one is listening
		// If the channel is full, we don't want to block
	}

	return nil
}

// PauseJob temporarily suspends a job
func (m *DefaultJobManager) PauseJob(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if job exists
	job, exists := m.jobs[id]
	if !exists {
		return fmt.Errorf("job %s not found", id)
	}

	// If it's a periodic job, remove from cron but keep it in m.cronEntries
	if entryID, exists := m.cronEntries[id]; exists {
		m.cron.Remove(entryID)
		// We keep the entry in the m.cronEntries map to remember it was scheduled
	}

	if job.Type() != Periodic {
		// If it's running, we can't pause it mid-execution
		if _, running := m.runningJobs[id]; running {
			return fmt.Errorf("job %s is currently running and cannot be paused", id)
		}
	}

	// Update job status
	if err := m.store.UpdateJobStatus(id, StatusPaused); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// Let the system know that jobs have been updated
	select {
	case m.jobsUpdated <- "updated":
		fmt.Println("Job update (paused) notification sent")
	default: // Non-blocking send to avoid blocking if no one is listening
		// If the channel is full, we don't want to block
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

	// Let the system know that jobs have been updated
	select {
	case m.jobsUpdated <- "updated":
		fmt.Println("Job update (resume) notification sent")
	default: // Non-blocking send to avoid blocking if no one is listening
		// If the channel is full, we don't want to block
	}

	return nil
}

// RescheduleJob changes the execution time of a scheduled one-time job
func (m *DefaultJobManager) RescheduleJob(id string, newSchedule string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if job exists
	job, exists := m.jobs[id]
	if !exists {
		return fmt.Errorf("job %s not found", id)
	}

	// Only one-time jobs can be rescheduled
	if job.Type() != OneTime {
		return fmt.Errorf("only one-time jobs can be rescheduled")
	}

	// Get current job status
	jobDef, err := m.store.GetJob(id)
	if err != nil {
		return fmt.Errorf("failed to get job details: %w", err)
	}

	// Can only reschedule jobs that are scheduled or created
	if jobDef.Status != StatusScheduled && jobDef.Status != StatusCreated {
		return fmt.Errorf("job %s cannot be rescheduled in status %s", id, jobDef.Status)
	}

	// Parse the new schedule
	newTime, err := util.ParseSchedule(newSchedule)
	if err != nil {
		return fmt.Errorf("invalid time format: %w", err)
	}

	// Cancel existing timer if exists
	if timer, exists := m.scheduledJobs[id]; exists {
		timer.Stop()
		delete(m.scheduledJobs, id)
	}

	// Update the job in the store
	jobDef.Schedule = newSchedule
	jobDef.NextRunTime = newTime
	jobDef.UpdatedAt = time.Now().UTC()
	if err := m.store.SaveJob(jobDef); err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	// Update next run time
	if err := m.store.UpdateNextRunTime(id, newTime); err != nil {
		return fmt.Errorf("failed to update next run time: %w", err)
	}

	// Reschedule the job with new time
	go func() {
		timer := RunAt(newTime, func() {
			// Remove the timer reference when the job starts
			m.mu.Lock()
			delete(m.scheduledJobs, id)
			m.mu.Unlock()

			m.executeJob(id)
		})

		// Store the new timer reference
		m.mu.Lock()
		m.scheduledJobs[id] = timer
		m.mu.Unlock()
	}()

	// Send notification
	select {
	case m.jobsUpdated <- "updated":
		fmt.Println("Job update (rescheduled) notification sent")
	default:
		// If the channel is full, we don't want to block
	}

	return nil
}

// TriggerJobNow immediately executes a job regardless of its schedule
func (m *DefaultJobManager) TriggerJobNow(id string) error {
	m.mu.Lock()

	// Check if job exists
	_, exists := m.jobs[id]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("job %s not found", id)
	}

	// Check if job manager is shutting down
	if m.shutdown {
		m.mu.Unlock()
		return fmt.Errorf("job manager is shutting down")
	}

	m.mu.Unlock()

	// Execute the job in a goroutine
	go m.executeJob(id)

	// Let the system know that jobs have been updated
	select {
	case m.jobsUpdated <- "updated":
		fmt.Println("Job update (triggered) notification sent")
	default: // Non-blocking send to avoid blocking if no one is listening
		// If the channel is full, we don't want to block
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

	// If it's a scheduled one-time job, cancel the timer
	if timer, scheduled := m.scheduledJobs[id]; scheduled {
		timer.Stop()
		delete(m.scheduledJobs, id)
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
func (m *DefaultJobManager) ListJobs() (jobs []JobRun, err error) {
	jobs, err = m.store.GetJobRuns(100)
	if err != nil {
		return jobs, serr.Wrap(err, "error listing jobs")
	}

	log.Printf("Loaded %d jobs results from store", len(jobs))

	return
}

// ListJobsWithPagination returns jobs with limited results per job and result counts
func (m *DefaultJobManager) ListJobsWithPagination(resultsPerJob int) (jobs []JobRun, resultCounts map[string]int, err error) {
	jobs, resultCounts, err = m.store.GetJobRunsWithPagination(resultsPerJob)
	if err != nil {
		return nil, nil, serr.Wrap(err, "error listing jobs with pagination")
	}

	log.Printf("Loaded %d jobs with pagination from store", len(jobs))

	return
}

// GetJobResultsPaginated returns paginated results for a specific job
func (m *DefaultJobManager) GetJobResultsPaginated(jobID string, offset, limit int) ([]JobResult, int, error) {
	return m.store.GetJobResultsPaginated(jobID, offset, limit)
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

	err := m.store.Close()
	if err != nil {
		return serr.Wrap(err, "error closing job store")
	}

	return nil
}

// GetJobHistory retrieves the execution history for a specific job
func (m *DefaultJobManager) GetJobHistory(jobID string, limit int) ([]JobResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if job exists
	if _, exists := m.jobs[jobID]; !exists {
		return nil, fmt.Errorf("job %s not found", jobID)
	}

	// Get results from the store
	results, err := m.store.GetJobResults(jobID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get job history: %w", err)
	}

	return results, nil
}

package jobpro

import (
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

// TestManualStartJob tests that jobs with AutoStart=false and no schedule don't auto-start
func TestManualStartJob(t *testing.T) {
	// Create a new job manager with in-memory store
	store, err := NewDuckDBStore(":memory:")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	mgr := NewJobManager(store)
	defer mgr.Shutdown(5 * time.Second)

	// Track job executions
	var manualJobRuns int32
	var autoStartJobRuns int32
	var scheduledJobRuns int32

	tests := []struct {
		name          string
		config        JobConfig
		expectAutoRun bool
		runCount      *int32
	}{
		{
			name: "Manual start job (AutoStart=false, no schedule)",
			config: JobConfig{
				Id:         "manual1",
				Name:       "Manual Job",
				IsPeriodic: false,
				Schedule:   "",
				AutoStart:  false,
				JobFunction: func() error {
					atomic.AddInt32(&manualJobRuns, 1)
					return nil
				},
			},
			expectAutoRun: false,
			runCount:      &manualJobRuns,
		},
		{
			name: "Auto-start job (AutoStart=true, no schedule)",
			config: JobConfig{
				Id:         "auto1",
				Name:       "Auto Job",
				IsPeriodic: false,
				Schedule:   "",
				AutoStart:  true,
				JobFunction: func() error {
					atomic.AddInt32(&autoStartJobRuns, 1)
					return nil
				},
			},
			expectAutoRun: true,
			runCount:      &autoStartJobRuns,
		},
		{
			name: "Scheduled job (AutoStart=true, with schedule)",
			config: JobConfig{
				Id:         "scheduled1",
				Name:       "Scheduled Job",
				IsPeriodic: false,
				Schedule:   "in 100ms",
				AutoStart:  true,
				JobFunction: func() error {
					atomic.AddInt32(&scheduledJobRuns, 1)
					return nil
				},
			},
			expectAutoRun: true,
			runCount:      &scheduledJobRuns,
		},
	}

	// Setup and register all jobs
	for _, tt := range tests {
		t.Run(tt.name+"_setup", func(t *testing.T) {
			// Clear the global job configs before registering
			jobCfgs = &jobConfigs{jobCfgs: []JobConfig{}}

			// Register the job
			RegisterJob(tt.config)

			// Setup the job in the manager
			err := setupJob(mgr, tt.config)
			if err != nil {
				t.Errorf("Failed to setup job %s: %v", tt.config.Id, err)
			}
		})
	}

	// Wait a bit to see if jobs auto-start
	time.Sleep(200 * time.Millisecond)

	// Check initial run counts
	t.Run("Check auto-start behavior", func(t *testing.T) {
		if atomic.LoadInt32(&manualJobRuns) != 0 {
			t.Errorf("Manual job should not have run automatically, but ran %d times", manualJobRuns)
		}

		if atomic.LoadInt32(&autoStartJobRuns) == 0 {
			t.Errorf("Auto-start job should have run automatically, but didn't run")
		}

		if atomic.LoadInt32(&scheduledJobRuns) == 0 {
			t.Errorf("Scheduled job should have run after 100ms, but didn't run")
		}
	})

	// Test manual start functionality
	t.Run("Test manual start", func(t *testing.T) {
		// Get current run count
		currentRuns := atomic.LoadInt32(&manualJobRuns)

		// Manually start the job
		err := mgr.StartJob("manual1")
		if err != nil {
			t.Fatalf("Failed to manually start job: %v", err)
		}

		// Wait for job to execute
		time.Sleep(100 * time.Millisecond)

		// Check that it ran exactly once
		newRuns := atomic.LoadInt32(&manualJobRuns)
		if newRuns != currentRuns+1 {
			t.Errorf("Manual job should have run exactly once after StartJob, expected %d runs, got %d", currentRuns+1, newRuns)
		}
	})

	// Test that manual jobs can be triggered multiple times
	t.Run("Test manual trigger multiple times", func(t *testing.T) {
		currentRuns := atomic.LoadInt32(&manualJobRuns)

		// Trigger the job multiple times
		for i := 0; i < 3; i++ {
			err := mgr.TriggerJobNow("manual1")
			if err != nil {
				t.Fatalf("Failed to trigger job: %v", err)
			}
			time.Sleep(50 * time.Millisecond)
		}

		// Check that it ran 3 more times
		newRuns := atomic.LoadInt32(&manualJobRuns)
		if newRuns != currentRuns+3 {
			t.Errorf("Manual job should have run 3 more times, expected %d total runs, got %d", currentRuns+3, newRuns)
		}
	})
}

// TestManualJobWithSchedule tests that manual jobs with schedules behave correctly
func TestManualJobWithSchedule(t *testing.T) {
	store, err := NewDuckDBStore(":memory:")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	mgr := NewJobManager(store)
	defer mgr.Shutdown(5 * time.Second)

	var jobRuns int32

	// Clear the global job configs
	jobCfgs = &jobConfigs{jobCfgs: []JobConfig{}}

	// Register a manual job with a future schedule
	config := JobConfig{
		Id:         "manual_scheduled",
		Name:       "Manual Scheduled Job",
		IsPeriodic: false,
		Schedule:   "in 500ms",
		AutoStart:  false,
		JobFunction: func() error {
			atomic.AddInt32(&jobRuns, 1)
			fmt.Printf("Manual scheduled job executed at %v\n", time.Now())
			return nil
		},
	}

	RegisterJob(config)
	if err := setupJob(mgr, config); err != nil {
		t.Fatalf("Failed to setup job: %v", err)
	}

	// Job should not run immediately
	time.Sleep(100 * time.Millisecond)
	if atomic.LoadInt32(&jobRuns) != 0 {
		t.Errorf("Manual job with schedule should not run without being started")
	}

	// Start the job - it should wait for the scheduled time
	err = mgr.StartJob("manual_scheduled")
	if err != nil {
		t.Fatalf("Failed to start job: %v", err)
	}

	// Job should not run immediately even after starting
	time.Sleep(100 * time.Millisecond)
	if atomic.LoadInt32(&jobRuns) != 0 {
		t.Errorf("Manual job should wait for scheduled time, not run immediately")
	}

	// Wait for the scheduled time
	time.Sleep(500 * time.Millisecond)
	if atomic.LoadInt32(&jobRuns) != 1 {
		t.Errorf("Manual job should have run at scheduled time, expected 1 run, got %d", jobRuns)
	}
}

// TestJobStatusTransitions tests the status transitions for manual jobs
func TestJobStatusTransitions(t *testing.T) {
	store, err := NewDuckDBStore(":memory:")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	mgr := NewJobManager(store)
	defer mgr.Shutdown(5 * time.Second)

	// Clear the global job configs
	jobCfgs = &jobConfigs{jobCfgs: []JobConfig{}}

	// Register a manual job
	config := JobConfig{
		Id:         "status_test",
		Name:       "Status Test Job",
		IsPeriodic: false,
		Schedule:   "",
		AutoStart:  false,
		JobFunction: func() error {
			time.Sleep(200 * time.Millisecond) // Simulate some work
			return nil
		},
	}

	RegisterJob(config)
	if err := setupJob(mgr, config); err != nil {
		t.Fatalf("Failed to setup job: %v", err)
	}

	// Check initial status
	status, err := mgr.GetJobStatus("status_test")
	if err != nil {
		t.Fatalf("Failed to get job status: %v", err)
	}
	if status != StatusCreated {
		t.Errorf("Expected initial status to be %s, got %s", StatusCreated, status)
	}

	// Start the job
	err = mgr.StartJob("status_test")
	if err != nil {
		t.Fatalf("Failed to start job: %v", err)
	}

	// Check running status
	time.Sleep(50 * time.Millisecond)
	status, err = mgr.GetJobStatus("status_test")
	if err != nil {
		t.Fatalf("Failed to get job status: %v", err)
	}
	if status != StatusRunning {
		t.Errorf("Expected status to be %s after start, got %s", StatusRunning, status)
	}

	// Wait for completion
	time.Sleep(300 * time.Millisecond)
	status, err = mgr.GetJobStatus("status_test")
	if err != nil {
		t.Fatalf("Failed to get job status: %v", err)
	}
	if status != StatusComplete {
		t.Errorf("Expected status to be %s after completion, got %s", StatusComplete, status)
	}
}

// TestManualJobPersistence tests that manual jobs remain in the correct state after restart
func TestManualJobPersistence(t *testing.T) {
	// Use a temp file for the database to test persistence
	dbPath := "/tmp/test_job_persistence.db"
	defer func() {
		// Clean up the test database
		_ = os.Remove(dbPath)
	}()

	// First, create a job manager and register a manual job
	store1, err := NewDuckDBStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	mgr1 := NewJobManager(store1)

	// Clear the global job configs
	jobCfgs = &jobConfigs{jobCfgs: []JobConfig{}}

	var jobRuns int32

	// Register a manual job
	config := JobConfig{
		Id:         "persist_test",
		Name:       "Persistence Test Job",
		IsPeriodic: false,
		Schedule:   "",
		AutoStart:  false,
		JobFunction: func() error {
			atomic.AddInt32(&jobRuns, 1)
			return nil
		},
	}

	RegisterJob(config)
	if err := setupJob(mgr1, config); err != nil {
		t.Fatalf("Failed to setup job: %v", err)
	}

	// Verify job was created but not started
	status, err := mgr1.GetJobStatus("persist_test")
	if err != nil {
		t.Fatalf("Failed to get job status: %v", err)
	}
	if status != StatusCreated {
		t.Errorf("Expected status to be %s, got %s", StatusCreated, status)
	}

	// Shutdown the first manager
	if err := mgr1.Shutdown(5 * time.Second); err != nil {
		t.Fatalf("Failed to shutdown manager: %v", err)
	}
	store1.Close()

	// Simulate system restart - create new manager with same database
	store2, err := NewDuckDBStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store after restart: %v", err)
	}
	defer store2.Close()

	mgr2 := NewJobManager(store2)
	defer mgr2.Shutdown(5 * time.Second)

	// Load jobs from the store
	jobs, err := store2.ListJobs("", "")
	if err != nil {
		t.Fatalf("Failed to list jobs: %v", err)
	}

	if len(jobs) != 1 {
		t.Fatalf("Expected 1 job in store, got %d", len(jobs))
	}

	// Check that the job still exists and has the correct status
	jobDef := jobs[0]
	if jobDef.JobID != "persist_test" {
		t.Errorf("Expected job Id 'persist_test', got %s", jobDef.JobID)
	}
	if jobDef.Status != StatusCreated {
		t.Errorf("Expected job status to be %s after restart, got %s", StatusCreated, jobDef.Status)
	}

	// Verify that the job didn't auto-start after restart
	time.Sleep(100 * time.Millisecond)
	if atomic.LoadInt32(&jobRuns) != 0 {
		t.Errorf("Manual job should not have run after restart, but ran %d times", jobRuns)
	}
}

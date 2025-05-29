package jobpro

import (
	"testing"
	"time"
)

func TestDuckDBStore_CleanupJobResults(t *testing.T) {
	// Setup in-memory database
	store, err := NewDuckDBStore("")
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Create a job to reference
	job := JobDef{
		JobID:       "test-job",
		JobName:     "Test Job",
		SchedType:   Periodic,
		Status:      StatusCreated,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
		NextRunTime: time.Now().UTC(),
	}
	if err := store.SaveJob(job); err != nil {
		t.Fatalf("Failed to save job: %v", err)
	}

	// Insert some old results (2 days old)
	oldTime := time.Now().Add(-48 * time.Hour)
	oldResult := JobResult{
		JobID:     "test-job",
		StartTime: oldTime,
		EndTime:   oldTime.Add(10 * time.Second),
		Duration:  10 * time.Second,
		Status:    StatusComplete,
	}
	if err := store.RecordJobResult(oldResult); err != nil {
		t.Fatalf("Failed to record old job result: %v", err)
	}

	// Insert some recent results (1 hour old)
	recentTime := time.Now().Add(-1 * time.Hour)
	recentResult := JobResult{
		JobID:     "test-job",
		StartTime: recentTime,
		EndTime:   recentTime.Add(10 * time.Second),
		Duration:  10 * time.Second,
		Status:    StatusComplete,
	}
	if err := store.RecordJobResult(recentResult); err != nil {
		t.Fatalf("Failed to record recent job result: %v", err)
	}

	// Verify both records exist
	results, err := store.GetJobResults("test-job", 10)
	if err != nil {
		t.Fatalf("Failed to get results: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	// Cleanup results older than 24 hours
	err = store.CleanupJobResults(24 * time.Hour)
	if err != nil {
		t.Fatalf("Failed to cleanup results: %v", err)
	}

	// Verify only recent record remains
	results, err = store.GetJobResults("test-job", 10)
	if err != nil {
		t.Fatalf("Failed to get results after cleanup: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected 1 result after cleanup, got %d", len(results))
	}

	// The remaining result should be the recent one
	if !results[0].EndTime.After(oldResult.EndTime) {
		t.Errorf("Expected only recent result to remain, but old result was found")
	}
}

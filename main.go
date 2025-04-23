package main

import (
	"fmt"
	"job_processor/jobprocessor"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/serr"
)

// Global shutdown flag
var (
	isShutdown bool
	mu         sync.RWMutex
)

// checkShutdown checks if shutdown flag is set
func checkShutdown() bool {
	mu.RLock()
	defer mu.RUnlock()
	return isShutdown
}

// setShutdown sets the shutdown flag
func setShutdown() {
	mu.Lock()
	isShutdown = true
	mu.Unlock()
	os.Setenv("SHUTDOWN", "true")
}

func main() {
	log.Println("Starting job processor")

	// Initialize DuckDB store
	dbPath := os.Getenv("DUCKDB_PATH")
	if dbPath == "" {
		dbPath = "jobs.duckdb"
	}

	store, err := jobprocessor.NewDuckDBStore(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize DuckDB store: %v", err)
	}
	defer store.Close()

	// Initialize job manager
	manager := jobprocessor.NewJobManager(store)

	// Setup shutdown signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create a done channel to signal when shutdown is complete
	done := make(chan struct{})

	// Handle shutdown signal in a goroutine
	go func() {
		sig := <-sigChan
		log.Printf("Received shutdown signal: %v", sig)
		setShutdown()

		// Give manager time to shutdown gracefully
		shutdownTimeout := 30 * time.Second
		log.Printf("Shutting down job manager with timeout of %v", shutdownTimeout)
		if err := manager.Shutdown(shutdownTimeout); err != nil {
			log.Printf("Error during shutdown: %v", err)
		}

		close(done)
	}()

	// Register some example jobs
	if err := registerExampleJobs(manager); err != nil {
		logger.LogErr(serr.F("Failed to register example jobs: %v", err))
	}

	// Block until done signal
	<-done
	log.Println("Job processor exited")
}

// registerExampleJobs adds some example jobs to the manager
func registerExampleJobs(manager jobprocessor.JobManager) error {
	// Create a one-time job that runs for 5 seconds with 90% success probability
	oneTimeJob := jobprocessor.NewDummyJob(
		"one-time-1",
		"One-time Test Job",
		jobprocessor.ScheduleOneTime,
		5*time.Second,
		0.9,
	)

	oneTimeID, err := manager.CreateJob(oneTimeJob, time.Now().Add(10*time.Second).Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("failed to create one-time job: %w", err)
	}
	log.Printf("Created one-time job with ID: %s", oneTimeID)

	// Start the one-time job
	if err := manager.StartJob(oneTimeID); err != nil {
		log.Printf("Failed to start one-time job: %v", err)
	}

	// Create a periodic job that runs every minute
	periodicJob := jobprocessor.NewLoggingJob(
		"periodic-1",
		"Periodic Logging Job",
		jobprocessor.SchedulePeriodic,
		30*time.Second,
		[]time.Duration{1 * time.Second, 5 * time.Second, 10 * time.Second},
	)

	periodicID, err := manager.CreateJob(periodicJob, "0 */1 * * * *") // Run every minute
	if err != nil {
		return serr.F("failed to create periodic job: %w", err)
	}
	log.Printf("Created periodic job with ID: %s", periodicID)

	// Start the periodic job
	if err := manager.StartJob(periodicID); err != nil {
		logger.LogErr(serr.Wrap(err, "Failed to start periodic job"))
	}

	return nil
}

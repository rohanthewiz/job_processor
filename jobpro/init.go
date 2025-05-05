package jobpro

import (
	"job_processor/shutdown"
	"log"
	"os"
	"time"

	"github.com/rohanthewiz/logger"
)

// Init initializes the job processor with a DuckDB store and a job manager
//
//	If dbPath is empty, the store will be in-memory
func Init(dbFilePath string) (manager *DefaultJobManager) {
	log.Println("Starting job processor")

	// Initialize DuckDB store
	if dbFilePath == "" {
		dbFilePath = os.Getenv("DB_FIlE_PATH")
	}

	// If dbFilePath is still empty, use in-memory database
	store, err := NewDuckDBStore(dbFilePath)
	if err != nil {
		logger.LogErr(err, "Failed to initialize DuckDB store")
		os.Exit(1)
	}

	// Initialize job manager
	jobMgr := NewJobManager(store)

	shutdown.RegisterHook(func(gracePeriod time.Duration) error {
		err := jobMgr.Shutdown(gracePeriod)
		if err != nil {
			logger.LogErr(err, "Error during job manager shutdown")
		} else {
			logger.Info("Job manager shutdown")
		}
		return err
	})

	return jobMgr
}

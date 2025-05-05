package jobpro

import (
	"log"
	"os"

	"github.com/rohanthewiz/logger"
)

// Init initializes the job processor with a DuckDB store and a job manager
//
//	If dbPath is empty, the store will be in-memory
func Init(dbFilePath string) (manager *DefaultJobManager, store *DuckDBStore) {
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
	return NewJobManager(store), store
}

package fs

import (
	"log"
	"os"
	"sync"
	"time"
)

// logger will record events for the recorder to file
var (
	logger  *log.Logger
	once    sync.Once
	outfile string
)

// GetEventFile gets an event file
func GetEventFile(outdir string) string {

	// If output directory not provided, default to pwd
	if outdir == "" {
		dir, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		outdir = dir
	}

	f, err := os.CreateTemp(outdir, "fs-record.log")
	if err != nil {
		panic(err)
	}

	// Get the full path of the temporary file
	tempFilePath := f.Name()
	defer f.Close()
	return tempFilePath
}

// logEvent logs the event to file with a unix nano timeseconds
func logEvent(event, path string) {
	// Cut out early if we didn't define a log file
	if outfile == "" {
		return
	}
	logger := getLogger()
	logger.Printf("%d %-*s %s\n", time.Now().UnixNano(), 10, event, path)
}

func getLogger() *log.Logger {
	once.Do(func() {
		file, err := os.OpenFile(outfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatal(err)
		}
		logger = log.New(file, "", log.LstdFlags|log.Lshortfile)
	})
	log.SetFlags(0)
	return logger
}

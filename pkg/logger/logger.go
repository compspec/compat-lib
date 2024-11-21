package logger

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// logger will record events for the recorder to file
var (
	logger *log.Logger
	once   sync.Once

	// Default output file available for setting externally
	Outfile string
)

// SetOutfile can be called from an external class to set
// the global variable.
func SetOutfile(outfile string) {
	Outfile = outfile
}

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
// and some arbitrary number of string arguments
func LogEvent(args ...string) {
	event, args := args[0], args[1:]

	// Cut out early if we didn't define a log file
	if Outfile == "" {
		return
	}
	prefix := fmt.Sprintf("%d %-*s", time.Now().UnixNano(), 10, event)
	for _, arg := range args {
		prefix += fmt.Sprintf("%-*s", 10, arg)
	}
	logger := getLogger()
	logger.Println(prefix)
}

func getLogger() *log.Logger {
	once.Do(func() {
		file, err := os.OpenFile(Outfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatal(err)
		}
		logger = log.New(file, "", log.LstdFlags|log.Lshortfile)
	})
	log.SetFlags(0)
	return logger
}

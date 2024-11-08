package fs

import (
	"log"
	"os"
	"sync"
)

// logger will record events for the recorder to file
var (
	logger  *log.Logger
	once    sync.Once
	outfile string
)

func GetEventFile() string {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	f, err := os.CreateTemp(dir, "fs-record.log")
	if err != nil {
		panic(err)
	}

	// Get the full path of the temporary file
	tempFilePath := f.Name()
	defer f.Close()
	return tempFilePath
}

func logEvent(event, path string) {
	// Cut out early if we didn't define a log file
	if outfile == "" {
		return
	}
	logger := getLogger()
	logger.Printf("%s %s\n", event, path)
}

func getLogger() *log.Logger {
	once.Do(func() {
		file, err := os.OpenFile(outfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatal(err)
		}
		logger = log.New(file, "", log.LstdFlags|log.Lshortfile)
	})
	return logger
}

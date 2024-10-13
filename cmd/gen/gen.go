package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/compspec/compat-lib/pkg/fs"
	"github.com/compspec/compat-lib/pkg/generate"
)

func main() {
	fmt.Println("⭐️ Compatibility Library Generator (clib-gen)")
	binary := flag.String("binary", "", "Binary to trace")

	flag.Parse()
	path := *binary

	fmt.Printf("Preparing to find shared libraries needed for %s\n", path)
	paths, err := generate.FindSharedLibs(path)
	if err != nil {
		log.Panicf("Error finding shared libraries for %s: %x", path, err)
	}
	for _, path := range paths {
		fmt.Println(path)
	}

	// Generate the fusefs server
	compatFS, err := fs.NewCompatFS()
	if err != nil {
		log.Panicf("Cannot generate fuse server: %x", err)
	}

	// Removes mount point directory when done
	defer compatFS.Cleanup()

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		compatFS.Server.Unmount()
	}()
	compatFS.Server.Wait()

}

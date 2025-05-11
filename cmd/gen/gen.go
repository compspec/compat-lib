package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/compspec/compat-lib/pkg/compat"
	"github.com/compspec/compat-lib/pkg/generate"
)

func main() {
	fmt.Println("⭐️ Compatibility Library Generator (clib-gen)")
	outfile := flag.String("out", "", "Output file path for artifact")

	flag.Parse()
	args := flag.Args()
	outPath := *outfile

	if len(args) == 0 {
		log.Fatal("Please provide the binary you want to generate an artifact for.")
	}

	// Get the full path of the command
	path := args[0]
	path, err := filepath.Abs(path)
	if err != nil {
		fmt.Println(err)
		log.Fatal("Error getting full path")
	}

	// This is where we should look them up in some cache
	fmt.Printf("Preparing to find shared libraries needed for %s\n", args)
	libs, err := generate.FindSharedLibs(path)
	if err != nil {
		fmt.Println(err)
		log.Fatalf("Error finding shared libraries for %s", path)
	}

	// Generate the artifact
	spec := compat.GenerateLibraryArtifact(path, libs)
	out, err := spec.ToJson()
	if err != nil {
		fmt.Println(err)
		log.Fatalf("Issue serializing spec to json")
	}
	if outPath == "" {
		fmt.Println(string(out))
	} else {
		fmt.Printf("🗒️ Writing to file %s\n", outPath)
		err = os.WriteFile(outPath, out, 0644)
		if err != nil {
			fmt.Println(err)
			log.Fatalf("Issue writing to output file")
		}
	}
}

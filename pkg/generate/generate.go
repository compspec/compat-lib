package generate

import (
	"debug/elf"
	"fmt"
	"log"
	"os"

	"github.com/u-root/u-root/pkg/ldd"
)

// FindSharedLibs (soname) from paths from ldd.FList
func FindSharedLibs(path string) ([]string, error) {

	libs, err := ldd.FList(path)
	if err != nil {
		log.Panicf("Issue with listing symlinks for %s\n", err)
	}
	// FList follows symlink paths
	// Do this and get the soname
	sonames := []string{}
	for _, path := range libs {
		soname, err := readSoname(path)
		if err != nil {
			fmt.Printf("Warning, cannot read soname of %s\n", path)
			continue
		}
		sonames = append(sonames, soname)
	}
	return sonames, err
}

// ReadSoname from an ELF file
func readSoname(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	elfFile, err := elf.NewFile(file)
	if err != nil {
		return "", fmt.Errorf("could not parse ELF file %s: %v", filename, err)
	}
	strings, err := elfFile.DynString(elf.DT_SONAME)
	if err != nil {
		return "", fmt.Errorf("when parsing .dynamic from %s: %v", filename, err)
	}
	if len(strings) == 0 {
		return "", nil
	}
	return strings[0], nil
}

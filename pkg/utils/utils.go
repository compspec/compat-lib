package utils

import (
	"bufio"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PathExists determines if a path exists
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return true, fmt.Errorf("warning: exists but another error happened (debug): %s", err)
	}
	return true, nil
}

// FullPath gets a full executable path, and makes it absolute
func FullPath(path string) (string, error) {
	path, err := exec.LookPath(path)
	if err != nil {
		return "", err
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return "", err

	}
	return path, nil
}

// chunkify a count of processors across sockets
func Chunkify(items []string, count int) [][]string {
	var chunks [][]string
	chunkSize := int(math.Ceil(float64(len(items) / count)))

	for i := 0; i < len(items); i += chunkSize {
		end := i + chunkSize
		if end > len(items) {
			end = len(items)
		}
		chunks = append(chunks, items[i:end])
	}
	return chunks
}

// SplitDelimiterList splits a list of items by an additional delimiter
func SplitDelimiterList(items []string, delim string) (map[string]string, error) {

	data := map[string]string{}
	for _, item := range items {
		pieces := strings.Split(item, delim)
		if len(pieces) > 1 {
			data[pieces[0]] = pieces[1]
		} else {
			data[pieces[0]] = ""
		}
	}
	return data, nil
}

// lookup a value, return error if not defined for either
func LookupValue(
	p map[string]string,
	keys ...string,
) (string, error) {

	var value string
	for _, key := range keys {
		value, ok := p[key]
		if ok {
			return value, nil
		}
	}
	return value, fmt.Errorf("cannot find any keys in %s", keys)
}

// ArrayContainsString determines if a string is in an array
// We return an array of invalid names in case the calling function needs
func StringArrayIsSubset(contenders, items []string) ([]string, bool) {

	validSet := map[string]bool{}
	for _, item := range items {
		validSet[item] = true
	}

	valid := true
	invalids := []string{}
	for _, contender := range contenders {
		_, ok := validSet[contender]

		// This contender is not known
		if !ok {
			valid = false
			invalids = append(invalids, contender)
		}
	}
	return invalids, valid
}

// ParseConfigFile parses a simple configuration file, with newlines for each thing,
// and a starting prefix to determine comma, and some other delimiter to determine
// key value pairs
func ParseConfigFile(path, comment, delim string) (map[string]string, error) {

	// We will return key value pairs
	data := map[string]string{}

	fd, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	// Read through with a scanner, this looks to be a parse file
	s := bufio.NewScanner(fd)
	s.Split(bufio.ScanLines)
	for s.Scan() {
		line := s.Text()

		// Skip over lines that are commented out
		if strings.HasPrefix(line, comment) {
			continue
		}

		// Parse lines (splitting by delim) that are not
		parts := strings.Split(line, delim)
		if len(parts) >= 2 {
			data[parts[0]] = parts[1]
		}
	}
	return data, nil
}

// RandomSort an array of strings, in place
func RandomSort(items []string) []string {
	for i := range items {
		j := rand.Intn(i + 1)
		items[i], items[j] = items[j], items[i]
	}
	return items
}

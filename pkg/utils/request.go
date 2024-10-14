package utils

import (
	"encoding/json"
	"net/http"
	"time"
)

// GetJsonUrl gets json from a url and decodes into target interface
func GetJsonUrl(url string, target any) error {

	cli := &http.Client{Timeout: 10 * time.Second}
	r, err := cli.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(target)
}

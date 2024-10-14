package oras

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/compspec/compat-lib/pkg/compat"
	"github.com/compspec/compat-lib/pkg/utils"
	oci "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/registry/remote"
	"sigs.k8s.io/yaml"
)

// toFilename converts the uri of an image to a filename
func toFilename(uri string) string {
	for _, repl := range []string{"/", ":"} {
		uri = strings.ReplaceAll(uri, repl, "-")
	}
	return fmt.Sprintf("%s.json", uri)
}

// LoadFromCache loads the compatibility request from cache
func LoadFromCache(uri, cache string) (*compat.CompatibiitySpec, error) {
	var request compat.CompatibiitySpec
	var err error
	cachePath := filepath.Join(cache, toFilename(uri))
	exists, err := utils.PathExists(cachePath)
	if err != nil {
		return &request, err
	}
	if exists {
		fd, err := os.Open(cachePath)
		b, err := io.ReadAll(fd)
		if err != nil {
			return &request, err
		}
		defer fd.Close()
		err = json.Unmarshal(b, &request)
	}
	return &request, err
}

// Oras will provide an interface to retrieve an artifact, specifically
// a compatibillity spec artifact media type
// LoadArtifact retrieves the artifact from the url string
// and returns based on the media type
func LoadArtifact(
	uri string,
	mediaType string,
	cache string,
) (*compat.CompatibiitySpec, error) {

	request := &compat.CompatibiitySpec{}
	var err error

	// If cache is desired and we have the artifact
	if cache != "" {

		// Must exist
		exists, err := utils.PathExists(cache)
		if err != nil {
			return request, err
		}
		if !exists {
			return request, fmt.Errorf("Cache path %s does not exist", cache)
		}
		request, err := LoadFromCache(uri, cache)
		if err != nil {
			return request, err
		}
	}

	// If we didn't get matches, load from registry
	if request.Version == "" {
		request, err = LoadFromRegistry(uri, mediaType)

		// If we loaded it and have a cache, save to cache
		if cache != "" {
			err = SaveToCache(request, uri, cache)
		}
	}
	return request, err
}

// Save to cache
func SaveToCache(request *compat.CompatibiitySpec, uri, cache string) error {
	cachePath := filepath.Join(cache, toFilename(uri))
	exists, err := utils.PathExists(cachePath)
	if err != nil {
		return err
	}

	// Don't overwrite
	if exists {
		return nil
	}
	content, err := json.Marshal(request)
	if err != nil {
		return err
	}
	err = os.WriteFile(cachePath, content, 0644)
	if err != nil {
		return err
	}
	return nil
}

// Load the artifact from a registry
func LoadFromRegistry(uri, mediaType string) (*compat.CompatibiitySpec, error) {
	request := compat.CompatibiitySpec{}
	ctx := context.Background()
	repo, err := remote.NewRepository(uri)
	if err != nil {
		return &request, err
	}

	// Disable plain http for now, assume using production registry
	repo.PlainHTTP = false

	// Split reference into tag, or assume latest
	tag := "latest"
	if strings.Contains(uri, ":") {
		parts := strings.Split(uri, ":")
		tag = parts[1]
		uri = parts[0]
	}

	// Fetch manifest for the tag
	desc, readCloser, err := repo.FetchReference(ctx, tag)
	if err != nil {
		return &request, err
	}
	defer readCloser.Close()

	// Read the pulled content
	manifestBytes, err := content.ReadAll(readCloser, desc)
	if err != nil {
		return &request, err
	}

	// Going to be a big wild here and not check the mnaifest media type.
	// We'd want to find oras, but no reason it can't be pushed another way...
	// unmarshall it
	var manifest oci.Manifest
	err = json.Unmarshal(manifestBytes, &manifest)
	if err != nil {
		return &request, err
	}

	// Loop through layers and find the media type we are looking for
	for _, layer := range manifest.Layers {

		// Skip layers that are not the compatibility spec... we seek
		if layer.MediaType != mediaType {
			continue
		}

		// Get the descriptor for the digest we want
		desc, err := repo.Blobs().Resolve(ctx, string(layer.Digest))
		if err != nil {
			return &request, err
		}

		// Download using the descriptor
		readCloser, err := repo.Fetch(ctx, desc)
		if err != nil {
			return &request, err
		}
		defer readCloser.Close()

		// Read the descriptor into bytes
		vr := content.NewVerifyReader(readCloser, desc)
		buffer := bytes.NewBuffer(nil)
		_, err = io.Copy(buffer, vr)
		if err != nil {
			return &request, err
		}

		// note: users should not trust the the read content until Verify returns nil
		if err := vr.Verify(); err != nil {
			return &request, err
		}

		// Convert this into our Compatibility Request
		// reading from the buffer into bytes proper
		readContent, err := io.ReadAll(buffer)
		if err != nil {
			return &request, err
		}

		err = yaml.Unmarshal(readContent, &request)
		if err != nil {
			return &request, err
		}
	}
	return &request, nil
}

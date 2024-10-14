package compat

import (
	"encoding/json"

	"github.com/compspec/compat-lib/pkg/version"
)

// NewCompatibilitySpec returns a new compatibility spec
func NewCompatibilitySpec() *CompatibiitySpec {
	spec := CompatibiitySpec{Version: version.Version}
	spec.Attributes = Attributes{}
	return &spec
}

// A compatibility spec describes an application
// Attributes are key value pairs, namespace TBA
// Let's keep it simple like that.
type CompatibiitySpec struct {
	Version    string     `json:"version"`
	Attributes Attributes `json:"attributes"`
}

type Attributes map[string]string

// AddAttributes adds attributes to the artifact
func (s *CompatibiitySpec) AddAttribute(key, value string) {
	s.Attributes[key] = value
}

// ToJson dumps our request to json for the artifact
func (r *CompatibiitySpec) ToJson() ([]byte, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return []byte{}, err
	}
	return b, err
}

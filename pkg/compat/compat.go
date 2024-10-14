package compat

import (
	"fmt"
	"path/filepath"
)

// GenerateLibraryArtifact generates an artifact to describe a library of interest
// We will want to use this to determine if a system can support running an application
func GenerateLibraryArtifact(name string, libs []string) *CompatibiitySpec {

	// Ensure we have the basename
	basename := filepath.Base(name) //use this built-in function to obtain filename

	// Generate the compatibility spec
	artifact := NewCompatibilitySpec()
	artifact.AddAttribute("llnl.compatlib.executable-name", basename)
	for i, lib := range libs {
		key := fmt.Sprintf("llnl.compatlib.library-name.%d", i)
		artifact.AddAttribute(key, lib)
	}
	return artifact
}

package cmd

import (
	"fmt"
	"io"
	"os"
)

// resolveBody returns the body text from either the --body flag value or the
// contents of the file specified by --body-file.  When bodyFile is "-", stdin
// is read.  If both values are provided the caller has a configuration error;
// mutual-exclusivity should be enforced by cobra flag groups, but this
// function returns a defensive error as well.  When neither is provided an
// empty string is returned — downstream validation decides whether that is
// acceptable.
func resolveBody(body, bodyFile string) (string, error) {
	if body != "" && bodyFile != "" {
		return "", fmt.Errorf("specify --body or --body-file, not both")
	}
	if bodyFile == "" {
		return body, nil
	}

	var data []byte
	var err error
	if bodyFile == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(bodyFile) // #nosec G304 — user-supplied path is intentional
	}
	if err != nil {
		return "", fmt.Errorf("failed to read body file: %w", err)
	}
	return string(data), nil
}

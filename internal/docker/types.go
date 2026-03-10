package docker

import (
	"path"
	"strings"
)

// MatchContainerNames returns the subset of container names that satisfy the
// provided glob pattern. Container names mirror Docker responses, so leading
// slashes are trimmed before matching.
func MatchContainerNames(names []string, pattern string) ([]string, error) {
	var matches []string
	for _, raw := range names {
		normalized := normalizeContainerName(raw)
		matched, err := path.Match(pattern, normalized)
		if err != nil {
			return nil, err
		}
		if matched {
			matches = append(matches, normalized)
		}
	}
	return matches, nil
}

func normalizeContainerName(name string) string {
	return strings.TrimPrefix(name, "/")
}

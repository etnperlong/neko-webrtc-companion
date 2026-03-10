package docker

import (
	"path"
	"strings"
)

// ContainerFilters defines optional container selection rules.
// Only configured filters participate, and all configured filters must match.
type ContainerFilters struct {
	NamePattern  string
	ImagePattern string
	LabelTrueKey string
}

// MatchContainerNames returns the subset of container names that satisfy the
// provided glob pattern. Container names mirror Docker responses, so leading
// slashes are trimmed before matching.
func MatchContainerNames(names []string, pattern string) ([]string, error) {
	var matches []string
	for _, raw := range names {
		normalized := normalizeContainerName(raw)
		if pattern == "" {
			matches = append(matches, normalized)
			continue
		}
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

func matchesImage(image, pattern string) (bool, error) {
	if pattern == "" {
		return true, nil
	}
	return path.Match(pattern, image)
}

func hasLabelTrueValue(labels map[string]string, key string) bool {
	if key == "" {
		return true
	}
	if labels == nil {
		return false
	}
	return labels[key] == "true"
}

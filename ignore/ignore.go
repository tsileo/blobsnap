/*

Package ignore provides utils to implement .gitignore like ignore mechanism.

*/
package ignore

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gignore "github.com/sabhiram/go-git-ignore"
)

// ParseIgnoreFile parses the given file and returns the lists of pattern extracted
func ParseIgnoreFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %v: %v", path, err)
	}
	defer f.Close()
	var patterns []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		pattern := filepath.Clean(strings.TrimSpace(scanner.Text()))
		if pattern != "" {
			patterns = append(patterns, pattern)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse file %v: %v", path, err)
	}
	return patterns, nil
}

// Matches returns true if name matches at least one of the pattern
func Matches(patterns []string, path string) (bool, error) {
	c, _ := gignore.CompileIgnoreLines(patterns...)
	return c.MatchesPath(path), nil
}

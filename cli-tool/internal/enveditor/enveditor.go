package enveditor

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ParseEnvFile reads a .env file and returns its contents as a map.
// Blank lines and lines beginning with # are ignored. Values may contain
// additional = characters; only the first = is used as the delimiter.
func ParseEnvFile(file string) (map[string]string, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	result := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx < 0 {
			continue
		}
		result[line[:idx]] = line[idx+1:]
	}
	return result, scanner.Err()
}

// SetEnvValue writes or updates a key=value in a .env file
func SetEnvValue(file, key, value string) error {
	lines := []string{}
	exists := false

	// Read existing lines
	f, err := os.Open(file)
	if err == nil {
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, key+"=") {
				line = fmt.Sprintf("%s=%s", key, value)
				exists = true
			}
			lines = append(lines, line)
		}
	}

	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// If the key didn't exist, append it
	if !exists {
		lines = append(lines, fmt.Sprintf("%s=%s", key, value))
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
		return err
	}

	// Write the file
	f2, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f2.Close()

	for _, line := range lines {
		_, _ = fmt.Fprintln(f2, line)
	}

	return nil
}

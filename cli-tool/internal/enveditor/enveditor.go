package enveditor

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

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

	// Write the file
	f2, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f2.Close()

	for _, line := range lines {
		_, _ = fmt.Fprintln(f2, line)
	}

	return nil
}

package core

import (
	"bufio"
	"os"
	"strings"
)

// LoadEnv loads environment variables from a .env file in the current working directory if it exists.
func LoadEnv() {
	file, err := os.Open(".env")
	if err != nil {
		return // Ignore if file does not exist or cannot be opened
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Ignore empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Strip quotes if present
		if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
			(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
			if len(value) >= 2 {
				value = value[1 : len(value)-1]
			}
		}

		// Set environment variable if not already defined
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, value)
		}
	}
}

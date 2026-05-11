package vocab

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Load returns a whisper.cpp initial_prompt string from the given comma-separated
// terms and/or vocab file path. Returns empty string if no vocabulary is configured.
func Load(terms, filePath string) (string, error) {
	var all []string

	if terms != "" {
		for _, t := range strings.Split(terms, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				all = append(all, t)
			}
		}
	}

	if filePath != "" {
		fileTerms, err := readFile(filePath)
		if err != nil {
			return "", err
		}
		all = append(all, fileTerms...)
	}

	if len(all) == 0 {
		return "", nil
	}

	return "The following terms may appear: " + strings.Join(all, ", ") + ".", nil
}

func readFile(path string) ([]string, error) {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("expand home directory: %w", err)
		}
		path = home + path[1:]
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open vocab file %s: %w", path, err)
	}
	defer f.Close()

	var terms []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		t := strings.TrimSpace(scanner.Text())
		if t != "" && !strings.HasPrefix(t, "#") {
			terms = append(terms, t)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read vocab file %s: %w", path, err)
	}
	return terms, nil
}

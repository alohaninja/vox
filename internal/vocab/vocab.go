package vocab

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// MaxTerms is the maximum number of vocabulary terms that will be included in
// the whisper.cpp initial_prompt. The prompt prefix window is ~224 tokens, so
// large term lists would be silently truncated by the model.
const MaxTerms = 500

// Load returns a whisper.cpp initial_prompt string from the given comma-separated
// terms and/or vocab file path. Returns empty string if no vocabulary is configured.
// Duplicate terms are removed and the total count is capped at MaxTerms.
func Load(terms, filePath string) (string, error) {
	var all []string
	seen := make(map[string]bool)

	addUnique := func(t string) {
		t = strings.TrimSpace(t)
		if t == "" {
			return
		}
		key := strings.ToLower(t)
		if !seen[key] {
			seen[key] = true
			all = append(all, t)
		}
	}

	if terms != "" {
		for _, t := range strings.Split(terms, ",") {
			addUnique(t)
		}
	}

	if filePath != "" {
		fileTerms, err := readFile(filePath)
		if err != nil {
			return "", err
		}
		for _, t := range fileTerms {
			addUnique(t)
		}
	}

	if len(all) == 0 {
		return "", nil
	}

	if len(all) > MaxTerms {
		all = all[:MaxTerms]
	}

	return "The following terms may appear: " + strings.Join(all, ", ") + ".", nil
}

func readFile(path string) ([]string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("expand home directory: %w", err)
		}
		if path == "~" {
			path = home
		} else {
			path = home + path[1:]
		}
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

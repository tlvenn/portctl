package cmd

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/tlvenn/portctl/internal/client"
)

// loadEnvFile reads KEY=VALUE lines like a docker compose .env file: blank
// lines and # comments are skipped, and matching quotes around a value are
// removed.
func loadEnvFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	values := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for n := 1; scanner.Scan(); n++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(strings.TrimPrefix(line, "export "), "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			return nil, fmt.Errorf("%s:%d: expected KEY=VALUE", path, n)
		}
		value = strings.TrimSpace(value)
		if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') && value[len(value)-1] == value[0] {
			value = value[1 : len(value)-1]
		}
		values[key] = value
	}
	return values, scanner.Err()
}

// mergeEnv sets the file's values over a stack's env. Keys the file does not
// mention, such as secrets set with `portctl env`, are kept.
func mergeEnv(env []client.EnvVar, file map[string]string) []client.EnvVar {
	merged := make(map[string]string, len(env)+len(file))
	for _, e := range env {
		merged[e.Name] = e.Value
	}
	for k, v := range file {
		merged[k] = v
	}

	keys := make([]string, 0, len(merged))
	for k := range merged {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make([]client.EnvVar, 0, len(keys))
	for _, k := range keys {
		out = append(out, client.EnvVar{Name: k, Value: merged[k]})
	}
	return out
}

// withEnvFile applies PORTCTL_ENV_FILE, when set, to a stack's env.
func withEnvFile(env []client.EnvVar) ([]client.EnvVar, error) {
	if cfg.EnvFile == "" {
		return env, nil
	}
	file, err := loadEnvFile(cfg.EnvFile)
	if err != nil {
		return nil, fmt.Errorf("PORTCTL_ENV_FILE: %w", err)
	}
	fmt.Printf("Applying %d variable(s) from %s\n", len(file), cfg.EnvFile)
	return mergeEnv(env, file), nil
}

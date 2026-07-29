// Package envfile loads KEY=VALUE pairs from a dotenv-style file into the process env.
// Values are never logged.
package envfile

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Load reads path and sets environment variables for lines of the form KEY=VALUE.
// Existing non-empty process env values are not overwritten.
// Missing path is a no-op (returns nil).
// Never logs keys or values.
func Load(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("envfile: open: %w", err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	// Allow long values (e.g. JSON RPC maps) without logging content.
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Optional export prefix.
		line = strings.TrimPrefix(line, "export ")
		line = strings.TrimSpace(line)
		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			return fmt.Errorf("envfile: invalid line %d (expected KEY=VALUE)", lineNo)
		}
		key := strings.TrimSpace(line[:eq])
		if key == "" {
			return fmt.Errorf("envfile: empty key on line %d", lineNo)
		}
		val := line[eq+1:]
		val = unquote(strings.TrimSpace(val))
		if cur := strings.TrimSpace(os.Getenv(key)); cur != "" {
			continue
		}
		if err := os.Setenv(key, val); err != nil {
			return fmt.Errorf("envfile: setenv: %w", err)
		}
	}
	if err := sc.Err(); err != nil {
		return fmt.Errorf("envfile: read: %w", err)
	}
	return nil
}

func unquote(v string) string {
	if len(v) < 2 {
		return v
	}
	if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
		return v[1 : len(v)-1]
	}
	return v
}

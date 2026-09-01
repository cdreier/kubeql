package graph

import (
	"strings"
	"time"
)

// splitLogTimestamp parses optional RFC3339 prefix from kubectl-style timestamps=true lines.
func splitLogTimestamp(line string) (time.Time, string, bool) {
	// Format: 2024-01-02T15:04:05.000000000Z message
	parts := strings.SplitN(line, " ", 2)
	if len(parts) != 2 {
		return time.Time{}, "", false
	}
	ts, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		ts, err = time.Parse(time.RFC3339, parts[0])
		if err != nil {
			return time.Time{}, "", false
		}
	}
	return ts, parts[1], true
}

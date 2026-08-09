package audit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// QueryBuilder builds audit log queries.
type QueryBuilder struct {
	file      string
	actor     string
	op        string
	startTime time.Time
	endTime   time.Time
}

// FilterByActor restricts results to the given actor.
// Pass empty string to skip this filter.
func (q *QueryBuilder) FilterByActor(actor string) *QueryBuilder {
	q.actor = actor
	return q
}

// FilterByOp restricts results to the given operation string
// (e.g. "READ", "WRITE", "ADMIN").
// Pass empty string to skip this filter.
func (q *QueryBuilder) FilterByOp(op string) *QueryBuilder {
	q.op = op
	return q
}

// FilterByTimeRange restricts results to entries within [start, end] (UTC).
func (q *QueryBuilder) FilterByTimeRange(start, end time.Time) *QueryBuilder {
	q.startTime = start
	q.endTime = end
	return q
}

// Execute runs the query and returns matching LogEntry records.
func (q *QueryBuilder) Execute() ([]LogEntry, error) {
	f, err := os.Open(q.file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // no log file yet — empty result
		}
		return nil, fmt.Errorf("audit.Query.Execute: Open: %w", err)
	}
	defer f.Close()

	var results []LogEntry
	scanner := bufio.NewScanner(f)
	var seen int

	for scanner.Scan() {
		if seen >= 10000 { // safety cap
			break
		}
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry LogEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}

		if q.actor != "" && entry.Actor != q.actor {
			continue
		}
		if q.op != "" && entry.Op != q.op {
			continue
		}
		if !q.startTime.IsZero() && entry.TS.Before(q.startTime) {
			continue
		}
		if !q.endTime.IsZero() && entry.TS.After(q.endTime) {
			continue
		}

		results = append(results, entry)
		seen++
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("audit.Query.Execute: scanner: %w", err)
	}
	return results, nil
}

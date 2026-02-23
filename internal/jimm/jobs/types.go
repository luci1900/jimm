// Copyright 2026 Canonical.

package jobs

import (
	"time"

	apiparams "github.com/canonical/jimm/v3/pkg/api/params"
)

// JobError represents an error that occurred during a job attempt.
type JobError struct {
	At      time.Time
	Attempt int
	Error   string
}

// JobInfo represents the information about a job, including
// its status, kind, attempts, and any errors that occurred.
type JobInfo struct {
	ID             int64
	Status         apiparams.JobStatus
	Kind           string
	CurrentAttempt int
	MaxAttempts    int
	FinishedAt     *time.Time
	Errors         []JobError
}

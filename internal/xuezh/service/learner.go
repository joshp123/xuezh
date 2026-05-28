package service

import (
	"time"

	"github.com/joshp123/xuezh/internal/xuezh/cram"
	"github.com/joshp123/xuezh/internal/xuezh/snapshot"
)

func (App) LearnerState(now time.Time) (cram.LearnerState, error) {
	return cram.LearnerStateFor(now)
}

func (App) Snapshot(window string, dueLimit int, evidenceLimit int, maxBytes int) (snapshot.Result, error) {
	return snapshot.BuildSnapshot(window, dueLimit, evidenceLimit, maxBytes)
}

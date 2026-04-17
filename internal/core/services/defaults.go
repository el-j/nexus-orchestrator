package services

import "time"

const (
	DefaultQueueCap           = 50
	DefaultPurgeDisconnectAge = 2 * time.Hour
	DefaultWatchdogStale      = 5 * time.Minute
)

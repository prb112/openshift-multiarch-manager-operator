package e2e

import "time"

const (
	WaitShort       = 1 * time.Minute
	WaitMedium      = 3 * time.Minute
	WaitOverMedium  = 5 * time.Minute
	WaitLong        = 15 * time.Minute
	WaitOverLong    = 30 * time.Minute
	PollingInterval = 1 * time.Second
)

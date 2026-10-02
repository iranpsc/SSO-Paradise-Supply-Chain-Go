package main

import (
	"time"
	_ "time/tzdata"
)

func nextMidnight(now time.Time, zone *time.Location) time.Time {
	local := now.In(zone)
	return time.Date(local.Year(), local.Month(), local.Day()+1, 0, 0, 0, 0, zone)
}

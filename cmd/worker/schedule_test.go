package main

import (
	"testing"
	"time"
)

func TestNextCleanupAtTehranMidnight(t *testing.T) {
	zone, err := time.LoadLocation("Asia/Tehran")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 2, 23, 59, 0, 0, zone)
	next := nextMidnight(now, zone)
	if next.Day() != 3 || next.Hour() != 0 || next.Sub(now) != time.Minute {
		t.Fatal("cleanup not scheduled at local midnight")
	}
}

package store

import (
	"crypto/rand"
	"fmt"
	"time"
)

// newUUID produces a random v4-shaped UUID using crypto/rand, mirroring the
// mock's own hand-rolled generator (kept dependency-free rather than
// pulling in google/uuid for a single function).
func newUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// nowISO matches JavaScript's `new Date().toISOString()` format exactly
// (millisecond precision, Z suffix).
func nowISO() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
}

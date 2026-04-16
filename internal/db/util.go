package db

import "time"

// nullTime returns nil for zero time values (maps to SQL NULL).
func nullTime(t time.Time) interface{} {
	if t.IsZero() {
		return nil
	}
	return t
}

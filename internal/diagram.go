package internal

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
	"time"
)

// NewDiagram returns a new diagram.
func NewDiagram(description []byte, imgType string) *Diagram {
	return &Diagram{
		description: []byte(strings.TrimSpace(string(description))),
		lastTouched: time.Now(),
		mu:          &sync.RWMutex{},
		imgType:     imgType,
	}
}

// Diagram represents a single diagram.
type Diagram struct {
	id          string
	description []byte
	// Output is the filepath to the output file.
	Output      string
	mu          *sync.RWMutex
	lastTouched time.Time
	imgType     string
}

// Touch updates the last touched time of the diagram.
func (d *Diagram) Touch() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.lastTouched = time.Now()
}

// TouchedInDuration returns true if the diagram has been touched in the given duration.
func (d *Diagram) TouchedInDuration(duration time.Duration) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return time.Now().Add(-duration).Before(d.lastTouched)
}

// ID returns an ID for the diagram. The ID is derived from the diagram description.
func (d *Diagram) ID() (string, error) {
	if d.id != "" {
		return d.id, nil
	}
	hash := sha256.Sum256(d.description)
	d.id = hex.EncodeToString(hash[:]) + d.imgType
	return d.id, nil
}

// Description returns the diagram description.
func (d *Diagram) Description() []byte {
	return d.description
}

// WithDescription replaces the description and invalidates the cached ID.
func (d *Diagram) WithDescription(description []byte) *Diagram {
	d.description = description
	d.id = ""
	return d
}

package internal

import (
	"fmt"
	"sync"
)

// DiagramCache provides the ability to cache diagram results.
type DiagramCache interface {
	// Store stores a diagram in the cache.
	Store(diagram *Diagram) error
	// Has returns true if we have a cache stored for the given diagram description.
	Has(diagram *Diagram) (bool, error)
	// Get returns a cached version of the given diagram description.
	Get(diagram *Diagram) (*Diagram, error)
	// GetAll returns all of the cached diagrams.
	GetAll() ([]*Diagram, error)
	// Delete deletes a cached version of the given diagram.
	Delete(diagram *Diagram) error
}

// NewDiagramCache returns an implementation of DiagramCache.
func NewDiagramCache() DiagramCache {
	return &inMemoryDiagramCache{
		idToDiagram: map[string]*Diagram{},
	}
}

// inMemoryDiagramCache is an in-memory implementation of DiagramCache.
type inMemoryDiagramCache struct {
	mu          sync.RWMutex
	idToDiagram map[string]*Diagram
}

// Store stores a diagram in the cache.
func (c *inMemoryDiagramCache) Store(diagram *Diagram) error {
	id, err := diagram.ID()
	if err != nil {
		return fmt.Errorf("cannot get diagram ID: %w", err)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.idToDiagram[id] = diagram
	return nil
}

// Has returns true if we have a cache stored for the given diagram description.
func (c *inMemoryDiagramCache) Has(diagram *Diagram) (bool, error) {
	id, err := diagram.ID()
	if err != nil {
		return false, fmt.Errorf("cannot get diagram ID: %w", err)
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	d, ok := c.idToDiagram[id]
	return ok && d != nil, nil
}

// Get returns a cached version of the given diagram description.
func (c *inMemoryDiagramCache) Get(diagram *Diagram) (*Diagram, error) {
	id, err := diagram.ID()
	if err != nil {
		return nil, err
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if d, ok := c.idToDiagram[id]; ok && d != nil {
		return d, nil
	}
	return nil, nil
}

// GetAll returns all of the cached diagrams.
func (c *inMemoryDiagramCache) GetAll() ([]*Diagram, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	res := make([]*Diagram, 0, len(c.idToDiagram))
	for _, diagram := range c.idToDiagram {
		res = append(res, diagram)
	}
	return res, nil
}

// Delete deletes a cached version of the given diagram.
func (c *inMemoryDiagramCache) Delete(diagram *Diagram) error {
	id, err := diagram.ID()
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.idToDiagram, id)
	return nil
}

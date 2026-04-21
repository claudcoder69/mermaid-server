package internal

import (
	"sync"
	"testing"
)

func TestCacheStoreHasGetDelete(t *testing.T) {
	c := NewDiagramCache()
	d := NewDiagram([]byte("graph LR\nA-->B"), "svg")

	has, err := c.Has(d)
	if err != nil {
		t.Fatal(err)
	}
	if has {
		t.Fatal("empty cache reported Has=true")
	}

	if err := c.Store(d); err != nil {
		t.Fatal(err)
	}

	has, _ = c.Has(d)
	if !has {
		t.Fatal("expected Has=true after Store")
	}

	got, err := c.Get(d)
	if err != nil {
		t.Fatal(err)
	}
	if got != d {
		t.Fatal("Get returned a different pointer than Store received")
	}

	all, err := c.GetAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 {
		t.Fatalf("GetAll returned %d entries, want 1", len(all))
	}

	if err := c.Delete(d); err != nil {
		t.Fatal(err)
	}
	has, _ = c.Has(d)
	if has {
		t.Fatal("Has=true after Delete")
	}
}

func TestCacheConcurrentAccess(t *testing.T) {
	c := NewDiagramCache()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			d := NewDiagram([]byte{byte(i)}, "svg")
			if err := c.Store(d); err != nil {
				t.Error(err)
			}
			if _, err := c.Has(d); err != nil {
				t.Error(err)
			}
			if _, err := c.GetAll(); err != nil {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()
}

package internal

import (
	"testing"
	"time"
)

func TestDiagramIDIsStableAndDependsOnTypeAndDescription(t *testing.T) {
	a := NewDiagram([]byte("graph LR\nA-->B"), "svg")
	b := NewDiagram([]byte("graph LR\nA-->B"), "svg")
	aID, err := a.ID()
	if err != nil {
		t.Fatal(err)
	}
	bID, err := b.ID()
	if err != nil {
		t.Fatal(err)
	}
	if aID != bID {
		t.Fatalf("expected identical diagrams to share an ID, got %q vs %q", aID, bID)
	}

	c := NewDiagram([]byte("graph LR\nA-->B"), "png")
	cID, _ := c.ID()
	if cID == aID {
		t.Fatalf("expected different image types to produce different IDs")
	}

	d := NewDiagram([]byte("graph LR\nA-->C"), "svg")
	dID, _ := d.ID()
	if dID == aID {
		t.Fatalf("expected different descriptions to produce different IDs")
	}
}

func TestDiagramDescriptionIsTrimmed(t *testing.T) {
	d := NewDiagram([]byte("  graph LR\nA-->B  \n"), "svg")
	if got, want := string(d.Description()), "graph LR\nA-->B"; got != want {
		t.Fatalf("description = %q, want %q", got, want)
	}
}

func TestTouchedInDuration(t *testing.T) {
	d := NewDiagram([]byte("x"), "svg")
	if !d.TouchedInDuration(time.Second) {
		t.Fatal("freshly created diagram should be within 1s of last-touched")
	}
	d.lastTouched = time.Now().Add(-2 * time.Hour)
	if d.TouchedInDuration(time.Hour) {
		t.Fatal("two-hour-old diagram should not be within 1h")
	}
	d.Touch()
	if !d.TouchedInDuration(time.Second) {
		t.Fatal("Touch should reset lastTouched to now")
	}
}

func TestWithDescriptionInvalidatesID(t *testing.T) {
	d := NewDiagram([]byte("a"), "svg")
	first, _ := d.ID()
	d.WithDescription([]byte("b"))
	second, _ := d.ID()
	if first == second {
		t.Fatal("changing the description should invalidate the cached ID")
	}
}

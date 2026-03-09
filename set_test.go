package sg

import (
	"testing"
)

func TestSetAddAndHas(t *testing.T) {
	s := NewSet[string]()
	s.Add("a")
	s.Add("b")
	if !s.Has("a") {
		t.Error("expected set to contain 'a'")
	}
	if !s.Has("b") {
		t.Error("expected set to contain 'b'")
	}
	if s.Has("c") {
		t.Error("expected set to not contain 'c'")
	}
}

func TestSetAddAll(t *testing.T) {
	s := NewSet[int]()
	s.AddAll(1, 2, 3)
	if s.Size() != 3 {
		t.Errorf("expected size 3, got %d", s.Size())
	}
}

func TestSetRemove(t *testing.T) {
	s := NewSet[string]()
	s.Add("x")
	if !s.Remove("x") {
		t.Error("expected Remove to return true for existing element")
	}
	if s.Has("x") {
		t.Error("expected set to not contain 'x' after removal")
	}
	if s.Remove("y") {
		t.Error("expected Remove to return false for non-existing element")
	}
}

func TestSetSize(t *testing.T) {
	s := NewSet[string]()
	if s.Size() != 0 {
		t.Errorf("expected empty set size 0, got %d", s.Size())
	}
	s.Add("a")
	s.Add("a") // duplicate
	if s.Size() != 1 {
		t.Errorf("expected size 1 after adding duplicate, got %d", s.Size())
	}
}

func TestSetSlice(t *testing.T) {
	s := NewSet[int]()
	s.AddAll(1, 2, 3)
	sl := s.Slice()
	if len(sl) != 3 {
		t.Errorf("expected slice length 3, got %d", len(sl))
	}
}

func TestSetElements(t *testing.T) {
	s := NewSet[string]()
	s.AddAll("a", "b")
	count := 0
	for range s.Elements() {
		count++
	}
	if count != 2 {
		t.Errorf("expected 2 elements, got %d", count)
	}
}

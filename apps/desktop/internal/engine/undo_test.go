package engine

import "testing"

func TestUndoLIFOPerWindow(t *testing.T) {
	u := NewUndoStack()
	u.Push(1, Rect{1, 0, 0, 0})
	u.Push(1, Rect{2, 0, 0, 0})
	u.Push(2, Rect{9, 0, 0, 0})
	if r, ok := u.Pop(1); !ok || r.X != 2 {
		t.Fatalf("want LIFO for window 1, got %+v %v", r, ok)
	}
	if r, ok := u.Pop(2); !ok || r.X != 9 {
		t.Fatalf("windows are independent, got %+v %v", r, ok)
	}
	if _, ok := u.Pop(2); ok {
		t.Fatal("empty stack pops nothing")
	}
}

func TestUndoCap(t *testing.T) {
	u := NewUndoStack()
	for i := 0; i < 30; i++ {
		u.Push(1, Rect{X: float64(i)})
	}
	count := 0
	for {
		if _, ok := u.Pop(1); !ok {
			break
		}
		count++
	}
	if count != 20 {
		t.Fatalf("cap at 20, got %d", count)
	}
}

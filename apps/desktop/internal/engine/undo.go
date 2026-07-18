package engine

import "sync"

const undoCap = 20

// UndoStack remembers previous window frames per CGWindowID.
type UndoStack struct {
	mu     sync.Mutex
	frames map[uint32][]Rect
}

func NewUndoStack() *UndoStack {
	return &UndoStack{frames: map[uint32][]Rect{}}
}

func (u *UndoStack) Push(winID uint32, frame Rect) {
	u.mu.Lock()
	defer u.mu.Unlock()
	s := append(u.frames[winID], frame)
	if len(s) > undoCap {
		s = s[len(s)-undoCap:]
	}
	u.frames[winID] = s
}

func (u *UndoStack) Pop(winID uint32) (Rect, bool) {
	u.mu.Lock()
	defer u.mu.Unlock()
	s := u.frames[winID]
	if len(s) == 0 {
		return Rect{}, false
	}
	r := s[len(s)-1]
	u.frames[winID] = s[:len(s)-1]
	return r, true
}

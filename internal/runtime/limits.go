package runtime

type MemoryTracker struct {
	Current int64
	Peak    int64
	Limit   int64
}

func NewMemoryTracker(maxMB int) *MemoryTracker {
	if maxMB <= 0 {
		maxMB = 1024
	}
	return &MemoryTracker{Limit: int64(maxMB) * 1024 * 1024}
}

func (m *MemoryTracker) Add(n int64) bool {
	if n <= 0 {
		return m.Current > m.Limit
	}
	m.Current += n
	if m.Current > m.Peak {
		m.Peak = m.Current
	}
	return m.Current > m.Limit
}

func (m *MemoryTracker) Release(n int64) {
	if n <= 0 {
		return
	}
	m.Current -= n
	if m.Current < 0 {
		m.Current = 0
	}
}

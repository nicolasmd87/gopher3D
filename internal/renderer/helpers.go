package renderer

type Unwind []func()

func (u Unwind) Add(cleanup func()) {
	u = append(u, cleanup)
}

func (u Unwind) Unwind() {
	for i := len(u) - 1; i >= 0; i-- {
		u[i]()
	}
}

func (u Unwind) Discard() {
	if len(u) > 0 {
		u = u[:0]
	}
}

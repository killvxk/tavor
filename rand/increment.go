package rand

type IncrementRand struct {
	seed  int64
	value int64
}

func NewIncrementRand(seed int64) *IncrementRand {
	return &IncrementRand{
		seed:  seed,
		value: 0,
	}
}

func (r *IncrementRand) Int() int {
	return r.Intn(int(r.seed))
}

func (r *IncrementRand) Intn(n int) int {
	if n <= int(r.value) {
		r.value = 0
	}

	v := r.value

	r.value++

	return int(v)
}

func (r *IncrementRand) Int63() int64 {
	return int64(r.Int())
}

func (r *IncrementRand) Int63n(n int64) int64 {
	return int64(r.Intn(int(n)))
}

func (r *IncrementRand) Seed(seed int64) {
	r.seed = seed
	r.value = 0
}

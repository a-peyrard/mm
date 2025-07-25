package set

type Set[T comparable] map[T]struct{}

func New[T comparable]() Set[T] {
	return make(map[T]struct{})
}

func From[T comparable](slice []T) Set[T] {
	s := New[T]()
	for _, e := range slice {
		s.Add(e)
	}
	return s
}

func Of[T comparable](s ...T) Set[T] {
	return From(s)
}

func (s Set[T]) Add(v T) Set[T] {
	s[v] = struct{}{}

	return s
}

func (s Set[T]) Remove(v T) {
	delete(s, v)
}

func (s Set[T]) Contains(v T) bool {
	_, ok := s[v]
	return ok
}

func (s Set[T]) DoesNotContain(v T) bool {
	return !s.Contains(v)
}

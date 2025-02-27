package datafx

type Repository[T any] struct {
	Queries *T
}

func NewRepository[T any](queries *T) *Repository[T] {
	return &Repository[T]{Queries: queries}
}

func (r *Repository[T]) GetQueries() *T {
	return r.Queries
}

func (r *Repository[T]) WithQueries(queries *T) *Repository[T] {
	return &Repository[T]{Queries: queries}
}

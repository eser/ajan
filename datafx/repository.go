package datafx

type Repository[T any] struct {
	Tx *T
}

func NewRepository[T any](tx *T) *Repository[T] {
	return &Repository[T]{Tx: tx}
}

func (r *Repository[T]) GetTx() *T {
	return r.Tx
}

func (r *Repository[T]) WithTx(tx *T) *Repository[T] {
	return &Repository[T]{Tx: tx}
}

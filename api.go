package golinq

type PredicateSpec[T any] struct{}

func Predicate[T any](fn any) PredicateSpec[T] {
	return PredicateSpec[T]{}
}

type JoinSpec struct{}

func Join(left, right any, on any) JoinSpec {
	return JoinSpec{}
}

type QueryBuilder[T any] struct {
}

func Select[T any](db *DB, cols ...string) *QueryBuilder[T] {
	return &QueryBuilder[T]{}
}

func (qb *QueryBuilder[T]) Where(pred PredicateSpec[T], args ...any) *QueryBuilder[T] {
	return qb
}

func (qb *QueryBuilder[T]) Join(join JoinSpec) *QueryBuilder[T] {
	return qb
}

func (qb *QueryBuilder[T]) Limit(limit int) *QueryBuilder[T] {
	return qb
}

func (qb *QueryBuilder[T]) Offset(offset int) *QueryBuilder[T] {
	return qb
}

func (qb *QueryBuilder[T]) OrderBy(field string, desc ...bool) *QueryBuilder[T] {
	return qb
}

func (qb *QueryBuilder[T]) ToList() any {
	return nil
}

func (qb *QueryBuilder[T]) First() any {
	return nil
}

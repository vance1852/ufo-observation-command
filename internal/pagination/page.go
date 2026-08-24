package pagination

const (
	DefaultLimit = 50
	MaxLimit     = 100
)

type Query struct {
	Offset int
	Limit  int
}

func Normalize(offset, limit int) Query {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}
	return Query{Offset: offset, Limit: limit}
}

type Page[T any] struct {
	Items  []T `json:"items"`
	Total  int `json:"total"`
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

func From[T any](items []T, total int, query Query) Page[T] {
	if items == nil {
		items = []T{}
	}
	return Page[T]{Items: append([]T(nil), items...), Total: total, Offset: query.Offset, Limit: query.Limit}
}

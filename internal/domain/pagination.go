package domain

// ListParams is the common pagination/filter input for every collection
// listing (portfolio, journal, careers, enquiries). Handlers parse query
// params into this; repositories translate it into LIMIT/OFFSET.
type ListParams struct {
	Page            int
	PerPage         int
	PublishedOnly   bool
	IncludeUnpublished bool
}

func (p ListParams) Offset() int {
	return (p.Page - 1) * p.PerPage
}

// PageResult wraps any collection with the pagination metadata needed to
// render "page N of M" controls on the frontend.
type PageResult[T any] struct {
	Items      []T
	TotalItems int
	Page       int
	PerPage    int
}

func (r PageResult[T]) TotalPages() int {
	if r.PerPage <= 0 {
		return 0
	}
	pages := r.TotalItems / r.PerPage
	if r.TotalItems%r.PerPage != 0 {
		pages++
	}
	return pages
}

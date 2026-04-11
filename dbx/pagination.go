package dbx

import "github.com/DaiYuANg/arcgo/collectionx"

const (
	// DefaultPage is used when a page request has no valid page number.
	DefaultPage = 1
	// DefaultPageSize is used when a page request has no valid page size.
	DefaultPageSize = 20
)

// PageRequest is the shared offset-pagination request model used by query and repository APIs.
type PageRequest struct {
	Page        int
	PageSize    int
	MaxPageSize int
}

// Page creates a normalized page request.
func Page(page, pageSize int) PageRequest {
	return NewPageRequest(page, pageSize)
}

// NewPageRequest creates a normalized page request.
func NewPageRequest(page, pageSize int) PageRequest {
	return PageRequest{Page: page, PageSize: pageSize}.Normalize()
}

// WithMaxPageSize applies an upper bound to the page size.
func (r PageRequest) WithMaxPageSize(maxPageSize int) PageRequest {
	r.MaxPageSize = maxPageSize
	return r.Normalize()
}

// Normalize returns a request with valid page and page size values.
func (r PageRequest) Normalize() PageRequest {
	if r.Page < 1 {
		r.Page = DefaultPage
	}
	if r.PageSize < 1 {
		r.PageSize = DefaultPageSize
	}
	if r.MaxPageSize > 0 && r.PageSize > r.MaxPageSize {
		r.PageSize = r.MaxPageSize
	}
	return r
}

// Offset returns the zero-based row offset for this request.
func (r PageRequest) Offset() int {
	r = r.Normalize()
	return safePageOffset(r.Page, r.PageSize)
}

// Apply applies this request as LIMIT/OFFSET on a select query.
func (r PageRequest) Apply(query *SelectQuery) *SelectQuery {
	if query == nil {
		return nil
	}
	r = r.Normalize()
	return query.Limit(r.PageSize).Offset(r.Offset())
}

// PageResult contains the items and metadata for a paginated query.
type PageResult[E any] struct {
	Items       collectionx.List[E]
	Total       int64
	Page        int
	PageSize    int
	Offset      int
	TotalPages  int
	HasNext     bool
	HasPrevious bool
}

// NewPageResult creates a page result from items, total row count, and request metadata.
func NewPageResult[E any](items collectionx.List[E], total int64, request PageRequest) PageResult[E] {
	request = request.Normalize()
	totalPages := pageTotal(total, request.PageSize)
	return PageResult[E]{
		Items:       items,
		Total:       total,
		Page:        request.Page,
		PageSize:    request.PageSize,
		Offset:      request.Offset(),
		TotalPages:  totalPages,
		HasNext:     request.Page < totalPages,
		HasPrevious: request.Page > DefaultPage,
	}
}

// MapPageResult maps page items while preserving pagination metadata.
func MapPageResult[E any, R any](result PageResult[E], mapper func(index int, item E) R) PageResult[R] {
	return PageResult[R]{
		Items:       collectionx.MapList(result.Items, mapper),
		Total:       result.Total,
		Page:        result.Page,
		PageSize:    result.PageSize,
		Offset:      result.Offset,
		TotalPages:  result.TotalPages,
		HasNext:     result.HasNext,
		HasPrevious: result.HasPrevious,
	}
}

func safePageOffset(page, pageSize int) int {
	if page <= DefaultPage || pageSize <= 0 {
		return 0
	}
	multiplier := page - 1
	maxValue := maxInt()
	if multiplier > maxValue/pageSize {
		return maxValue
	}
	return multiplier * pageSize
}

func pageTotal(total int64, pageSize int) int {
	if total <= 0 || pageSize <= 0 {
		return 0
	}
	pages := ((total - 1) / int64(pageSize)) + 1
	if pages > int64(maxInt()) {
		return maxInt()
	}
	return int(pages)
}

func maxInt() int {
	return int(^uint(0) >> 1)
}

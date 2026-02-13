package domain

// PaginationParams holds pagination request parameters
type PaginationParams struct {
	Page     int `query:"page"`
	PageSize int `query:"page_size"`
}

// DefaultPage is the default page number
const DefaultPage = 1

// DefaultPageSize is the default number of items per page
const DefaultPageSize = 10

// MaxPageSize is the maximum allowed page size
const MaxPageSize = 100

// Normalize ensures pagination params are within valid bounds
func (p *PaginationParams) Normalize() {
	if p.Page < 1 {
		p.Page = DefaultPage
	}
	if p.PageSize < 1 {
		p.PageSize = DefaultPageSize
	}
	if p.PageSize > MaxPageSize {
		p.PageSize = MaxPageSize
	}
}

// Skip returns the number of documents to skip
func (p *PaginationParams) Skip() int64 {
	return int64((p.Page - 1) * p.PageSize)
}

// Limit returns the number of documents to return
func (p *PaginationParams) Limit() int64 {
	return int64(p.PageSize)
}

// PaginatedResponse wraps paginated data with metadata
type PaginatedResponse[T any] struct {
	Data       []T `json:"data"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int   `json:"total_pages"`
}

// NewPaginatedResponse creates a new paginated response
func NewPaginatedResponse[T any](data []T, total int64, params PaginationParams) PaginatedResponse[T] {
	totalPages := int(total) / params.PageSize
	if int(total)%params.PageSize > 0 {
		totalPages++
	}

	return PaginatedResponse[T]{
		Data:       data,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}
}

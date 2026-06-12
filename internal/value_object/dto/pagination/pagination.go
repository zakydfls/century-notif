// Package pagination provides utilities for paginating database queries,
// particularly with GORM. It handles parsing pagination parameters from
// URL query strings and provides methods for building paginated responses.
package pagination

import (
	"net/url"
	"strconv"
)

// Default pagination constants.
const (
	// DefaultLimit is the default number of items per page when not specified.
	DefaultLimit = 10
	// MaxLimit is the maximum allowed items per page to prevent excessive queries.
	MaxLimit = 100
	// DefaultPage is the default page number when not specified.
	DefaultPage = 1
)

// Pagination holds pagination state including current page, limit,
// total counts, and the result rows. It also stores additional query
// parameters that can be preserved across paginated requests.
type Pagination struct {
	// Limit is the maximum number of items to return per page.
	Limit int `json:"limit"`

	// Page is the current page number (1-indexed).
	Page int `json:"page"`
	// TotalPages is the total number of pages available.
	TotalPages int `json:"total_pages"`

	// TotalRows is the total count of all matching rows.
	TotalRows int64 `json:"total_rows"`
	// Rows holds the actual result data for the current page.
	Rows any `json:"rows"`

	// Query stores additional query parameters to preserve across requests.
	Query map[string]string `json:"queries,omitempty"`
}

// NewPagination creates a new Pagination instance from URL query parameters.
// It extracts "page" and "limit" from the query values and applies defaults
// and bounds checking. The query parameter is optional - if not provided or nil,
// default values will be used.
//
// The page defaults to DefaultPage (1) if not provided or invalid.
// The limit defaults to DefaultLimit (10) if not provided or invalid,
// and is capped at MaxLimit (100).
func NewPagination(query ...url.Values) *Pagination {
	if len(query) == 0 || query[0] == nil {
		return &Pagination{
			Page:  DefaultPage,
			Limit: DefaultLimit,
			Query: make(map[string]string),
		}
	}

	q := query[0]
	limitStr := q.Get("limit")
	pageStr := q.Get("page")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = DefaultPage
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}

	return &Pagination{
		Limit: limit,
		Page:  page,
		Query: make(map[string]string),
	}
}

// GetOffset calculates the offset for database queries based on
// the current page and limit. This is used for SQL OFFSET clauses.
func (p *Pagination) GetOffset() int {
	return (p.GetPage() - 1) * p.GetLimit()
}

// GetPage returns the current page number, ensuring it is at least 1.
// If the page is not set or invalid, it defaults to DefaultPage.
func (p *Pagination) GetPage() int {
	if p.Page <= 0 {
		p.Page = DefaultPage
	}
	return p.Page
}

// GetLimit returns the items per page limit, ensuring it is within
// valid bounds. Returns DefaultLimit if not set, and caps at MaxLimit.
func (p *Pagination) GetLimit() int {
	if p.Limit <= 0 {
		p.Limit = DefaultLimit
	}
	if p.Limit > MaxLimit {
		p.Limit = MaxLimit
	}
	return p.Limit
}

// QueryAdd adds or updates a query parameter in the pagination's
// query map. This is useful for preserving filter parameters
// across paginated requests.
func (p *Pagination) QueryAdd(key, value string) {
	if p.Query == nil {
		p.Query = make(map[string]string)
	}
	p.Query[key] = value
}

// QueryHas checks if a query parameter exists in the pagination's query map.
func (p *Pagination) QueryHas(key string) bool {
	if p.Query == nil {
		return false
	}
	_, ok := p.Query[key]
	return ok
}

// QueryGet retrieves a query parameter value from the pagination's query map.
// Returns an empty string if the key does not exist.
func (p *Pagination) QueryGet(key string) string {
	if p.Query == nil {
		return ""
	}
	return p.Query[key]
}

// ToQueryStr encodes the pagination state and additional query parameters
// into a URL query string. This can be used to generate links to other pages.
func (p *Pagination) ToQueryStr() string {
	query := url.Values{}
	query.Set("limit", strconv.Itoa(p.GetLimit()))
	query.Set("page", strconv.Itoa(p.GetPage()))

	// add other query parameters
	if p.Query != nil {
		for k, v := range p.Query {
			query.Set(k, v)
		}
	}

	return query.Encode()
}

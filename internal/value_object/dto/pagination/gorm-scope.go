package pagination

import (
	"math"

	"gorm.io/gorm"
)

// Paginate creates a GORM scope function that applies pagination to a query.
// It first counts the total rows matching the model, then returns a scope
// that applies OFFSET and LIMIT to the query.
//
// The model parameter should be a pointer to the model type being queried.
// The pg parameter holds the pagination state and will be updated with
// TotalRows and TotalPages after the count query.
// The db parameter is used to perform the count query.
//
// Returns the GORM scope function and any error from the count query.
// If the count query fails, the scope is still returned but TotalRows
// and TotalPages will be zero.
//
// Example usage:
//
//	pg := pagination.NewPagination(r.URL.Query())
//	var users []User
//	db.Scopes(pagination.Paginate(&User{}, pg, db)).Find(&users)
//	pg.Rows = users
func Paginate(model any, pg *Pagination, db *gorm.DB) func(*gorm.DB) *gorm.DB {
	var totalRows int64

	_ = db.
		Model(model).
		Count(&totalRows).
		Error

	pg.TotalRows = totalRows

	totalPages := int(math.Ceil(float64(totalRows) / float64(pg.GetLimit())))
	pg.TotalPages = totalPages

	return func(db *gorm.DB) *gorm.DB {
		return db.
			Offset(pg.GetOffset()).
			Limit(pg.GetLimit())
	}
}

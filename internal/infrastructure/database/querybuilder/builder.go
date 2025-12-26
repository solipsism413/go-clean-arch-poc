// Package querybuilder provides a flexible query builder using Squirrel for dynamic SQL queries.
package querybuilder

import (
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
)

// Psql is a StatementBuilder configured for PostgreSQL.
var Psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

// FilterOptions represents common filtering options.
type FilterOptions struct {
	Search   string
	SortBy   string
	SortDesc bool
	Page     int
	PageSize int
	Filters  map[string]any
}

// DefaultFilterOptions returns default filter options.
func DefaultFilterOptions() FilterOptions {
	return FilterOptions{
		Page:     1,
		PageSize: 20,
		Filters:  make(map[string]any),
	}
}

// Paginate adds LIMIT and OFFSET to a SelectBuilder for pagination.
func Paginate(q sq.SelectBuilder, page, pageSize int) sq.SelectBuilder {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize
	return q.Limit(uint64(pageSize)).Offset(uint64(offset))
}

// OrderBySafe adds ORDER BY with column validation to a SelectBuilder.
func OrderBySafe(q sq.SelectBuilder, column string, desc bool, allowedColumns []string) sq.SelectBuilder {
	if column == "" {
		return q
	}

	// Validate column name
	isAllowed := false
	for _, allowed := range allowedColumns {
		if strings.EqualFold(column, allowed) {
			column = allowed // Use the canonical name
			isAllowed = true
			break
		}
	}

	if !isAllowed {
		return q
	}

	direction := "ASC"
	if desc {
		direction = "DESC"
	}
	return q.OrderBy(fmt.Sprintf("%s %s", column, direction))
}

// ApplyFilters applies common filters to a SelectBuilder.
func ApplyFilters(q sq.SelectBuilder, opts FilterOptions, searchColumns []string, allowedSortColumns []string) sq.SelectBuilder {
	// Apply search
	if opts.Search != "" && len(searchColumns) > 0 {
		or := sq.Or{}
		for _, col := range searchColumns {
			or = append(or, sq.ILike{col: "%" + opts.Search + "%"})
		}
		q = q.Where(or)
	}

	// Apply dynamic filters
	for column, value := range opts.Filters {
		if value != nil {
			q = q.Where(sq.Eq{column: value})
		}
	}

	// Apply sorting
	if opts.SortBy != "" {
		q = OrderBySafe(q, opts.SortBy, opts.SortDesc, allowedSortColumns)
	}

	// Apply pagination
	q = Paginate(q, opts.Page, opts.PageSize)

	return q
}

// WhereNotEmpty adds a WHERE condition only if the string is not empty.
func WhereNotEmpty(q sq.SelectBuilder, column string, value string) sq.SelectBuilder {
	if value != "" {
		return q.Where(sq.Eq{column: value})
	}
	return q
}

// WhereILike adds a case-insensitive LIKE condition (PostgreSQL) only if the value is not empty.
func WhereILike(q sq.SelectBuilder, column string, value string) sq.SelectBuilder {
	if value != "" {
		return q.Where(sq.ILike{column: "%" + value + "%"})
	}
	return q
}

// Count returns a modified SelectBuilder for counting.
func Count(q sq.SelectBuilder) sq.SelectBuilder {
	return q.RemoveLimit().RemoveOffset().RemoveColumns().Columns("COUNT(*)")
}

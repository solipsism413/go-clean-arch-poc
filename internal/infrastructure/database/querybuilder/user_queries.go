// Package querybuilder provides user-specific query builder.
package querybuilder

import (
	"context"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/handiism/go-clean-arch-poc/internal/application/dto"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
)

// Allowed columns for user sorting
var UserAllowedSortColumns = []string{
	"name",
	"email",
	"created_at",
	"updated_at",
}

// UserQueryBuilder provides methods for building user queries.
type UserQueryBuilder struct {
	pool *pgxpool.Pool
}

// NewUserQueryBuilder creates a new user query builder.
func NewUserQueryBuilder(pool *pgxpool.Pool) *UserQueryBuilder {
	return &UserQueryBuilder{
		pool: pool,
	}
}

// UserRow represents a row from the users table.
type UserRow struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	Name         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// FindWithFilter finds users matching the given filter and pagination.
func (uqb *UserQueryBuilder) FindWithFilter(ctx context.Context, filter dto.UserFilter, pagination dto.Pagination) ([]*entity.User, int64, error) {
	// Build count query
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	countQuery := psql.Select("COUNT(*)").From("users")

	if filter.Search != "" {
		countQuery = countQuery.Where(sq.ILike{"name": "%" + filter.Search + "%"})
	}
	if filter.RoleID != nil {
		countQuery = countQuery.Where("id IN (SELECT user_id FROM user_roles WHERE role_id = ?)", *filter.RoleID)
	}

	countSQL, countArgs, err := countQuery.ToSql()
	if err != nil {
		return nil, 0, err
	}

	var total int64
	err = uqb.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []*entity.User{}, 0, nil
	}

	// Build select query
	selectQuery := psql.Select(
		"id", "email", "password_hash", "name", "created_at", "updated_at",
	).From("users")

	if filter.Search != "" {
		selectQuery = selectQuery.Where(sq.ILike{"name": "%" + filter.Search + "%"})
	}
	if filter.RoleID != nil {
		selectQuery = selectQuery.Where("id IN (SELECT user_id FROM user_roles WHERE role_id = ?)", *filter.RoleID)
	}

	// Apply sorting
	sortBy := pagination.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}

	isAllowedSort := false
	for _, allowed := range UserAllowedSortColumns {
		if strings.EqualFold(sortBy, allowed) {
			sortBy = allowed
			isAllowedSort = true
			break
		}
	}

	if isAllowedSort {
		direction := "ASC"
		if pagination.SortDesc {
			direction = "DESC"
		}
		selectQuery = selectQuery.OrderBy(fmt.Sprintf("%s %s", sortBy, direction))
	}

	// Apply pagination
	page := pagination.Page
	if page < 1 {
		page = 1
	}
	pageSize := pagination.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize
	selectQuery = selectQuery.Limit(uint64(pageSize)).Offset(uint64(offset))

	selectSQL, selectArgs, err := selectQuery.ToSql()
	if err != nil {
		return nil, 0, err
	}

	rows, err := uqb.pool.Query(ctx, selectSQL, selectArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	users, err := scanUsers(rows)
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// Search performs a search on users by name or email.
func (uqb *UserQueryBuilder) Search(ctx context.Context, query string, pagination dto.Pagination) ([]*entity.User, int64, error) {
	// Count query
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	countQuery := psql.Select("COUNT(*)").From("users")
	if query != "" {
		or := sq.Or{
			sq.ILike{"name": "%" + query + "%"},
			sq.ILike{"email": "%" + query + "%"},
		}
		countQuery = countQuery.Where(or)
	}

	countSQL, countArgs, err := countQuery.ToSql()
	if err != nil {
		return nil, 0, err
	}

	var total int64
	err = uqb.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []*entity.User{}, 0, nil
	}

	// Select query
	selectQuery := psql.Select(
		"id", "email", "password_hash", "name", "created_at", "updated_at",
	).From("users")

	if query != "" {
		or := sq.Or{
			sq.ILike{"name": "%" + query + "%"},
			sq.ILike{"email": "%" + query + "%"},
		}
		selectQuery = selectQuery.Where(or)
	}

	// Apply sorting
	sortBy := pagination.SortBy
	if sortBy != "" {
		isAllowedSort := false
		for _, allowed := range UserAllowedSortColumns {
			if strings.EqualFold(sortBy, allowed) {
				sortBy = allowed
				isAllowedSort = true
				break
			}
		}
		if isAllowedSort {
			direction := "ASC"
			if pagination.SortDesc {
				direction = "DESC"
			}
			selectQuery = selectQuery.OrderBy(fmt.Sprintf("%s %s", sortBy, direction))
		}
	}

	// Apply pagination
	page := pagination.Page
	if page < 1 {
		page = 1
	}
	pageSize := pagination.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize
	selectQuery = selectQuery.Limit(uint64(pageSize)).Offset(uint64(offset))

	selectSQL, selectArgs, err := selectQuery.ToSql()
	if err != nil {
		return nil, 0, err
	}

	rows, err := uqb.pool.Query(ctx, selectSQL, selectArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	users, err := scanUsers(rows)
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func scanUsers(rows pgx.Rows) ([]*entity.User, error) {
	var users []*entity.User

	for rows.Next() {
		var row UserRow
		err := rows.Scan(
			&row.ID,
			&row.Email,
			&row.PasswordHash,
			&row.Name,
			&row.CreatedAt,
			&row.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		user := &entity.User{
			ID:           row.ID,
			Email:        row.Email,
			PasswordHash: row.PasswordHash,
			Name:         row.Name,
			CreatedAt:    row.CreatedAt,
			UpdatedAt:    row.UpdatedAt,
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

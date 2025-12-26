// Package querybuilder provides user-specific query builder.
package querybuilder

import (
	"context"
	"time"

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
	countQuery := Psql.Select("COUNT(*)").From("users")

	if filter.Search != "" {
		countQuery = WhereILike(countQuery, "name", filter.Search)
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
	selectQuery := Psql.Select(
		"id", "email", "password_hash", "name", "created_at", "updated_at",
	).From("users")

	if filter.Search != "" {
		selectQuery = WhereILike(selectQuery, "name", filter.Search)
	}
	if filter.RoleID != nil {
		selectQuery = selectQuery.Where("id IN (SELECT user_id FROM user_roles WHERE role_id = ?)", *filter.RoleID)
	}

	// Apply sorting
	sortBy := pagination.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	selectQuery = OrderBySafe(selectQuery, sortBy, pagination.SortDesc, UserAllowedSortColumns)

	// Apply pagination
	selectQuery = Paginate(selectQuery, pagination.Page, pagination.PageSize)

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
	searchColumns := []string{"name", "email"}

	opts := FilterOptions{
		Search:   query,
		SortBy:   pagination.SortBy,
		SortDesc: pagination.SortDesc,
		Page:     pagination.Page,
		PageSize: pagination.PageSize,
	}

	// Count query
	countQuery := Psql.Select("COUNT(*)").From("users")
	if query != "" {
		countQuery = WhereILike(countQuery, "name", query)
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
	selectQuery := Psql.Select(
		"id", "email", "password_hash", "name", "created_at", "updated_at",
	).From("users")

	selectQuery = ApplyFilters(selectQuery, opts, searchColumns, UserAllowedSortColumns)

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

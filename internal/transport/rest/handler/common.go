package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/handiism/go-clean-arch-poc/internal/application/dto"
	"github.com/handiism/go-clean-arch-poc/internal/application/validation"
	"github.com/handiism/go-clean-arch-poc/internal/domain/entity"
	domainerror "github.com/handiism/go-clean-arch-poc/internal/domain/error"
	"github.com/handiism/go-clean-arch-poc/internal/domain/valueobject"
	"github.com/handiism/go-clean-arch-poc/internal/transport/rest/presenter"
)

// handleError maps domain and validation errors to HTTP responses.
func handleError(w http.ResponseWriter, err error) {
	if validationErr, ok := err.(*validation.ValidationError); ok {
		presenter.ValidationError(w, validationErr)
		return
	}

	if domainerror.IsNotFoundError(err) {
		presenter.Error(w, http.StatusNotFound, err.Error(), nil)
		return
	}

	if domainerror.IsValidationError(err) {
		presenter.Error(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	if domainerror.IsConflictError(err) {
		presenter.Error(w, http.StatusConflict, err.Error(), nil)
		return
	}

	if errors.Is(err, entity.ErrInvalidStatus) || errors.Is(err, entity.ErrInvalidStatusTransition) ||
		errors.Is(err, entity.ErrTaskArchived) || errors.Is(err, entity.ErrTaskNotDone) ||
		errors.Is(err, valueobject.ErrInvalidTaskStatus) {
		presenter.Error(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	if domainerror.IsUnauthorizedError(err) {
		presenter.Error(w, http.StatusUnauthorized, err.Error(), nil)
		return
	}

	if domainerror.IsForbiddenError(err) {
		presenter.Error(w, http.StatusForbidden, err.Error(), nil)
		return
	}

	presenter.Error(w, http.StatusInternalServerError, "Internal server error", err)
}

func parsePagination(r *http.Request) (dto.Pagination, error) {
	pagination := dto.DefaultPagination()

	if page := r.URL.Query().Get("page"); page != "" {
		p, err := parsePositiveInt(page, "page")
		if err != nil {
			return dto.Pagination{}, err
		}
		pagination.Page = p
	}

	if pageSize := r.URL.Query().Get("pageSize"); pageSize != "" {
		ps, err := parsePositiveInt(pageSize, "pageSize")
		if err != nil {
			return dto.Pagination{}, err
		}
		if ps > 100 {
			return dto.Pagination{}, fmt.Errorf("pageSize must be between 1 and 100")
		}
		pagination.PageSize = ps
	}

	if sortBy := r.URL.Query().Get("sortBy"); sortBy != "" {
		pagination.SortBy = sortBy
	}

	if sortDesc := r.URL.Query().Get("sortDesc"); sortDesc != "" {
		parsedSortDesc, err := strconv.ParseBool(sortDesc)
		if err != nil {
			return dto.Pagination{}, fmt.Errorf("sortDesc must be a boolean")
		}
		pagination.SortDesc = parsedSortDesc
	}

	return pagination, nil
}

func parsePositiveInt(s, field string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("%s must be a positive integer", field)
	}
	if n <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", field)
	}
	return n, nil
}

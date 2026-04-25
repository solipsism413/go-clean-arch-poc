package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/handiism/go-clean-arch-poc/internal/application/dto"
	"github.com/handiism/go-clean-arch-poc/internal/application/port/input"
	"github.com/handiism/go-clean-arch-poc/internal/transport/rest/presenter"
)

type LabelHandler struct {
	labelService input.LabelService
	logger       *slog.Logger
}

func NewLabelHandler(labelService input.LabelService, logger *slog.Logger) *LabelHandler {
	return &LabelHandler{labelService: labelService, logger: logger}
}

// Create handles POST /labels
// @Summary Create label
// @Description Create a new label
// @Tags labels
// @Accept json
// @Produce json
// @Param label body dto.CreateLabelInput true "Label details"
// @Success 201 {object} presenter.Response{data=dto.LabelOutput}
// @Failure 400 {object} presenter.ErrorResponse
// @Failure 401 {object} presenter.ErrorResponse
// @Failure 403 {object} presenter.ErrorResponse
// @Failure 409 {object} presenter.ErrorResponse
// @Failure 500 {object} presenter.ErrorResponse
// @Security BearerAuth
// @Router /labels [post]
func (h *LabelHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input dto.CreateLabelInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	label, err := h.labelService.CreateLabel(r.Context(), input)
	if err != nil {
		handleError(w, err)
		return
	}

	presenter.Created(w, label)
}

// List handles GET /labels
// @Summary List labels
// @Description Get all available labels
// @Tags labels
// @Accept json
// @Produce json
// @Success 200 {object} presenter.Response{data=[]dto.LabelOutput}
// @Failure 401 {object} presenter.ErrorResponse
// @Failure 500 {object} presenter.ErrorResponse
// @Security BearerAuth
// @Router /labels [get]
func (h *LabelHandler) List(w http.ResponseWriter, r *http.Request) {
	labels, err := h.labelService.ListLabels(r.Context())
	if err != nil {
		handleError(w, err)
		return
	}

	presenter.OK(w, labels)
}

// Get handles GET /labels/{id}
// @Summary Get label
// @Description Get a label by ID
// @Tags labels
// @Accept json
// @Produce json
// @Param id path string true "Label ID" format(uuid)
// @Success 200 {object} presenter.Response{data=dto.LabelOutput}
// @Failure 400 {object} presenter.ErrorResponse
// @Failure 401 {object} presenter.ErrorResponse
// @Failure 404 {object} presenter.ErrorResponse
// @Failure 500 {object} presenter.ErrorResponse
// @Security BearerAuth
// @Router /labels/{id} [get]
func (h *LabelHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid label ID", err)
		return
	}

	label, err := h.labelService.GetLabel(r.Context(), id)
	if err != nil {
		handleError(w, err)
		return
	}

	presenter.OK(w, label)
}

// Update handles PUT /labels/{id}
// @Summary Update label
// @Description Update an existing label
// @Tags labels
// @Accept json
// @Produce json
// @Param id path string true "Label ID" format(uuid)
// @Param label body dto.UpdateLabelInput true "Label updates"
// @Success 200 {object} presenter.Response{data=dto.LabelOutput}
// @Failure 400 {object} presenter.ErrorResponse
// @Failure 401 {object} presenter.ErrorResponse
// @Failure 403 {object} presenter.ErrorResponse
// @Failure 404 {object} presenter.ErrorResponse
// @Failure 409 {object} presenter.ErrorResponse
// @Failure 500 {object} presenter.ErrorResponse
// @Security BearerAuth
// @Router /labels/{id} [put]
func (h *LabelHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid label ID", err)
		return
	}

	var input dto.UpdateLabelInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	label, err := h.labelService.UpdateLabel(r.Context(), id, input)
	if err != nil {
		handleError(w, err)
		return
	}

	presenter.OK(w, label)
}

// Delete handles DELETE /labels/{id}
// @Summary Delete label
// @Description Delete a label by ID
// @Tags labels
// @Accept json
// @Produce json
// @Success 204 "No Content"
// @Failure 400 {object} presenter.ErrorResponse
// @Failure 401 {object} presenter.ErrorResponse
// @Failure 403 {object} presenter.ErrorResponse
// @Failure 404 {object} presenter.ErrorResponse
// @Failure 500 {object} presenter.ErrorResponse
// @Security BearerAuth
// @Router /labels/{id} [delete]
func (h *LabelHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		presenter.Error(w, http.StatusBadRequest, "Invalid label ID", err)
		return
	}

	if err := h.labelService.DeleteLabel(r.Context(), id); err != nil {
		handleError(w, err)
		return
	}

	presenter.NoContent(w)
}

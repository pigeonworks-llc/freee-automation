package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/pigeonworks-llc/freee-emulator/internal/models"
	"github.com/pigeonworks-llc/freee-emulator/internal/store"
)

// DealsHandler handles deal-related API endpoints.
type DealsHandler struct {
	store *store.Store
}

// NewDealsHandler creates a new DealsHandler.
func NewDealsHandler(s *store.Store) *DealsHandler {
	return &DealsHandler{store: s}
}

// DealsListResponse represents the response for GET /api/1/deals
type DealsListResponse struct {
	Deals []models.Deal `json:"deals"`
}

// List handles GET /api/1/deals.
// @Summary List deals
// @Description Get list of deals with optional company_id filter
// @Tags deals
// @Accept json
// @Produce json
// @Param company_id query int false "Company ID to filter deals"
// @Success 200 {object} DealsListResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /deals [get]
// @Security BearerAuth
func (h *DealsHandler) List(w http.ResponseWriter, r *http.Request) {
	companyIDStr := r.URL.Query().Get("company_id")
	var companyID *int64

	if companyIDStr != "" {
		id, err := strconv.ParseInt(companyIDStr, 10, 64)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid_parameter", "Invalid company_id")
			return
		}
		companyID = &id
	}

	deals, err := h.store.ListDeals(companyID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "server_error", "Failed to list deals")
		return
	}

	response := map[string]interface{}{
		"deals": deals,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

// Get handles GET /api/1/deals/{id}.
func (h *DealsHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_parameter", "Invalid deal ID")
		return
	}

	deal, err := h.store.GetDeal(id)
	if err != nil {
		if err == store.ErrNotFound {
			writeJSONError(w, http.StatusNotFound, "not_found", "Deal not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "server_error", "Failed to get deal")
		return
	}

	response := map[string]interface{}{
		"deal": deal,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

// Create handles POST /api/1/deals.
func (h *DealsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateDealRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "Failed to parse request body")
		return
	}

	// Validate required fields.
	if req.CompanyID == 0 {
		writeJSONError(w, http.StatusBadRequest, "invalid_parameter", "Missing company_id")
		return
	}
	if req.IssueDate == "" {
		writeJSONError(w, http.StatusBadRequest, "invalid_parameter", "Missing issue_date")
		return
	}
	if req.Type == "" {
		writeJSONError(w, http.StatusBadRequest, "invalid_parameter", "Missing type")
		return
	}
	if len(req.Details) == 0 {
		writeJSONError(w, http.StatusBadRequest, "invalid_parameter", "Missing details")
		return
	}

	deal, err := h.store.CreateDeal(&req)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "server_error", "Failed to create deal")
		return
	}

	response := map[string]interface{}{
		"deal": deal,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(response)
}

// Update handles PUT /api/1/deals/{id}.
func (h *DealsHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_parameter", "Invalid deal ID")
		return
	}

	var req models.UpdateDealRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "Failed to parse request body")
		return
	}

	deal, err := h.store.UpdateDeal(id, &req)
	if err != nil {
		if err == store.ErrNotFound {
			writeJSONError(w, http.StatusNotFound, "not_found", "Deal not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "server_error", "Failed to update deal")
		return
	}

	response := map[string]interface{}{
		"deal": deal,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

// Delete handles DELETE /api/1/deals/{id}.
func (h *DealsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_parameter", "Invalid deal ID")
		return
	}

	if err := h.store.DeleteDeal(id); err != nil {
		if err == store.ErrNotFound {
			writeJSONError(w, http.StatusNotFound, "not_found", "Deal not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "server_error", "Failed to delete deal")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

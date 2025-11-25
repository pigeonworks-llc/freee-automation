package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/pigeonworks-llc/freee-emulator/internal/models"
	"github.com/pigeonworks-llc/freee-emulator/internal/store"
)

// JournalsHandler handles journal-related API endpoints.
type JournalsHandler struct {
	store *store.Store
}

// NewJournalsHandler creates a new JournalsHandler.
func NewJournalsHandler(s *store.Store) *JournalsHandler {
	return &JournalsHandler{store: s}
}

// List handles GET /api/1/journals.
func (h *JournalsHandler) List(w http.ResponseWriter, r *http.Request) {
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

	journals, err := h.store.ListJournals(companyID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "server_error", "Failed to list journals")
		return
	}

	response := map[string]interface{}{
		"journals": journals,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

// Get handles GET /api/1/journals/{id}.
func (h *JournalsHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_parameter", "Invalid journal ID")
		return
	}

	journal, err := h.store.GetJournal(id)
	if err != nil {
		if err == store.ErrNotFound {
			writeJSONError(w, http.StatusNotFound, "not_found", "Journal not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "server_error", "Failed to get journal")
		return
	}

	response := map[string]interface{}{
		"journal": journal,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

// Create handles POST /api/1/journals.
func (h *JournalsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateJournalRequest
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
	if len(req.Details) == 0 {
		writeJSONError(w, http.StatusBadRequest, "invalid_parameter", "Missing details")
		return
	}

	journal, err := h.store.CreateJournal(&req)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "server_error", "Failed to create journal")
		return
	}

	response := map[string]interface{}{
		"journal": journal,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(response)
}

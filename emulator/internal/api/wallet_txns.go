package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/pigeonworks-llc/freee-emulator/internal/models"
	"github.com/pigeonworks-llc/freee-emulator/internal/store"
)

// WalletTxnsHandler handles wallet transaction-related API endpoints.
type WalletTxnsHandler struct {
	store *store.Store
}

// NewWalletTxnsHandler creates a new WalletTxnsHandler.
func NewWalletTxnsHandler(s *store.Store) *WalletTxnsHandler {
	return &WalletTxnsHandler{store: s}
}

// List handles GET /api/1/wallet_txns.
func (h *WalletTxnsHandler) List(w http.ResponseWriter, r *http.Request) {
	companyIDStr := r.URL.Query().Get("company_id")
	statusStr := r.URL.Query().Get("status")

	var companyID *int64
	var status *string

	if companyIDStr != "" {
		id, err := strconv.ParseInt(companyIDStr, 10, 64)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid_parameter", "Invalid company_id")
			return
		}
		companyID = &id
	}

	if statusStr != "" {
		// Convert numeric status to string status
		// freee API: 1 = 消込待ち(unbooked), 2 = 消込済み(booked/settled)
		convertedStatus := statusStr
		switch statusStr {
		case "1":
			convertedStatus = "unbooked"
		case "2":
			convertedStatus = "settled"
		}
		status = &convertedStatus
	}

	walletTxns, err := h.store.ListWalletTxns(companyID, status)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "server_error", "Failed to list wallet transactions")
		return
	}

	response := map[string]interface{}{
		"wallet_txns": walletTxns,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

// Get handles GET /api/1/wallet_txns/{id}.
func (h *WalletTxnsHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_parameter", "Invalid wallet transaction ID")
		return
	}

	walletTxn, err := h.store.GetWalletTxn(id)
	if err != nil {
		if err == store.ErrNotFound {
			writeJSONError(w, http.StatusNotFound, "not_found", "Wallet transaction not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "server_error", "Failed to get wallet transaction")
		return
	}

	response := map[string]interface{}{
		"wallet_txn": walletTxn,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

// Create handles POST /api/1/wallet_txns.
func (h *WalletTxnsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateWalletTxnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "Failed to parse request body")
		return
	}

	// Validate required fields
	if req.CompanyID == 0 {
		writeJSONError(w, http.StatusBadRequest, "invalid_parameter", "Missing company_id")
		return
	}
	if req.Date == "" {
		writeJSONError(w, http.StatusBadRequest, "invalid_parameter", "Missing date")
		return
	}
	if req.WalletableType == "" {
		writeJSONError(w, http.StatusBadRequest, "invalid_parameter", "Missing walletable_type")
		return
	}

	walletTxn, err := h.store.CreateWalletTxn(&req)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "server_error", "Failed to create wallet transaction")
		return
	}

	response := map[string]interface{}{
		"wallet_txn": walletTxn,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(response)
}

// Update handles PUT /api/1/wallet_txns/{id}.
func (h *WalletTxnsHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_parameter", "Invalid wallet transaction ID")
		return
	}

	var req models.UpdateWalletTxnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "Failed to parse request body")
		return
	}

	walletTxn, err := h.store.UpdateWalletTxn(id, &req)
	if err != nil {
		if err == store.ErrNotFound {
			writeJSONError(w, http.StatusNotFound, "not_found", "Wallet transaction not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "server_error", "Failed to update wallet transaction")
		return
	}

	response := map[string]interface{}{
		"wallet_txn": walletTxn,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

// Delete handles DELETE /api/1/wallet_txns/{id}.
func (h *WalletTxnsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_parameter", "Invalid wallet transaction ID")
		return
	}

	if err := h.store.DeleteWalletTxn(id); err != nil {
		if err == store.ErrNotFound {
			writeJSONError(w, http.StatusNotFound, "not_found", "Wallet transaction not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "server_error", "Failed to delete wallet transaction")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

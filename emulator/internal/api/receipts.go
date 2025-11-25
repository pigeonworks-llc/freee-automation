package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/pigeonworks-llc/freee-emulator/internal/models"
	"github.com/pigeonworks-llc/freee-emulator/internal/store"
)

// ReceiptsHandler handles receipt-related API requests.
type ReceiptsHandler struct {
	store     *store.Store
	uploadDir string
}

// NewReceiptsHandler creates a new ReceiptsHandler.
func NewReceiptsHandler(s *store.Store, uploadDir string) *ReceiptsHandler {
	return &ReceiptsHandler{
		store:     s,
		uploadDir: uploadDir,
	}
}

// List handles GET /api/1/receipts
func (h *ReceiptsHandler) List(w http.ResponseWriter, r *http.Request) {
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

	receipts, err := h.store.ListReceipts(companyID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "server_error", "Failed to list receipts")
		return
	}

	response := map[string]interface{}{
		"receipts": receipts,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

// Create handles POST /api/1/receipts
func (h *ReceiptsHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB max
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "Failed to parse multipart form")
		return
	}

	// Extract company_id
	companyIDStr := r.FormValue("company_id")
	if companyIDStr == "" {
		writeJSONError(w, http.StatusBadRequest, "invalid_parameter", "Missing company_id")
		return
	}
	companyID, err := strconv.ParseInt(companyIDStr, 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_parameter", "Invalid company_id")
		return
	}

	// Extract issue_date
	issueDate := r.FormValue("issue_date")
	if issueDate == "" {
		writeJSONError(w, http.StatusBadRequest, "invalid_parameter", "Missing issue_date")
		return
	}

	// Extract description
	description := r.FormValue("description")

	// Extract file
	file, fileHeader, err := r.FormFile("receipt")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_parameter", "Missing receipt file")
		return
	}
	defer file.Close()

	// Create upload directory if it doesn't exist
	uploadPath := filepath.Join(h.uploadDir, fmt.Sprintf("%d", companyID))
	if err := os.MkdirAll(uploadPath, 0o755); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "server_error", "Failed to create upload directory")
		return
	}

	// Generate temporary filename (will be renamed after getting receipt ID)
	tempFileName := fmt.Sprintf("temp_%s", fileHeader.Filename)
	tempFilePath := filepath.Join(uploadPath, tempFileName)

	// Save file temporarily
	destFile, err := os.Create(tempFilePath)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "server_error", "Failed to create file")
		return
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, file); err != nil {
		_ = os.Remove(tempFilePath)
		writeJSONError(w, http.StatusInternalServerError, "server_error", "Failed to save file")
		return
	}

	// Create receipt record
	req := &models.CreateReceiptRequest{
		CompanyID:   companyID,
		IssueDate:   issueDate,
		Description: description,
		FileName:    fileHeader.Filename,
		FilePath:    "", // Will be set after renaming
	}

	receipt, err := h.store.CreateReceipt(req)
	if err != nil {
		_ = os.Remove(tempFilePath)
		writeJSONError(w, http.StatusInternalServerError, "server_error", "Failed to create receipt")
		return
	}

	// Rename file with receipt ID
	finalFileName := fmt.Sprintf("%d.pdf", receipt.ID)
	finalFilePath := filepath.Join(uploadPath, finalFileName)
	if err := os.Rename(tempFilePath, finalFilePath); err != nil {
		_ = os.Remove(tempFilePath)
		writeJSONError(w, http.StatusInternalServerError, "server_error", "Failed to rename file")
		return
	}

	// Update receipt with final file path
	receipt.FilePath = finalFilePath

	response := map[string]interface{}{
		"receipt": receipt,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(response)
}

// Get handles GET /api/1/receipts/{id}
func (h *ReceiptsHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_parameter", "Invalid receipt ID")
		return
	}

	receipt, err := h.store.GetReceipt(id)
	if err != nil {
		if err == store.ErrNotFound {
			writeJSONError(w, http.StatusNotFound, "not_found", "Receipt not found")
		} else {
			writeJSONError(w, http.StatusInternalServerError, "server_error", "Failed to get receipt")
		}
		return
	}

	response := map[string]interface{}{
		"receipt": receipt,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

// Delete handles DELETE /api/1/receipts/{id}
func (h *ReceiptsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_parameter", "Invalid receipt ID")
		return
	}

	// Get receipt to find file path
	receipt, err := h.store.GetReceipt(id)
	if err != nil {
		if err == store.ErrNotFound {
			writeJSONError(w, http.StatusNotFound, "not_found", "Receipt not found")
		} else {
			writeJSONError(w, http.StatusInternalServerError, "server_error", "Failed to get receipt")
		}
		return
	}

	// Delete file
	if receipt.FilePath != "" {
		_ = os.Remove(receipt.FilePath)
	}

	// Delete receipt record
	if err := h.store.DeleteReceipt(id); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "server_error", "Failed to delete receipt")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

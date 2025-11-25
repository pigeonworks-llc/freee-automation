package api

import (
	"encoding/json"
	"net/http"

	"github.com/pigeonworks-llc/freee-emulator/internal/models"
)

// CompaniesHandler handles company-related API endpoints.
type CompaniesHandler struct {
	companies []models.Company
}

// NewCompaniesHandler creates a new CompaniesHandler with default company.
func NewCompaniesHandler() *CompaniesHandler {
	return &CompaniesHandler{
		companies: []models.Company{
			{
				ID:          1,
				DisplayName: "Pigeonworks LLC",
				Name:        "合同会社Pigeonworks",
				NameKana:    "ゴウドウガイシャピジョンワークス",
			},
		},
	}
}

// List handles GET /api/1/companies.
// @Summary List companies
// @Description Get list of companies accessible by the authenticated user
// @Tags companies
// @Accept json
// @Produce json
// @Success 200 {object} models.CompaniesResponse
// @Failure 500 {object} ErrorResponse
// @Router /companies [get]
// @Security BearerAuth
func (h *CompaniesHandler) List(w http.ResponseWriter, r *http.Request) {
	response := models.CompaniesResponse{
		Companies: h.companies,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

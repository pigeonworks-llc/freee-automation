package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/pigeonworks-llc/freee-emulator/internal/models"
)

// WalletablesHandler handles walletable-related API endpoints.
type WalletablesHandler struct {
	walletables []models.Walletable
}

// NewWalletablesHandler creates a new WalletablesHandler with default walletables.
func NewWalletablesHandler() *WalletablesHandler {
	return &WalletablesHandler{
		walletables: defaultWalletables(),
	}
}

// defaultWalletables returns a list of default walletables for testing.
func defaultWalletables() []models.Walletable {
	bankID := int64(1)
	return []models.Walletable{
		// Bank accounts
		{
			ID:                1,
			Name:              "GMOあおぞらネット銀行",
			Type:              "bank_account",
			BankID:            &bankID,
			LastBalance:       1000000,
			WalletableBalance: 1000000,
		},
		// Credit cards
		{
			ID:                2,
			Name:              "アメリカン・エキスプレス",
			Type:              "credit_card",
			BankID:            nil,
			LastBalance:       -50000,
			WalletableBalance: -50000,
		},
		{
			ID:                3,
			Name:              "三井住友カード",
			Type:              "credit_card",
			BankID:            nil,
			LastBalance:       -30000,
			WalletableBalance: -30000,
		},
		// Wallet (cash)
		{
			ID:                4,
			Name:              "現金",
			Type:              "wallet",
			BankID:            nil,
			LastBalance:       50000,
			WalletableBalance: 50000,
		},
	}
}

// List handles GET /api/1/walletables.
// @Summary List walletables
// @Description Get list of walletables (bank accounts, credit cards, wallets) for a company
// @Tags walletables
// @Accept json
// @Produce json
// @Param company_id query int true "Company ID"
// @Param type query string false "Filter by type (bank_account, credit_card, wallet)"
// @Success 200 {object} models.WalletablesResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /walletables [get]
// @Security BearerAuth
func (h *WalletablesHandler) List(w http.ResponseWriter, r *http.Request) {
	companyIDStr := r.URL.Query().Get("company_id")
	if companyIDStr == "" {
		writeJSONError(w, http.StatusBadRequest, "invalid_parameter", "company_id is required")
		return
	}

	_, err := strconv.ParseInt(companyIDStr, 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_parameter", "Invalid company_id")
		return
	}

	// Filter by type if specified
	typeFilter := r.URL.Query().Get("type")
	var filtered []models.Walletable
	if typeFilter != "" {
		for _, w := range h.walletables {
			if w.Type == typeFilter {
				filtered = append(filtered, w)
			}
		}
	} else {
		filtered = h.walletables
	}

	response := models.WalletablesResponse{
		Walletables: filtered,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

// GetByID returns a walletable by ID.
func (h *WalletablesHandler) GetByID(id int64) *models.Walletable {
	for _, w := range h.walletables {
		if w.ID == id {
			return &w
		}
	}
	return nil
}

// GetByType returns walletables by type.
func (h *WalletablesHandler) GetByType(walletableType string) []models.Walletable {
	var result []models.Walletable
	for _, w := range h.walletables {
		if w.Type == walletableType {
			result = append(result, w)
		}
	}
	return result
}

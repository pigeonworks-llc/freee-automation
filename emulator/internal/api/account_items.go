package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/pigeonworks-llc/freee-emulator/internal/models"
)

// AccountItemsHandler handles account item-related API endpoints.
type AccountItemsHandler struct {
	accountItems []models.AccountItem
}

// NewAccountItemsHandler creates a new AccountItemsHandler with default account items.
func NewAccountItemsHandler() *AccountItemsHandler {
	return &AccountItemsHandler{
		accountItems: defaultAccountItems(),
	}
}

// defaultAccountItems returns a list of common account items for testing.
func defaultAccountItems() []models.AccountItem {
	return []models.AccountItem{
		// Assets (資産)
		{ID: 101, Name: "現金", AccountCategory: "asset", DefaultTaxCode: 0},
		{ID: 102, Name: "普通預金", AccountCategory: "asset", DefaultTaxCode: 0},
		{ID: 103, Name: "売掛金", AccountCategory: "asset", DefaultTaxCode: 0},

		// Liabilities (負債)
		{ID: 201, Name: "買掛金", AccountCategory: "liability", DefaultTaxCode: 0},
		{ID: 202, Name: "未払金", AccountCategory: "liability", DefaultTaxCode: 0},
		{ID: 203, Name: "クレジットカード", AccountCategory: "liability", DefaultTaxCode: 0},

		// Income (収益)
		{ID: 401, Name: "売上高", AccountCategory: "income", DefaultTaxCode: 21},

		// Expenses (費用)
		{ID: 501, Name: "仕入高", AccountCategory: "expense", DefaultTaxCode: 136},
		{ID: 502, Name: "新聞図書費", AccountCategory: "expense", DefaultTaxCode: 136},
		{ID: 503, Name: "研修費", AccountCategory: "expense", DefaultTaxCode: 136},
		{ID: 504, Name: "消耗品費", AccountCategory: "expense", DefaultTaxCode: 136},
		{ID: 505, Name: "通信費", AccountCategory: "expense", DefaultTaxCode: 136},
		{ID: 506, Name: "支払手数料", AccountCategory: "expense", DefaultTaxCode: 136},
		{ID: 507, Name: "旅費交通費", AccountCategory: "expense", DefaultTaxCode: 136},
		{ID: 508, Name: "接待交際費", AccountCategory: "expense", DefaultTaxCode: 136},
		{ID: 509, Name: "雑費", AccountCategory: "expense", DefaultTaxCode: 136},
		{ID: 510, Name: "広告宣伝費", AccountCategory: "expense", DefaultTaxCode: 136},
		{ID: 511, Name: "地代家賃", AccountCategory: "expense", DefaultTaxCode: 136},
		{ID: 512, Name: "水道光熱費", AccountCategory: "expense", DefaultTaxCode: 136},
		{ID: 513, Name: "保険料", AccountCategory: "expense", DefaultTaxCode: 136},
		{ID: 514, Name: "研究開発費", AccountCategory: "expense", DefaultTaxCode: 136},
	}
}

// List handles GET /api/1/account_items.
// @Summary List account items
// @Description Get list of account items for a company
// @Tags account_items
// @Accept json
// @Produce json
// @Param company_id query int true "Company ID"
// @Success 200 {object} models.AccountItemsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /account_items [get]
// @Security BearerAuth
func (h *AccountItemsHandler) List(w http.ResponseWriter, r *http.Request) {
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

	// Return all account items (in real freee, this would filter by company)
	response := models.AccountItemsResponse{
		AccountItems: h.accountItems,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

// GetByID returns an account item by ID.
func (h *AccountItemsHandler) GetByID(id int64) *models.AccountItem {
	for _, item := range h.accountItems {
		if item.ID == id {
			return &item
		}
	}
	return nil
}

// GetByName returns an account item by name.
func (h *AccountItemsHandler) GetByName(name string) *models.AccountItem {
	for _, item := range h.accountItems {
		if item.Name == name {
			return &item
		}
	}
	return nil
}

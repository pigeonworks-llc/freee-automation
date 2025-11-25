package models

// AccountItem represents an account item (勘定科目) in freee.
type AccountItem struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	AccountCategory string `json:"account_category"` // asset, liability, equity, income, expense
	DefaultTaxCode  int    `json:"default_tax_code"`
}

// AccountItemsResponse represents the response for GET /api/1/account_items
type AccountItemsResponse struct {
	AccountItems []AccountItem `json:"account_items"`
}

package models

// Company represents a company/business entity in freee.
type Company struct {
	ID          int64  `json:"id"`
	DisplayName string `json:"display_name"`
	Name        string `json:"name"`
	NameKana    string `json:"name_kana"`
}

// CompaniesResponse represents the response for GET /api/1/companies
type CompaniesResponse struct {
	Companies []Company `json:"companies"`
}

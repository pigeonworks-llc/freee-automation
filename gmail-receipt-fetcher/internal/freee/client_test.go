package freee

import (
	"testing"
	"time"
)

func TestFilterByVendor(t *testing.T) {
	txns := []WalletTransaction{
		{ID: 1, Date: "2024-01-15", Amount: -1000, EntrySide: "expense", Description: "AMAZON.CO.JP購入"},
		{ID: 2, Date: "2024-01-16", Amount: -2000, EntrySide: "expense", Description: "楽天市場注文"},
		{ID: 3, Date: "2024-01-17", Amount: -3000, EntrySide: "expense", Description: "Other Store"},
		{ID: 4, Date: "2024-01-18", Amount: 5000, EntrySide: "income", Description: "AMAZON返金"},
	}

	tests := []struct {
		name     string
		vendors  []string
		expected int
	}{
		{"single vendor amazon", []string{"amazon"}, 1},
		{"single vendor rakuten", []string{"楽天"}, 1},
		{"multiple vendors", []string{"amazon", "楽天"}, 2},
		{"no match", []string{"yodobashi"}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterByVendor(txns, tt.vendors)
			if len(result) != tt.expected {
				t.Errorf("FilterByVendor() returned %d items, expected %d", len(result), tt.expected)
			}
		})
	}
}

func TestFilterAll(t *testing.T) {
	txns := []WalletTransaction{
		{ID: 1, Date: "2024-01-15", Amount: -1000, EntrySide: "expense", Description: "AMAZON購入"},
		{ID: 2, Date: "2024-01-16", Amount: -2000, EntrySide: "expense", Description: "楽天市場"},
		{ID: 3, Date: "2024-01-17", Amount: 5000, EntrySide: "income", Description: "売上"},
	}

	result := FilterAll(txns)

	if len(result) != 2 {
		t.Errorf("FilterAll() returned %d expense transactions, expected 2", len(result))
	}

	// Check amounts are converted to positive
	for _, txn := range result {
		if txn.Amount < 0 {
			t.Errorf("FilterAll() amount should be positive, got %d", txn.Amount)
		}
	}
}

func TestExtractVendorFromDescription(t *testing.T) {
	tests := []struct {
		desc     string
		expected string
	}{
		{"AMAZON.CO.JP購入", "amazon"},
		{"楽天市場 注文確認", "rakuten"},
		{"ヨドバシカメラ決済", "yodobashi"},
		{"Apple Store購入", "apple"},
		{"Google Play決済", "google"},
		{"不明な取引先", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := extractVendor(tt.desc)
			if result != tt.expected {
				t.Errorf("extractVendor(%q) = %q, expected %q", tt.desc, result, tt.expected)
			}
		})
	}
}

func TestAbs(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{-100, 100},
		{100, 100},
		{0, 0},
		{-1, 1},
	}

	for _, tt := range tests {
		result := abs(tt.input)
		if result != tt.expected {
			t.Errorf("abs(%d) = %d, expected %d", tt.input, result, tt.expected)
		}
	}
}

func TestFilterByVendorDateParsing(t *testing.T) {
	txns := []WalletTransaction{
		{ID: 1, Date: "2024-01-15", Amount: -1000, EntrySide: "expense", Description: "AMAZON購入"},
	}

	result := FilterByVendor(txns, []string{"amazon"})

	if len(result) != 1 {
		t.Fatal("Expected 1 result")
	}

	expected := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	if !result[0].Date.Equal(expected) {
		t.Errorf("Date parsing failed: got %v, expected %v", result[0].Date, expected)
	}
}

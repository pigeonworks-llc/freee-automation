package integration

import (
	"fmt"
	"time"

	"github.com/pigeonworks-llc/freee-emulator/internal/models"
)

// TestDataBuilder provides helper methods for building test data.
type TestDataBuilder struct {
	companyID int64
}

// NewTestDataBuilder creates a new TestDataBuilder.
func NewTestDataBuilder(companyID int64) *TestDataBuilder {
	return &TestDataBuilder{companyID: companyID}
}

// IncomeDeal creates a test income deal request.
func (b *TestDataBuilder) IncomeDeal(amount int64, issueDate string) models.CreateDealRequest {
	if issueDate == "" {
		issueDate = time.Now().Format("2006-01-02")
	}

	return models.CreateDealRequest{
		CompanyID: b.companyID,
		IssueDate: issueDate,
		Type:      "income",
		Details: []models.CreateDetailRequest{
			{
				AccountItemID: 400, // 売上高
				TaxCode:       1,
				Amount:        amount,
			},
		},
	}
}

// ExpenseDeal creates a test expense deal request.
func (b *TestDataBuilder) ExpenseDeal(amount int64, issueDate string) models.CreateDealRequest {
	if issueDate == "" {
		issueDate = time.Now().Format("2006-01-02")
	}

	return models.CreateDealRequest{
		CompanyID: b.companyID,
		IssueDate: issueDate,
		Type:      "expense",
		Details: []models.CreateDetailRequest{
			{
				AccountItemID: 801, // 仕入高
				TaxCode:       1,
				Amount:        amount,
			},
		},
	}
}

// MultiDetailDeal creates a deal with multiple details.
func (b *TestDataBuilder) MultiDetailDeal(amounts []int64, issueDate string) models.CreateDealRequest {
	if issueDate == "" {
		issueDate = time.Now().Format("2006-01-02")
	}

	details := make([]models.CreateDetailRequest, len(amounts))
	for i, amount := range amounts {
		details[i] = models.CreateDetailRequest{
			AccountItemID: int64(400 + i), // 400, 401, 402...
			TaxCode:       1,
			Amount:        amount,
		}
	}

	return models.CreateDealRequest{
		CompanyID: b.companyID,
		IssueDate: issueDate,
		Type:      "income",
		Details:   details,
	}
}

// SimpleJournal creates a simple balanced journal entry.
func (b *TestDataBuilder) SimpleJournal(amount int64, issueDate string) models.CreateJournalRequest {
	if issueDate == "" {
		issueDate = time.Now().Format("2006-01-02")
	}

	// Calculate amount without tax (10% tax)
	amountWithoutTax := amount * 10 / 11
	vat := amount - amountWithoutTax

	return models.CreateJournalRequest{
		CompanyID: b.companyID,
		IssueDate: issueDate,
		Details: []models.CreateJournalDetailRequest{
			{
				EntryType:     "debit",
				AccountItemID: 135, // 普通預金
				TaxCode:       0,
				Amount:        amount,
				Vat:           0,
			},
			{
				EntryType:     "credit",
				AccountItemID: 400, // 売上高
				TaxCode:       1,
				Amount:        amountWithoutTax,
				Vat:           vat,
			},
		},
	}
}

// ComplexJournal creates a complex journal entry with multiple lines.
func (b *TestDataBuilder) ComplexJournal(issueDate string) models.CreateJournalRequest {
	if issueDate == "" {
		issueDate = time.Now().Format("2006-01-02")
	}

	return models.CreateJournalRequest{
		CompanyID: b.companyID,
		IssueDate: issueDate,
		Details: []models.CreateJournalDetailRequest{
			{
				EntryType:     "debit",
				AccountItemID: 801, // 仕入高
				TaxCode:       1,
				Amount:        45454,
				Vat:           4546,
			},
			{
				EntryType:     "debit",
				AccountItemID: 138, // 支払手数料
				TaxCode:       0,
				Amount:        5000,
				Vat:           0,
			},
			{
				EntryType:     "credit",
				AccountItemID: 135, // 普通預金
				TaxCode:       0,
				Amount:        50000,
				Vat:           0,
			},
		},
	}
}

// DealWithRefNumber creates a deal with a reference number.
func (b *TestDataBuilder) DealWithRefNumber(amount int64, refNum string, issueDate string) models.CreateDealRequest {
	deal := b.IncomeDeal(amount, issueDate)
	deal.RefNumber = &refNum
	return deal
}

// DealWithPartner creates a deal with a partner.
func (b *TestDataBuilder) DealWithPartner(amount int64, partnerID int64, issueDate string) models.CreateDealRequest {
	deal := b.IncomeDeal(amount, issueDate)
	deal.PartnerID = &partnerID
	return deal
}

// GenerateRefNumber generates a reference number for testing.
func GenerateRefNumber(prefix string, seq int) string {
	return fmt.Sprintf("%s-%04d", prefix, seq)
}

// GenerateDateSequence generates a sequence of dates for testing.
func GenerateDateSequence(start time.Time, count int) []string {
	dates := make([]string, count)
	for i := 0; i < count; i++ {
		dates[i] = start.AddDate(0, 0, i).Format("2006-01-02")
	}
	return dates
}

package converter

import (
	"fmt"
	"math"
	"strings"

	"github.com/shunichi-ikebuchi/accounting-system/pkg/freee"
)

// BeancountTransaction represents a Beancount transaction.
type BeancountTransaction struct {
	Date      string
	Narration string
	Payee     string
	Tags      []string
	Postings  []BeancountPosting
}

// BeancountPosting represents a posting in a Beancount transaction.
type BeancountPosting struct {
	Account  string
	Amount   float64
	Currency string
	Comment  string
}

// Converter converts freee transactions to Beancount format.
type Converter struct {
	mapper   *Mapper
	currency string
}

// NewConverter creates a new Converter.
func NewConverter(mapper *Mapper, currency string) *Converter {
	if currency == "" {
		currency = "JPY"
	}
	return &Converter{
		mapper:   mapper,
		currency: currency,
	}
}

// ConvertDeal converts a Deal to Beancount transaction.
func (c *Converter) ConvertDeal(deal freee.Deal) BeancountTransaction {
	var postings []BeancountPosting

	// For income transactions, amounts should be negative (credit side)
	// For expense transactions, amounts should be positive (debit side)
	amountMultiplier := 1.0
	if deal.Type == "income" {
		amountMultiplier = -1.0
	}

	// Process each detail line in the deal
	for _, detail := range deal.Details {
		beancountAccount := c.mapper.GetBeancountAccount(detail.AccountItemName)

		if beancountAccount == "" {
			// Use unmapped account if no mapping found
			sanitized := sanitizeAccountName(detail.AccountItemName)
			beancountAccount = fmt.Sprintf("Expenses:Unmapped:%s", sanitized)
		}

		// Add posting for the main account (excluding VAT)
		postings = append(postings, BeancountPosting{
			Account:  beancountAccount,
			Amount:   float64(detail.Amount) * amountMultiplier,
			Currency: c.currency,
			Comment:  ptrToString(detail.Description),
		})

		// Add VAT posting if applicable
		if detail.Vat > 0 {
			taxAccount := c.mapper.GetTaxAccount("tax_10") // Default to 10% tax
			if taxAccount != nil {
				postings = append(postings, BeancountPosting{
					Account:  *taxAccount,
					Amount:   float64(detail.Vat) * amountMultiplier,
					Currency: c.currency,
					Comment:  "消費税",
				})
			}
		}
	}

	// Add payment postings
	if len(deal.Payments) > 0 {
		for _, payment := range deal.Payments {
			walletAccount := getWalletAccount(payment.FromWalletableType, payment.FromWalletableID)
			postings = append(postings, BeancountPosting{
				Account:  walletAccount,
				Amount:   -float64(payment.Amount), // Negative for outflow
				Currency: c.currency,
				Comment:  fmt.Sprintf("Payment from %s", payment.FromWalletableType),
			})
		}
	} else {
		// If no payment specified, add a balancing entry to a default account
		totalAmount := float64(deal.Amount)
		defaultAccount := "Assets:Current:Bank:Ordinary"

		// Opposite sign from the detail amounts to balance the transaction
		postings = append(postings, BeancountPosting{
			Account:  defaultAccount,
			Amount:   totalAmount * -amountMultiplier,
			Currency: c.currency,
		})
	}

	return BeancountTransaction{
		Date:      deal.IssueDate,
		Narration: buildDealNarration(deal),
		Payee:     ptrToString(deal.PartnerCode),
		Tags:      buildTags(deal.RefNumber),
		Postings:  postings,
	}
}

// ConvertJournal converts a Journal to Beancount transaction.
func (c *Converter) ConvertJournal(journal freee.Journal) BeancountTransaction {
	var postings []BeancountPosting

	for _, detail := range journal.Details {
		beancountAccount := c.mapper.GetBeancountAccount(detail.AccountItemName)

		if beancountAccount == "" {
			sanitized := sanitizeAccountName(detail.AccountItemName)
			beancountAccount = fmt.Sprintf("Expenses:Unmapped:%s", sanitized)
		}

		// Debit = positive, Credit = negative
		amount := float64(detail.Amount)
		if detail.EntryType == "credit" {
			amount = -amount
		}

		postings = append(postings, BeancountPosting{
			Account:  beancountAccount,
			Amount:   amount,
			Currency: c.currency,
			Comment:  ptrToString(detail.Description),
		})

		// Add VAT posting if applicable
		if detail.Vat > 0 {
			taxAccount := c.mapper.GetTaxAccount("tax_10")
			if taxAccount != nil {
				vatAmount := float64(detail.Vat)
				if detail.EntryType == "credit" {
					vatAmount = -vatAmount
				}
				postings = append(postings, BeancountPosting{
					Account:  *taxAccount,
					Amount:   vatAmount,
					Currency: c.currency,
					Comment:  "消費税",
				})
			}
		}
	}

	return BeancountTransaction{
		Date:      journal.IssueDate,
		Narration: buildJournalNarration(journal),
		Postings:  postings,
	}
}

// FormatTransaction formats a Beancount transaction as a string.
func (c *Converter) FormatTransaction(txn BeancountTransaction) string {
	var sb strings.Builder

	// Transaction header
	sb.WriteString(txn.Date)
	sb.WriteString(" *")
	if txn.Payee != "" {
		sb.WriteString(fmt.Sprintf(" \"%s\"", txn.Payee))
	}
	sb.WriteString(fmt.Sprintf(" \"%s\"", txn.Narration))
	if len(txn.Tags) > 0 {
		sb.WriteString(" #")
		sb.WriteString(strings.Join(txn.Tags, " #"))
	}
	sb.WriteString("\n")

	// Postings
	for _, posting := range txn.Postings {
		sb.WriteString("  ")
		sb.WriteString(posting.Account)

		// Right-align amount (typical Beancount style)
		spaces := int(math.Max(1, 60-float64(len(posting.Account))))
		sb.WriteString(strings.Repeat(" ", spaces))

		// Format amount
		sign := ""
		absAmount := posting.Amount
		if posting.Amount < 0 {
			sign = "-"
			absAmount = -posting.Amount
		}
		sb.WriteString(fmt.Sprintf("%s%.0f %s", sign, absAmount, posting.Currency))

		if posting.Comment != "" {
			sb.WriteString(fmt.Sprintf(" ; %s", posting.Comment))
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

// Helper functions

func sanitizeAccountName(name string) string {
	// Remove spaces for Beancount account names
	return strings.ReplaceAll(name, " ", "")
}

func ptrToString(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func buildTags(refNumber *string) []string {
	if refNumber == nil || *refNumber == "" {
		return nil
	}
	return []string{*refNumber}
}

func buildDealNarration(deal freee.Deal) string {
	if len(deal.Details) == 1 && deal.Details[0].Description != nil {
		return *deal.Details[0].Description
	}

	var accountNames []string
	for _, d := range deal.Details {
		accountNames = append(accountNames, d.AccountItemName)
	}

	typeLabel := "支出"
	if deal.Type == "income" {
		typeLabel = "収入"
	}

	return fmt.Sprintf("%s: %s", typeLabel, strings.Join(accountNames, ", "))
}

func buildJournalNarration(journal freee.Journal) string {
	var descriptions []string
	for _, d := range journal.Details {
		if d.Description != nil && *d.Description != "" {
			descriptions = append(descriptions, *d.Description)
		}
	}

	if len(descriptions) > 0 {
		return descriptions[0]
	}

	return "仕訳"
}

func getWalletAccount(walletType string, walletID int64) string {
	if walletType == "bank_account" {
		return "Assets:Current:Bank:Ordinary"
	} else if walletType == "credit_card" {
		return "Liabilities:Current:CreditCard"
	}
	return "Assets:Current:Bank:Ordinary"
}

// Package freee provides account mapping for freee API transactions.
package freee

import (
	"regexp"
	"strings"
)

// AccountMapping represents the mapping from a transaction to freee account items.
type AccountMapping struct {
	AccountItemID   int64  // 勘定科目ID
	AccountItemName string // 勘定科目名
	TaxCode         int    // 税区分コード
	TaxCodeName     string // 税区分名
}

// MappingRule defines a rule for mapping transactions to accounts.
type MappingRule struct {
	Pattern         *regexp.Regexp
	Keywords        []string // Case-insensitive keyword matching
	AccountItemID   int64
	AccountItemName string
	TaxCode         int
	TaxCodeName     string
}

// DefaultMappingRules provides the default account mapping rules.
// These are based on common expense categories for Japanese businesses.
var DefaultMappingRules = []MappingRule{
	// Books, subscriptions (新聞図書費)
	{
		Keywords:        []string{"amazon", "kindle", "book", "紀伊國屋", "丸善", "ジュンク堂"},
		AccountItemID:   502,
		AccountItemName: "新聞図書費",
		TaxCode:         136,
		TaxCodeName:     "課税仕入10%",
	},
	// Training and education (研修費)
	{
		Keywords:        []string{"udemy", "coursera", "skillshare", "セミナー", "研修"},
		AccountItemID:   503,
		AccountItemName: "研修費",
		TaxCode:         136,
		TaxCodeName:     "課税仕入10%",
	},
	// Software and subscriptions (支払手数料/ソフトウェア)
	{
		Keywords:        []string{"github", "notion", "slack", "figma", "adobe", "google", "microsoft", "apple.com/bill", "openai", "anthropic"},
		AccountItemID:   504,
		AccountItemName: "支払手数料",
		TaxCode:         136,
		TaxCodeName:     "課税仕入10%",
	},
	// Cloud services (通信費)
	{
		Keywords:        []string{"aws", "amazon web services", "gcp", "google cloud", "azure", "cloudflare", "vercel", "heroku", "netlify"},
		AccountItemID:   505,
		AccountItemName: "通信費",
		TaxCode:         136,
		TaxCodeName:     "課税仕入10%",
	},
	// Office supplies (消耗品費)
	{
		Keywords:        []string{"yodobashi", "ヨドバシ", "bic camera", "ビックカメラ", "loft", "東急ハンズ", "100均", "daiso"},
		AccountItemID:   506,
		AccountItemName: "消耗品費",
		TaxCode:         136,
		TaxCodeName:     "課税仕入10%",
	},
	// Travel expenses (旅費交通費)
	{
		Keywords:        []string{"suica", "pasmo", "jr ", "新幹線", "ana", "jal", "航空", "airlines", "expedia", "booking.com"},
		AccountItemID:   507,
		AccountItemName: "旅費交通費",
		TaxCode:         136,
		TaxCodeName:     "課税仕入10%",
	},
	// Entertainment (接待交際費)
	{
		Keywords:        []string{"飲食", "レストラン", "居酒屋", "カフェ", "starbucks", "スターバックス"},
		AccountItemID:   508,
		AccountItemName: "接待交際費",
		TaxCode:         136,
		TaxCodeName:     "課税仕入10%",
	},
	// General e-commerce (消耗品費 - fallback for general shopping)
	{
		Keywords:        []string{"rakuten", "楽天"},
		AccountItemID:   506,
		AccountItemName: "消耗品費",
		TaxCode:         136,
		TaxCodeName:     "課税仕入10%",
	},
}

// DefaultMapping is the fallback mapping when no rules match.
var DefaultMapping = AccountMapping{
	AccountItemID:   509,
	AccountItemName: "雑費",
	TaxCode:         136,
	TaxCodeName:     "課税仕入10%",
}

// AccountMapper provides account mapping functionality.
type AccountMapper struct {
	rules          []MappingRule
	defaultMapping AccountMapping
}

// NewAccountMapper creates a new AccountMapper with default rules.
func NewAccountMapper() *AccountMapper {
	return &AccountMapper{
		rules:          DefaultMappingRules,
		defaultMapping: DefaultMapping,
	}
}

// NewAccountMapperWithRules creates a new AccountMapper with custom rules.
func NewAccountMapperWithRules(rules []MappingRule, defaultMapping AccountMapping) *AccountMapper {
	return &AccountMapper{
		rules:          rules,
		defaultMapping: defaultMapping,
	}
}

// GetMapping returns the account mapping for a given transaction description.
func (m *AccountMapper) GetMapping(description string) AccountMapping {
	descLower := strings.ToLower(description)

	for _, rule := range m.rules {
		// Check pattern first if defined
		if rule.Pattern != nil && rule.Pattern.MatchString(description) {
			return AccountMapping{
				AccountItemID:   rule.AccountItemID,
				AccountItemName: rule.AccountItemName,
				TaxCode:         rule.TaxCode,
				TaxCodeName:     rule.TaxCodeName,
			}
		}

		// Check keywords
		for _, keyword := range rule.Keywords {
			if strings.Contains(descLower, strings.ToLower(keyword)) {
				return AccountMapping{
					AccountItemID:   rule.AccountItemID,
					AccountItemName: rule.AccountItemName,
					TaxCode:         rule.TaxCode,
					TaxCodeName:     rule.TaxCodeName,
				}
			}
		}
	}

	return m.defaultMapping
}

// GetMappingForTransaction returns the account mapping for a Transaction.
func (m *AccountMapper) GetMappingForTransaction(txn Transaction) AccountMapping {
	// Try vendor first if available
	if txn.Vendor != "" && txn.Vendor != "unknown" {
		mapping := m.GetMapping(txn.Vendor)
		if mapping.AccountItemID != m.defaultMapping.AccountItemID {
			return mapping
		}
	}
	// Fall back to description
	return m.GetMapping(txn.Description)
}

// GetMappingForWalletTransaction returns the account mapping for a WalletTransaction.
func (m *AccountMapper) GetMappingForWalletTransaction(txn WalletTransaction) AccountMapping {
	return m.GetMapping(txn.Description)
}

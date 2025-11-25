// Package converter provides conversion from freee data to Beancount format.
package converter

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// AccountMapping represents a mapping between freee and Beancount account names.
type AccountMapping struct {
	Freee     string `yaml:"freee"`
	Beancount string `yaml:"beancount"`
	Type      string `yaml:"type"`
}

// TaxCodeMapping represents a tax code mapping.
type TaxCodeMapping struct {
	Code             string  `yaml:"code"`
	Rate             float64 `yaml:"rate"`
	Description      string  `yaml:"description"`
	BeancountAccount *string `yaml:"beancount_account"`
}

// AccountMappingConfig represents the complete account mapping configuration.
type AccountMappingConfig struct {
	Assets struct {
		Current []AccountMapping `yaml:"current"`
		Fixed   []AccountMapping `yaml:"fixed"`
	} `yaml:"assets"`
	Liabilities struct {
		Current  []AccountMapping `yaml:"current"`
		Longterm []AccountMapping `yaml:"longterm"`
	} `yaml:"liabilities"`
	Equity   []AccountMapping   `yaml:"equity"`
	Income   []AccountMapping   `yaml:"income"`
	Expenses struct {
		COGS         []AccountMapping `yaml:"cogs"`
		SGA          []AccountMapping `yaml:"sga"`
		Nonoperating []AccountMapping `yaml:"nonoperating"`
	} `yaml:"expenses"`
	TaxCodes []TaxCodeMapping `yaml:"tax_codes"`
}

// Mapper maps freee account names to Beancount account names.
type Mapper struct {
	config           AccountMappingConfig
	freeeToBean      map[string]string
	taxCodeMap       map[string]TaxCodeMapping
}

// NewMapper creates a new Mapper from a YAML configuration file.
func NewMapper(configPath string) (*Mapper, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config AccountMappingConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	mapper := &Mapper{
		config:      config,
		freeeToBean: make(map[string]string),
		taxCodeMap:  make(map[string]TaxCodeMapping),
	}

	mapper.buildMappingMaps()

	return mapper, nil
}

// buildMappingMaps builds internal mapping maps from configuration.
func (m *Mapper) buildMappingMaps() {
	// Assets
	for _, mapping := range m.config.Assets.Current {
		m.freeeToBean[mapping.Freee] = mapping.Beancount
	}
	for _, mapping := range m.config.Assets.Fixed {
		m.freeeToBean[mapping.Freee] = mapping.Beancount
	}

	// Liabilities
	for _, mapping := range m.config.Liabilities.Current {
		m.freeeToBean[mapping.Freee] = mapping.Beancount
	}
	for _, mapping := range m.config.Liabilities.Longterm {
		m.freeeToBean[mapping.Freee] = mapping.Beancount
	}

	// Equity
	for _, mapping := range m.config.Equity {
		m.freeeToBean[mapping.Freee] = mapping.Beancount
	}

	// Income
	for _, mapping := range m.config.Income {
		m.freeeToBean[mapping.Freee] = mapping.Beancount
	}

	// Expenses
	for _, mapping := range m.config.Expenses.COGS {
		m.freeeToBean[mapping.Freee] = mapping.Beancount
	}
	for _, mapping := range m.config.Expenses.SGA {
		m.freeeToBean[mapping.Freee] = mapping.Beancount
	}
	for _, mapping := range m.config.Expenses.Nonoperating {
		m.freeeToBean[mapping.Freee] = mapping.Beancount
	}

	// Tax codes
	for _, taxCode := range m.config.TaxCodes {
		m.taxCodeMap[taxCode.Code] = taxCode
	}
}

// GetBeancountAccount returns the Beancount account name for a freee account name.
// Returns empty string if no mapping is found.
func (m *Mapper) GetBeancountAccount(freeeName string) string {
	return m.freeeToBean[freeeName]
}

// GetBeancountAccountWithFallback returns the Beancount account name with a fallback.
func (m *Mapper) GetBeancountAccountWithFallback(freeeName, fallback string) string {
	if account := m.freeeToBean[freeeName]; account != "" {
		return account
	}
	return fallback
}

// GetTaxCode returns tax code mapping information.
func (m *Mapper) GetTaxCode(taxCode string) *TaxCodeMapping {
	if mapping, ok := m.taxCodeMap[taxCode]; ok {
		return &mapping
	}
	return nil
}

// GetTaxRate returns the tax rate for a given tax code.
// Returns 0 if the tax code is not found.
func (m *Mapper) GetTaxRate(taxCode string) float64 {
	if mapping, ok := m.taxCodeMap[taxCode]; ok {
		return mapping.Rate
	}
	return 0
}

// GetTaxAccount returns the Beancount tax account for a given tax code.
// Returns nil if exempt or not applicable.
func (m *Mapper) GetTaxAccount(taxCode string) *string {
	if mapping, ok := m.taxCodeMap[taxCode]; ok {
		return mapping.BeancountAccount
	}
	return nil
}

// HasMapping checks if a mapping exists for a freee account.
func (m *Mapper) HasMapping(freeeName string) bool {
	_, ok := m.freeeToBean[freeeName]
	return ok
}

// GetAllMappings returns all mapped account names.
func (m *Mapper) GetAllMappings() map[string]string {
	result := make(map[string]string, len(m.freeeToBean))
	for k, v := range m.freeeToBean {
		result[k] = v
	}
	return result
}

// Package config provides configuration management for the accounting system.
// It loads configuration from environment variables and .env files.
package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config represents the application configuration.
type Config struct {
	Freee     FreeeConfig
	Beancount BeancountConfig
	Debug     bool
	NodeEnv   string
}

// FreeeConfig represents freee API configuration.
type FreeeConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	AccessToken  string
	CompanyID    int64
	APIURL       string
}

// BeancountConfig represents Beancount-related configuration.
type BeancountConfig struct {
	Root           string
	DBPath         string
	AttachmentsDir string
}

// Load loads configuration from environment variables.
// It automatically loads .env file from the current directory if available.
// You can optionally specify a custom .env file path.
func Load(envPath ...string) (*Config, error) {
	// Load .env file
	if len(envPath) > 0 && envPath[0] != "" {
		if err := godotenv.Load(envPath[0]); err != nil {
			return nil, fmt.Errorf("failed to load .env file: %w", err)
		}
	} else {
		// Try to load .env from current directory (ignore error if not found)
		_ = godotenv.Load()
	}

	// Parse CompanyID
	companyID, err := parseInt64Env("FREEE_COMPANY_ID", 0)
	if err != nil {
		return nil, fmt.Errorf("invalid FREEE_COMPANY_ID: %w", err)
	}

	config := &Config{
		Freee: FreeeConfig{
			ClientID:     os.Getenv("FREEE_CLIENT_ID"),
			ClientSecret: os.Getenv("FREEE_CLIENT_SECRET"),
			RedirectURI:  os.Getenv("FREEE_REDIRECT_URI"),
			AccessToken:  os.Getenv("FREEE_ACCESS_TOKEN"),
			CompanyID:    companyID,
			APIURL:       getEnvOrDefault("FREEE_API_URL", "http://localhost:8080"),
		},
		Beancount: BeancountConfig{
			Root:           getEnvOrDefault("BEANCOUNT_ROOT", "./beancount"),
			DBPath:         os.Getenv("BEANCOUNT_DB_PATH"),
			AttachmentsDir: os.Getenv("BEANCOUNT_ATTACHMENTS_DIR"),
		},
		Debug:   os.Getenv("DEBUG") == "true",
		NodeEnv: getEnvOrDefault("NODE_ENV", "development"),
	}

	return config, nil
}

// Validate validates the configuration.
// It checks if all required fields are set.
func (c *Config) Validate(required ...[]string) error {
	var missing []string

	for _, path := range required {
		if len(path) == 0 {
			continue
		}

		var value string
		switch path[0] {
		case "freee":
			if len(path) < 2 {
				continue
			}
			switch path[1] {
			case "clientId":
				value = c.Freee.ClientID
			case "clientSecret":
				value = c.Freee.ClientSecret
			case "redirectUri":
				value = c.Freee.RedirectURI
			case "accessToken":
				value = c.Freee.AccessToken
			case "companyId":
				if c.Freee.CompanyID == 0 {
					value = ""
				} else {
					value = "set"
				}
			case "apiUrl":
				value = c.Freee.APIURL
			}
		case "beancount":
			if len(path) < 2 {
				continue
			}
			switch path[1] {
			case "root":
				value = c.Beancount.Root
			case "dbPath":
				value = c.Beancount.DBPath
			case "attachmentsDir":
				value = c.Beancount.AttachmentsDir
			}
		}

		if value == "" {
			missing = append(missing, joinPath(path))
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required configuration: %v\nPlease check your .env file or environment variables", missing)
	}

	return nil
}

// getEnvOrDefault returns the value of the environment variable or a default value if not set.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// parseInt64Env parses an int64 from an environment variable.
// Returns defaultValue if the environment variable is not set.
func parseInt64Env(key string, defaultValue int64) (int64, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid integer value for %s: %s", key, value)
	}

	return parsed, nil
}

// joinPath joins a path slice into a dot-separated string.
func joinPath(path []string) string {
	result := ""
	for i, p := range path {
		if i > 0 {
			result += "."
		}
		result += p
	}
	return result
}

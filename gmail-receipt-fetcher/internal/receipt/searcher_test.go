package receipt

import "testing"

func TestExtractAmount(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{"yen symbol", "合計: ¥1,234", 1234},
		{"yen symbol fullwidth", "￥5,678", 5678},
		{"yen suffix", "合計 9,999円", 9999},
		{"total keyword", "total: 12345", 12345},
		{"amount keyword", "amount ¥100", 100},
		{"no comma", "¥1000", 1000},
		{"large amount", "合計金額: ¥1,234,567", 1234567},
		{"mixed text", "Your order total is ¥3,500 including tax", 3500},
		{"japanese total", "お支払い金額: 2,980円", 2980},
		{"subtotal", "小計: ¥500", 500},
		{"jpy format", "JPY 1500", 1500},
		{"jpy suffix", "1500 jpy", 1500},
		{"no amount", "Thank you for your order", 0},
		{"zero amount", "¥0", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractAmount(tt.text)
			if result != tt.expected {
				t.Errorf("extractAmount(%q) = %d, expected %d", tt.text, result, tt.expected)
			}
		})
	}
}

func TestExtractVendor(t *testing.T) {
	tests := []struct {
		name     string
		from     string
		expected string
	}{
		{"amazon lowercase", "noreply@amazon.co.jp", "amazon"},
		{"amazon with name", "Amazon.co.jp <noreply@amazon.co.jp>", "amazon"},
		{"rakuten", "楽天市場 <order@rakuten.co.jp>", "rakuten"},
		{"yodobashi", "ヨドバシ・ドット・コム <info@yodobashi.com>", "yodobashi"},
		{"apple", "Apple <no_reply@email.apple.com>", "apple"},
		{"google", "Google Play <googleplay-noreply@google.com>", "google"},
		{"unknown domain", "support@example.com", "example"},
		{"unknown with name", "Example Store <shop@store.example.com>", "store"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractVendor(tt.from)
			if result != tt.expected {
				t.Errorf("extractVendor(%q) = %q, expected %q", tt.from, result, tt.expected)
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"normal", "amazon", "amazon"},
		{"with slash", "test/file", "test_file"},
		{"with colon", "file:name", "file_name"},
		{"with quotes", `file"name`, "file_name"},
		{"with spaces trim", "  filename  ", "filename"},
		{"with dots trim", "..filename..", "filename"},
		{"long name", "this_is_a_very_long_filename_that_should_be_truncated_to_fifty_characters", "this_is_a_very_long_filename_that_should_be_trunca"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseEmailDate(t *testing.T) {
	tests := []struct {
		name      string
		dateStr   string
		expectErr bool
	}{
		{"RFC1123Z", "Mon, 02 Jan 2006 15:04:05 -0700", false},
		{"RFC1123", "Mon, 02 Jan 2006 15:04:05 MST", false},
		{"with timezone name", "Mon, 02 Jan 2006 15:04:05 -0700 (JST)", false},
		{"invalid", "not a date", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseEmailDate(tt.dateStr)
			if (err != nil) != tt.expectErr {
				t.Errorf("parseEmailDate(%q) error = %v, expectErr = %v", tt.dateStr, err, tt.expectErr)
			}
		})
	}
}

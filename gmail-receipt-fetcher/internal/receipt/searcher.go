// Package receipt provides receipt search and download functionality.
package receipt

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"google.golang.org/api/gmail/v1"
)

// Searcher searches Gmail for receipt emails and downloads attachments.
type Searcher struct {
	service    *gmail.Service
	userID     string
	outputDir  string
}

// Receipt represents a found receipt email.
type Receipt struct {
	MessageID   string
	Subject     string
	From        string
	To          string
	Date        time.Time
	Amount      int       // Amount in JPY
	Vendor      string
	Attachments []Attachment
	HTMLBody    string    // HTML body of the email (when no attachment)
}

// Attachment represents an email attachment.
type Attachment struct {
	Filename    string
	MimeType    string
	Size        int64
	AttachmentID string
}

// DownloadResult represents the result of a download operation.
type DownloadResult struct {
	Receipt     *Receipt
	FilePath    string
	Status      string // "downloaded", "skipped", "error"
	Error       string
}

// SearchParams holds parameters for receipt search.
type SearchParams struct {
	FromDate  time.Time
	ToDate    time.Time
	Vendors   []string // Vendor names to search for (e.g., "Amazon", "楽天")
	Label     string   // Gmail label to filter by (e.g., "領収書")
	MinAmount int
	MaxAmount int
}

// NewSearcher creates a new receipt searcher.
func NewSearcher(service *gmail.Service, userID, outputDir string) *Searcher {
	return &Searcher{
		service:   service,
		userID:    userID,
		outputDir: outputDir,
	}
}

// Search searches Gmail for receipt emails matching the parameters.
func (s *Searcher) Search(ctx context.Context, params SearchParams) ([]*Receipt, error) {
	// Build Gmail search query
	query := s.buildQuery(params)
	fmt.Printf("Gmail search query: %s\n", query)

	// Search messages
	var receipts []*Receipt
	pageToken := ""

	for {
		call := s.service.Users.Messages.List(s.userID).Q(query).MaxResults(100)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		resp, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("failed to list messages: %w", err)
		}

		for _, msg := range resp.Messages {
			receipt, err := s.parseMessage(ctx, msg.Id)
			if err != nil {
				fmt.Printf("Warning: failed to parse message %s: %v\n", msg.Id, err)
				continue
			}
			if receipt != nil {
				receipts = append(receipts, receipt)
			}
		}

		pageToken = resp.NextPageToken
		if pageToken == "" {
			break
		}
	}

	return receipts, nil
}

// buildQuery builds Gmail search query from parameters.
func (s *Searcher) buildQuery(params SearchParams) string {
	var parts []string

	// Label filter (if specified, use label instead of general keywords)
	if params.Label != "" {
		parts = append(parts, fmt.Sprintf("label:%s", params.Label))
	}

	// Date range
	if !params.FromDate.IsZero() {
		parts = append(parts, fmt.Sprintf("after:%s", params.FromDate.Format("2006/01/02")))
	}
	if !params.ToDate.IsZero() {
		parts = append(parts, fmt.Sprintf("before:%s", params.ToDate.Format("2006/01/02")))
	}

	// Vendor filter
	if len(params.Vendors) > 0 {
		vendorParts := make([]string, len(params.Vendors))
		for i, v := range params.Vendors {
			vendorParts[i] = fmt.Sprintf("from:%s", v)
		}
		parts = append(parts, fmt.Sprintf("{%s}", strings.Join(vendorParts, " ")))
	}

	// Common receipt keywords (only if no label filter)
	if params.Label == "" {
		parts = append(parts, "{subject:領収書 subject:receipt subject:ご注文 subject:order}")
		// Has attachment (only when not using label filter)
		parts = append(parts, "has:attachment")
	}

	return strings.Join(parts, " ")
}

// parseMessage parses a Gmail message into a Receipt.
func (s *Searcher) parseMessage(ctx context.Context, messageID string) (*Receipt, error) {
	msg, err := s.service.Users.Messages.Get(s.userID, messageID).Format("full").Do()
	if err != nil {
		return nil, err
	}

	receipt := &Receipt{
		MessageID: messageID,
	}

	// Parse headers
	for _, header := range msg.Payload.Headers {
		switch header.Name {
		case "Subject":
			receipt.Subject = header.Value
		case "From":
			receipt.From = header.Value
			receipt.Vendor = extractVendor(header.Value)
		case "To":
			receipt.To = header.Value
		case "Date":
			if t, err := parseEmailDate(header.Value); err == nil {
				receipt.Date = t
			}
		}
	}

	// Extract amount from subject first
	receipt.Amount = extractAmount(receipt.Subject)

	// If not found in subject, try extracting from body
	if receipt.Amount == 0 {
		body := extractBody(msg.Payload)
		receipt.Amount = extractAmount(body)
	}

	// Find PDF attachments
	s.findAttachments(msg.Payload, receipt)

	// If no PDF attachments, store HTML body for PDF conversion
	if len(receipt.Attachments) == 0 {
		receipt.HTMLBody = extractHTMLBody(msg.Payload)
		// Skip if neither attachment nor HTML body
		if receipt.HTMLBody == "" {
			return nil, nil
		}
	}

	return receipt, nil
}

// findAttachments recursively finds attachments in message parts.
func (s *Searcher) findAttachments(part *gmail.MessagePart, receipt *Receipt) {
	if part.Filename != "" && part.Body != nil && part.Body.AttachmentId != "" {
		// Check if it's a PDF
		if strings.HasSuffix(strings.ToLower(part.Filename), ".pdf") ||
			part.MimeType == "application/pdf" {
			receipt.Attachments = append(receipt.Attachments, Attachment{
				Filename:     part.Filename,
				MimeType:     part.MimeType,
				Size:         part.Body.Size,
				AttachmentID: part.Body.AttachmentId,
			})
		}
	}

	// Recurse into parts
	for _, p := range part.Parts {
		s.findAttachments(p, receipt)
	}
}

// Download downloads PDF attachments from a receipt or converts HTML body to PDF.
func (s *Searcher) Download(ctx context.Context, receipt *Receipt) (*DownloadResult, error) {
	result := &DownloadResult{
		Receipt: receipt,
	}

	// Ensure output directory exists
	if err := os.MkdirAll(s.outputDir, 0755); err != nil {
		result.Status = "error"
		result.Error = err.Error()
		return result, err
	}

	// Case 1: Has PDF attachment - download it
	if len(receipt.Attachments) > 0 {
		return s.downloadAttachment(ctx, receipt, result)
	}

	// Case 2: No attachment but has HTML body - convert to PDF
	if receipt.HTMLBody != "" {
		return s.convertHTMLToPDF(ctx, receipt, result)
	}

	result.Status = "skipped"
	result.Error = "no PDF attachments and no HTML body"
	return result, nil
}

// downloadAttachment downloads PDF attachment from the receipt.
func (s *Searcher) downloadAttachment(ctx context.Context, receipt *Receipt, result *DownloadResult) (*DownloadResult, error) {
	att := receipt.Attachments[0]

	// Generate filename
	filename := s.generateFilename(receipt, att)
	filePath := filepath.Join(s.outputDir, filename)

	// Check if already exists
	if _, err := os.Stat(filePath); err == nil {
		result.Status = "skipped"
		result.FilePath = filePath
		return result, nil
	}

	// Download attachment
	attData, err := s.service.Users.Messages.Attachments.Get(s.userID, receipt.MessageID, att.AttachmentID).Do()
	if err != nil {
		result.Status = "error"
		result.Error = err.Error()
		return result, err
	}

	// Decode base64
	data, err := base64.URLEncoding.DecodeString(attData.Data)
	if err != nil {
		result.Status = "error"
		result.Error = err.Error()
		return result, err
	}

	// Write file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		result.Status = "error"
		result.Error = err.Error()
		return result, err
	}

	result.Status = "downloaded"
	result.FilePath = filePath
	return result, nil
}

// convertHTMLToPDF converts email HTML body to PDF using chromedp.
func (s *Searcher) convertHTMLToPDF(ctx context.Context, receipt *Receipt, result *DownloadResult) (*DownloadResult, error) {
	// Generate filename for HTML-converted PDF
	date := receipt.Date.Format("2006-01-02")
	vendor := sanitizeFilename(receipt.Vendor)
	if vendor == "" {
		vendor = "unknown"
	}
	filename := fmt.Sprintf("%s_%s_receipt.pdf", date, vendor)
	filePath := filepath.Join(s.outputDir, filename)

	// Check if already exists
	if _, err := os.Stat(filePath); err == nil {
		result.Status = "skipped"
		result.FilePath = filePath
		return result, nil
	}

	// Prepare email info for PDF header
	emailInfo := &EmailInfo{
		Subject: receipt.Subject,
		From:    receipt.From,
		To:      receipt.To,
		Date:    receipt.Date,
	}

	// Convert HTML to PDF using chromedp
	pdfData, err := htmlToPDF(ctx, receipt.HTMLBody, emailInfo)
	if err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("PDF conversion failed: %v", err)
		return result, err
	}

	// Write PDF file
	if err := os.WriteFile(filePath, pdfData, 0644); err != nil {
		result.Status = "error"
		result.Error = err.Error()
		return result, err
	}

	result.Status = "downloaded"
	result.FilePath = filePath
	return result, nil
}

// generateFilename generates a filename for the receipt.
func (s *Searcher) generateFilename(receipt *Receipt, att Attachment) string {
	date := receipt.Date.Format("2006-01-02")
	vendor := sanitizeFilename(receipt.Vendor)
	if vendor == "" {
		vendor = "unknown"
	}

	// Use original filename or generate one
	ext := ".pdf"
	if strings.HasSuffix(strings.ToLower(att.Filename), ".pdf") {
		// Use original filename with date prefix
		return fmt.Sprintf("%s_%s_%s", date, vendor, att.Filename)
	}

	return fmt.Sprintf("%s_%s_receipt%s", date, vendor, ext)
}

// extractVendor extracts vendor name from email From header.
func extractVendor(from string) string {
	// Common patterns
	vendorPatterns := map[*regexp.Regexp]string{
		regexp.MustCompile(`(?i)amazon`):            "amazon",
		regexp.MustCompile(`(?i)rakuten|楽天`):         "rakuten",
		regexp.MustCompile(`(?i)yodobashi|ヨドバシ`):     "yodobashi",
		regexp.MustCompile(`(?i)apple`):             "apple",
		regexp.MustCompile(`(?i)google`):            "google",
		regexp.MustCompile(`(?i)nomad`):             "nomad",
		regexp.MustCompile(`(?i)teachable|コース`):     "teachable",
		regexp.MustCompile(`(?i)substack`):          "substack",
		regexp.MustCompile(`(?i)paypal`):            "paypal",
		regexp.MustCompile(`(?i)stripe`):            "stripe",
		regexp.MustCompile(`(?i)gumroad`):           "gumroad",
	}

	for pattern, name := range vendorPatterns {
		if pattern.MatchString(from) {
			return name
		}
	}

	// Try to extract display name first (e.g., "John Doe <john@example.com>")
	if idx := strings.Index(from, "<"); idx > 0 {
		displayName := strings.TrimSpace(from[:idx])
		displayName = strings.Trim(displayName, "\"'")
		if displayName != "" && len(displayName) <= 30 {
			return sanitizeFilename(displayName)
		}
	}

	// Extract email domain as fallback
	if idx := strings.Index(from, "@"); idx != -1 {
		end := strings.Index(from[idx:], ">")
		if end == -1 {
			end = len(from) - idx
		}
		domain := from[idx+1 : idx+end]
		parts := strings.Split(domain, ".")
		if len(parts) > 0 {
			return parts[0]
		}
	}

	return ""
}

// extractBody extracts text body from message parts recursively.
func extractBody(part *gmail.MessagePart) string {
	if part == nil {
		return ""
	}

	// Check if this part is text
	if strings.HasPrefix(part.MimeType, "text/") && part.Body != nil && part.Body.Data != "" {
		data, err := base64.URLEncoding.DecodeString(part.Body.Data)
		if err == nil {
			return string(data)
		}
	}

	// Recurse into parts
	var result strings.Builder
	for _, p := range part.Parts {
		if body := extractBody(p); body != "" {
			result.WriteString(body)
			result.WriteString("\n")
		}
	}
	return result.String()
}

// extractHTMLBody extracts HTML body from message parts, preferring HTML over plain text.
func extractHTMLBody(part *gmail.MessagePart) string {
	if part == nil {
		return ""
	}

	// First pass: look for HTML only (no fallback)
	if html := findHTMLPart(part); html != "" {
		return html
	}

	// Second pass: fallback to plain text
	return findPlainTextAsFallback(part)
}

// findHTMLPart recursively searches for text/html part only.
func findHTMLPart(part *gmail.MessagePart) string {
	if part == nil {
		return ""
	}

	// Check if this part is HTML
	if part.MimeType == "text/html" && part.Body != nil && part.Body.Data != "" {
		data, err := base64.URLEncoding.DecodeString(part.Body.Data)
		if err == nil {
			return string(data)
		}
	}

	// Recurse into parts
	for _, p := range part.Parts {
		if html := findHTMLPart(p); html != "" {
			return html
		}
	}

	return ""
}

// findPlainTextAsFallback recursively searches for text/plain part and wraps it in HTML.
func findPlainTextAsFallback(part *gmail.MessagePart) string {
	if part == nil {
		return ""
	}

	if part.MimeType == "text/plain" && part.Body != nil && part.Body.Data != "" {
		data, err := base64.URLEncoding.DecodeString(part.Body.Data)
		if err == nil {
			return "<html><body><pre>" + string(data) + "</pre></body></html>"
		}
	}

	// Recurse into parts
	for _, p := range part.Parts {
		if text := findPlainTextAsFallback(p); text != "" {
			return text
		}
	}

	return ""
}

// extractAmount extracts amount in JPY from text.
func extractAmount(text string) int {
	// Remove HTML tags
	htmlTagPattern := regexp.MustCompile(`<[^>]*>`)
	text = htmlTagPattern.ReplaceAllString(text, " ")

	// Normalize whitespace
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")

	// Normalize text for better matching
	text = strings.ToLower(text)

	// Patterns for Japanese yen amounts (ordered by specificity)
	patterns := []*regexp.Regexp{
		// Total/subtotal patterns
		regexp.MustCompile(`(?:合計金額|ご請求額|お支払い金額|小計|subtotal|total amount|grand total)[^\d]*[¥￥]?\s*([\d,]+)`),
		// "合計 (JPY)" pattern (e.g., Teachable receipts)
		regexp.MustCompile(`合計\s*\(jpy\)[^\d]*[¥￥]?\s*([\d,]+)`),
		regexp.MustCompile(`(?:合計|total|amount)[^\d]*[¥￥]?\s*([\d,]+)`),
		// JPY patterns
		regexp.MustCompile(`jpy\s*([\d,]+)`),
		regexp.MustCompile(`([\d,]+)\s*jpy`),
	}

	for _, pattern := range patterns {
		if matches := pattern.FindStringSubmatch(text); len(matches) > 1 {
			amountStr := strings.ReplaceAll(matches[1], ",", "")
			if amount, err := strconv.Atoi(amountStr); err == nil && amount > 0 {
				return amount
			}
		}
	}

	// Fallback: find the largest yen amount in the text
	fallbackPatterns := []*regexp.Regexp{
		regexp.MustCompile(`[¥￥]\s*([\d,]+)`),
		regexp.MustCompile(`([\d,]+)\s*円`),
	}

	var maxAmount int
	for _, pattern := range fallbackPatterns {
		matches := pattern.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) > 1 {
				amountStr := strings.ReplaceAll(match[1], ",", "")
				if amount, err := strconv.Atoi(amountStr); err == nil && amount > maxAmount {
					maxAmount = amount
				}
			}
		}
	}
	return maxAmount
}

// parseEmailDate parses email Date header.
func parseEmailDate(dateStr string) (time.Time, error) {
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"2 Jan 2006 15:04:05 -0700",
		"Mon, 02 Jan 2006 15:04:05 -0700 (MST)",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

// sanitizeFilename removes invalid characters from filename.
func sanitizeFilename(name string) string {
	// Remove or replace invalid characters
	invalid := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	name = invalid.ReplaceAllString(name, "_")

	// Trim spaces and dots
	name = strings.Trim(name, " .")

	// Limit length
	if len(name) > 50 {
		name = name[:50]
	}

	return name
}

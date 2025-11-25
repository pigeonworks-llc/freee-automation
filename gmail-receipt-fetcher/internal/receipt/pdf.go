package receipt

import (
	"context"
	"encoding/base64"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// EmailInfo contains email metadata for PDF generation.
type EmailInfo struct {
	Subject string
	From    string
	To      string
	Date    time.Time
}

// htmlToPDF converts HTML content to PDF using headless Chrome (chromedp).
func htmlToPDF(ctx context.Context, body string, info *EmailInfo) ([]byte, error) {
	// Inline external images as base64
	body = inlineImages(body)

	// Wrap HTML body with email header
	fullHTML := wrapWithEmailHeader(body, info)

	// Create a new chromedp context
	allocCtx, cancel := chromedp.NewExecAllocator(ctx,
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", true),
			chromedp.Flag("disable-gpu", true),
			chromedp.Flag("no-sandbox", true),
		)...,
	)
	defer cancel()

	chromedpCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Set timeout
	chromedpCtx, cancel = context.WithTimeout(chromedpCtx, 30*time.Second)
	defer cancel()

	var pdfData []byte

	// Navigate to data URL with the HTML content and print to PDF
	err := chromedp.Run(chromedpCtx,
		chromedp.Navigate("about:blank"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			frameTree, err := page.GetFrameTree().Do(ctx)
			if err != nil {
				return err
			}
			return page.SetDocumentContent(frameTree.Frame.ID, fullHTML).Do(ctx)
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			pdfData, _, err = page.PrintToPDF().
				WithPrintBackground(true).
				WithPreferCSSPageSize(true).
				Do(ctx)
			return err
		}),
	)
	if err != nil {
		return nil, err
	}

	return pdfData, nil
}

// wrapWithEmailHeader injects email header into HTML body.
func wrapWithEmailHeader(body string, info *EmailInfo) string {
	if info == nil {
		return body
	}

	// Header block to inject
	headerBlock := fmt.Sprintf(`<div style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,'Helvetica Neue',Arial,sans-serif;padding:20px 24px;margin-bottom:20px;border:1px solid #e0e0e0;border-radius:8px;background-color:#fafafa;">
  <div style="font-size:18px;font-weight:500;color:#202124;margin-bottom:12px;">%s</div>
  <div style="font-size:13px;color:#5f6368;">
    <div style="margin-bottom:6px;">
      <span style="font-weight:500;color:#3c4043;">From:</span>
      <span>%s</span>
    </div>
    <div style="margin-bottom:6px;">
      <span style="font-weight:500;color:#3c4043;">To:</span>
      <span>%s</span>
    </div>
    <div>
      <span style="font-weight:500;color:#3c4043;">Date:</span>
      <span>%s</span>
    </div>
  </div>
</div>`,
		html.EscapeString(info.Subject),
		html.EscapeString(info.From),
		html.EscapeString(info.To),
		info.Date.Format("2006-01-02 15:04"),
	)

	// Try to inject after <body> tag
	bodyPattern := regexp.MustCompile(`(?i)(<body[^>]*>)`)
	if bodyPattern.MatchString(body) {
		return bodyPattern.ReplaceAllString(body, "${1}"+headerBlock)
	}

	// Fallback: prepend header block
	return headerBlock + body
}

// formatAmountWithCommas formats an integer with comma separators.
func formatAmountWithCommas(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

// inlineImages downloads external images and embeds them as base64 data URIs.
func inlineImages(htmlContent string) string {
	// Match img tags with src attribute
	imgPattern := regexp.MustCompile(`<img\s+([^>]*?)src=["']([^"']+)["']([^>]*)>`)

	return imgPattern.ReplaceAllStringFunc(htmlContent, func(match string) string {
		submatches := imgPattern.FindStringSubmatch(match)
		if len(submatches) < 4 {
			return match
		}

		beforeSrc := submatches[1]
		srcURL := submatches[2]
		afterSrc := submatches[3]

		// Skip already inlined images
		if strings.HasPrefix(srcURL, "data:") {
			return match
		}

		// Skip non-HTTP URLs
		if !strings.HasPrefix(srcURL, "http://") && !strings.HasPrefix(srcURL, "https://") {
			return match
		}

		// Download and encode image
		dataURI, err := downloadAndEncodeImage(srcURL)
		if err != nil {
			// Keep original URL if download fails
			return match
		}

		return fmt.Sprintf(`<img %ssrc="%s"%s>`, beforeSrc, dataURI, afterSrc)
	})
}

// downloadAndEncodeImage downloads an image and returns it as a base64 data URI.
func downloadAndEncodeImage(url string) (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download image: %s", resp.Status)
	}

	// Read image data
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Detect content type
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}

	// Clean up content type (remove charset etc)
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = strings.TrimSpace(contentType[:idx])
	}

	// Encode as base64 data URI
	encoded := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", contentType, encoded), nil
}

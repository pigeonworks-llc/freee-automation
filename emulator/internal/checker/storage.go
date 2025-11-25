// storage.go provides token storage implementations (Local file & Cloud Storage)
package checker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/oauth2"
)

// TokenStorage defines the interface for token persistence
type TokenStorage interface {
	Load(ctx context.Context) (*oauth2.Token, error)
	Save(ctx context.Context, token *oauth2.Token) error
}

// NewTokenStorage creates appropriate storage based on path
func NewTokenStorage(path string) TokenStorage {
	// Cloud Storage path: gs://bucket-name/path/to/token.json
	if strings.HasPrefix(path, "gs://") {
		return &GCSTokenStorage{path: path}
	}
	// Local file path
	return &FileTokenStorage{path: path}
}

// FileTokenStorage implements local file storage
type FileTokenStorage struct {
	path string
}

func (f *FileTokenStorage) Load(ctx context.Context) (*oauth2.Token, error) {
	data, err := os.ReadFile(f.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read token file: %w", err)
	}

	var saved SavedToken
	if err := json.Unmarshal(data, &saved); err != nil {
		return nil, fmt.Errorf("failed to parse token file: %w", err)
	}

	return &oauth2.Token{
		AccessToken:  saved.AccessToken,
		RefreshToken: saved.RefreshToken,
		TokenType:    saved.TokenType,
		Expiry:       saved.Expiry,
	}, nil
}

func (f *FileTokenStorage) Save(ctx context.Context, token *oauth2.Token) error {
	saved := SavedToken{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		Expiry:       token.Expiry,
	}

	data, err := json.MarshalIndent(saved, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	dir := filepath.Dir(f.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(f.path, data, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	return nil
}

// GCSTokenStorage implements Cloud Storage backend
// Note: Requires cloud.google.com/go/storage package
type GCSTokenStorage struct {
	path string // format: gs://bucket-name/path/to/token.json
}

func (g *GCSTokenStorage) Load(ctx context.Context) (*oauth2.Token, error) {
	// Parse GCS path
	bucket, object, err := parseGCSPath(g.path)
	if err != nil {
		return nil, err
	}

	// Note: この実装はプレースホルダーです
	// 実際の実装では cloud.google.com/go/storage を使用してください
	//
	// import "cloud.google.com/go/storage"
	//
	// client, err := storage.NewClient(ctx)
	// if err != nil {
	//     return nil, fmt.Errorf("failed to create GCS client: %w", err)
	// }
	// defer client.Close()
	//
	// reader, err := client.Bucket(bucket).Object(object).NewReader(ctx)
	// if err != nil {
	//     if err == storage.ErrObjectNotExist {
	//         return nil, nil
	//     }
	//     return nil, fmt.Errorf("failed to read from GCS: %w", err)
	// }
	// defer reader.Close()
	//
	// data, err := io.ReadAll(reader)
	// if err != nil {
	//     return nil, fmt.Errorf("failed to read data: %w", err)
	// }

	_ = bucket
	_ = object

	return nil, fmt.Errorf("GCS storage not yet implemented - add cloud.google.com/go/storage dependency")
}

func (g *GCSTokenStorage) Save(ctx context.Context, token *oauth2.Token) error {
	bucket, object, err := parseGCSPath(g.path)
	if err != nil {
		return err
	}

	saved := SavedToken{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		Expiry:       token.Expiry,
	}

	data, err := json.MarshalIndent(saved, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	// Note: この実装はプレースホルダーです
	// 実際の実装では cloud.google.com/go/storage を使用してください
	//
	// client, err := storage.NewClient(ctx)
	// if err != nil {
	//     return fmt.Errorf("failed to create GCS client: %w", err)
	// }
	// defer client.Close()
	//
	// writer := client.Bucket(bucket).Object(object).NewWriter(ctx)
	// if _, err := writer.Write(data); err != nil {
	//     return fmt.Errorf("failed to write to GCS: %w", err)
	// }
	// if err := writer.Close(); err != nil {
	//     return fmt.Errorf("failed to close GCS writer: %w", err)
	// }

	_ = bucket
	_ = object
	_ = data

	return fmt.Errorf("GCS storage not yet implemented - add cloud.google.com/go/storage dependency")
}

// parseGCSPath parses gs://bucket/path into bucket and object
func parseGCSPath(gcsPath string) (bucket, object string, err error) {
	if !strings.HasPrefix(gcsPath, "gs://") {
		return "", "", fmt.Errorf("invalid GCS path: %s", gcsPath)
	}

	path := strings.TrimPrefix(gcsPath, "gs://")
	parts := strings.SplitN(path, "/", 2)

	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid GCS path format: %s", gcsPath)
	}

	return parts[0], parts[1], nil
}

// Utility to suppress unused import warnings
var _ io.Reader

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

func main() {
	baseURL := os.Getenv("FREEE_API_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	// Get token
	tokenResp, err := http.Post(
		baseURL+"/oauth/token",
		"application/x-www-form-urlencoded",
		bytes.NewBufferString("grant_type=client_credentials"),
	)
	if err != nil {
		fmt.Println("Token Error:", err)
		return
	}
	defer tokenResp.Body.Close()

	var tokenData struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenData); err != nil {
		fmt.Println("Token Decode Error:", err)
		return
	}

	token := tokenData.AccessToken
	fmt.Printf("Token: %s\n\n", token)

	// Create test transactions
	testTxns := []map[string]interface{}{
		{
			"company_id":      1,
			"date":            "2025-10-01",
			"amount":          5000,
			"description":     "テスト取引1（未仕訳）",
			"walletable_type": "bank_account",
			"walletable_id":   1,
			"status":          "unbooked",
		},
		{
			"company_id":      1,
			"date":            "2025-10-02",
			"amount":          10000,
			"description":     "テスト取引2（未仕訳）",
			"walletable_type": "bank_account",
			"walletable_id":   1,
			"status":          "unbooked",
		},
		{
			"company_id":      1,
			"date":            "2025-10-03",
			"amount":          3000,
			"description":     "テスト取引3（仕訳済み）",
			"walletable_type": "bank_account",
			"walletable_id":   1,
			"status":          "settled",
		},
	}

	for i, txn := range testTxns {
		jsonData, _ := json.Marshal(txn)
		req, err := http.NewRequest("POST", baseURL+"/api/1/wallet_txns", bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Printf("Request %d Error: %v\n", i+1, err)
			continue
		}

		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Printf("API %d Error: %v\n", i+1, err)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		fmt.Printf("Transaction %d: Status=%d, Response=%s\n", i+1, resp.StatusCode, string(body))
	}

	fmt.Println("\nテストデータの作成が完了しました！")
}

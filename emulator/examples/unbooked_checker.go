package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type WalletTxn struct {
	Status string `json:"status"` // "unbooked" for 未仕分け
}

type Response struct {
	WalletTxns []WalletTxn `json:"wallet_txns"`
}

func main() {
	// エミュレータのURL（環境変数で切り替え可能）
	baseURL := os.Getenv("FREEE_API_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080" // デフォルトはエミュレータ
	}

	// アクセストークン取得
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
	fmt.Printf("Access Token: %s\n\n", token)

	// Company ID
	companyID := os.Getenv("FREEE_COMPANY_ID")
	if companyID == "" {
		companyID = "1" // デフォルト
	}

	// Wallet Transactions取得
	url := fmt.Sprintf("%s/api/1/wallet_txns?company_id=%s", baseURL, companyID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Request Error:", err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("API Error:", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Read Error:", err)
		return
	}

	fmt.Printf("Response Status: %d\n", resp.StatusCode)
	fmt.Printf("Response Body: %s\n\n", string(body))

	var data Response
	if err := json.Unmarshal(body, &data); err != nil {
		fmt.Println("JSON Decode Error:", err)
		return
	}

	// 未仕訳をカウント
	unclassifiedCount := 0
	for _, txn := range data.WalletTxns {
		if txn.Status == "unbooked" {
			unclassifiedCount++
		}
	}

	fmt.Printf("未仕分け明細: %d件\n", unclassifiedCount)

	if unclassifiedCount > 0 {
		// Google Chat通知（オプション）
		webhookURL := os.Getenv("GOOGLE_CHAT_WEBHOOK")
		if webhookURL != "" {
			payload := map[string]string{"text": fmt.Sprintf("未仕分け明細: %d件", unclassifiedCount)}
			jsonPayload, _ := json.Marshal(payload)
			notifyResp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonPayload))
			if err != nil {
				fmt.Println("Notification Error:", err)
			} else {
				defer notifyResp.Body.Close()
				fmt.Println("通知を送信しました")
			}
		}
	}
}

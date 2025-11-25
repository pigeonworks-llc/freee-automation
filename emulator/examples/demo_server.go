// demo_server はPlaywrightデモ用の簡易サーバーです
package main

import (
	"fmt"
	"log"
	"net/http"
)

const loginPageHTML = `<!DOCTYPE html>
<html lang="ja">
<head>
    <meta charset="UTF-8">
    <title>freee ログイン (Demo)</title>
    <style>
        body { font-family: sans-serif; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); display: flex; justify-content: center; align-items: center; min-height: 100vh; margin: 0; }
        .container { background: white; border-radius: 10px; padding: 40px; width: 400px; box-shadow: 0 10px 40px rgba(0,0,0,0.2); }
        .logo { text-align: center; font-size: 32px; font-weight: bold; color: #667eea; margin-bottom: 30px; }
        input { width: 100%; padding: 12px; margin: 10px 0; border: 1px solid #ddd; border-radius: 5px; box-sizing: border-box; }
        button { width: 100%; padding: 12px; background: #667eea; color: white; border: none; border-radius: 5px; font-size: 16px; font-weight: bold; cursor: pointer; margin-top: 10px; }
        button:hover { background: #5568d3; }
        .badge { background: #fbbf24; color: #78350f; padding: 5px 10px; border-radius: 5px; text-align: center; margin-bottom: 20px; font-size: 12px; font-weight: bold; }
    </style>
</head>
<body>
    <div class="container">
        <div class="badge">🧪 DEMO MODE</div>
        <div class="logo">freee</div>
        <form method="POST" action="/login">
            <input type="email" name="email" placeholder="メールアドレス" required autofocus>
            <input type="password" name="password" placeholder="パスワード" required>
            <button type="submit">ログイン</button>
        </form>
    </div>
</body>
</html>`

const tfaPageHTML = `<!DOCTYPE html>
<html lang="ja">
<head>
    <meta charset="UTF-8">
    <title>2要素認証 (Demo)</title>
    <style>
        body { font-family: sans-serif; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); display: flex; justify-content: center; align-items: center; min-height: 100vh; margin: 0; }
        .container { background: white; border-radius: 10px; padding: 40px; width: 400px; box-shadow: 0 10px 40px rgba(0,0,0,0.2); }
        .logo { text-align: center; font-size: 32px; font-weight: bold; color: #667eea; margin-bottom: 30px; }
        input { width: 100%; padding: 12px; margin: 10px 0; border: 1px solid #ddd; border-radius: 5px; box-sizing: border-box; }
        button { width: 100%; padding: 12px; background: #667eea; color: white; border: none; border-radius: 5px; font-size: 16px; font-weight: bold; cursor: pointer; margin-top: 10px; }
        button:hover { background: #5568d3; }
        .badge { background: #fbbf24; color: #78350f; padding: 5px 10px; border-radius: 5px; text-align: center; margin-bottom: 20px; font-size: 12px; font-weight: bold; }
    </style>
</head>
<body>
    <div class="container">
        <div class="badge">🧪 DEMO MODE - STEP 2/3</div>
        <div class="logo">freee</div>
        <form method="POST" action="/2fa">
            <input type="text" name="otp" placeholder="認証コード (6桁)" maxlength="6" required autofocus>
            <button type="submit">認証</button>
        </form>
    </div>
</body>
</html>`

const authPageHTML = `<!DOCTYPE html>
<html lang="ja">
<head>
    <meta charset="UTF-8">
    <title>アプリ認証 (Demo)</title>
    <style>
        body { font-family: sans-serif; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); display: flex; justify-content: center; align-items: center; min-height: 100vh; margin: 0; }
        .container { background: white; border-radius: 10px; padding: 40px; width: 400px; box-shadow: 0 10px 40px rgba(0,0,0,0.2); }
        .logo { text-align: center; font-size: 32px; font-weight: bold; color: #667eea; margin-bottom: 30px; }
        button { width: 100%; padding: 12px; border: none; border-radius: 5px; font-size: 16px; font-weight: bold; cursor: pointer; margin-top: 10px; }
        .btn-auth { background: #667eea; color: white; }
        .btn-auth:hover { background: #5568d3; }
        .badge { background: #fbbf24; color: #78350f; padding: 5px 10px; border-radius: 5px; text-align: center; margin-bottom: 20px; font-size: 12px; font-weight: bold; }
        .app-info { background: #f3f4f6; padding: 20px; border-radius: 5px; margin-bottom: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="badge">🧪 DEMO MODE - STEP 3/3</div>
        <div class="logo">freee</div>
        <div class="app-info">
            <div style="font-size: 18px; font-weight: bold; margin-bottom: 10px;">Unbooked Checker</div>
            <div style="font-size: 14px; color: #666;">
                <p>このアプリは以下の権限を要求しています:</p>
                <ul><li>取引データの読み取り</li><li>明細データの読み取り</li></ul>
            </div>
        </div>
        <form method="POST" action="/authorize">
            <button type="submit" class="btn-auth">許可する</button>
        </form>
    </div>
</body>
</html>`

const codePageHTML = `<!DOCTYPE html>
<html lang="ja">
<head>
    <meta charset="UTF-8">
    <title>認証完了 (Demo)</title>
    <style>
        body { font-family: sans-serif; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); display: flex; justify-content: center; align-items: center; min-height: 100vh; margin: 0; }
        .container { background: white; border-radius: 10px; padding: 40px; width: 400px; box-shadow: 0 10px 40px rgba(0,0,0,0.2); text-align: center; }
        .logo { font-size: 32px; font-weight: bold; color: #667eea; margin-bottom: 30px; }
        .code { background: #f3f4f6; padding: 20px; border-radius: 5px; font-size: 24px; font-weight: bold; font-family: monospace; color: #667eea; margin: 20px 0; user-select: all; }
        .badge { background: #fbbf24; color: #78350f; padding: 5px 10px; border-radius: 5px; margin-bottom: 20px; font-size: 12px; font-weight: bold; }
    </style>
</head>
<body>
    <div class="container">
        <div class="badge">🧪 DEMO MODE</div>
        <div class="logo">freee</div>
        <div style="font-size: 64px; margin-bottom: 20px;">✅</div>
        <div style="font-size: 18px; margin-bottom: 20px;">認証が完了しました</div>
        <div style="font-size: 14px; color: #666;">以下の認証コードをアプリケーションに入力してください</div>
        <div class="code" id="auth-code">ABC123DEF456GHI7</div>
    </div>
</body>
</html>`

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(loginPageHTML))
	})

	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(tfaPageHTML))
		} else {
			http.Redirect(w, r, "/", http.StatusSeeOther)
		}
	})

	http.HandleFunc("/2fa", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(authPageHTML))
		} else {
			http.Redirect(w, r, "/", http.StatusSeeOther)
		}
	})

	http.HandleFunc("/authorize", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(codePageHTML))
		} else {
			http.Redirect(w, r, "/", http.StatusSeeOther)
		}
	})

	fmt.Println("Demo server starting on http://localhost:9090")
	fmt.Println("Test credentials: test@example.com / password")
	fmt.Println("2FA code: 123456")
	log.Fatal(http.ListenAndServe(":9090", nil))
}

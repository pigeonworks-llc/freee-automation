package oauth

import (
	"fmt"
	"net/http"
)

const loginPageHTML = `
<!DOCTYPE html>
<html lang="ja">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>freee ログイン (Emulator)</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            display: flex;
            justify-content: center;
            align-items: center;
            min-height: 100vh;
            margin: 0;
        }
        .login-container {
            background: white;
            border-radius: 10px;
            box-shadow: 0 10px 40px rgba(0,0,0,0.2);
            padding: 40px;
            width: 100%;
            max-width: 400px;
        }
        .logo {
            text-align: center;
            margin-bottom: 30px;
            font-size: 32px;
            font-weight: bold;
            color: #667eea;
        }
        .form-group {
            margin-bottom: 20px;
        }
        label {
            display: block;
            margin-bottom: 5px;
            color: #333;
            font-weight: 500;
        }
        input[type="email"],
        input[type="password"],
        input[type="text"] {
            width: 100%;
            padding: 12px;
            border: 1px solid #ddd;
            border-radius: 5px;
            font-size: 14px;
            box-sizing: border-box;
        }
        input:focus {
            outline: none;
            border-color: #667eea;
        }
        button[type="submit"] {
            width: 100%;
            padding: 12px;
            background: #667eea;
            color: white;
            border: none;
            border-radius: 5px;
            font-size: 16px;
            font-weight: bold;
            cursor: pointer;
            transition: background 0.3s;
        }
        button[type="submit"]:hover {
            background: #5568d3;
        }
        .step-indicator {
            text-align: center;
            margin-bottom: 20px;
            color: #666;
            font-size: 14px;
        }
        .emulator-badge {
            background: #fbbf24;
            color: #78350f;
            padding: 5px 10px;
            border-radius: 5px;
            text-align: center;
            margin-bottom: 20px;
            font-size: 12px;
            font-weight: bold;
        }
    </style>
</head>
<body>
    <div class="login-container">
        <div class="emulator-badge">🧪 EMULATOR MODE</div>
        <div class="logo">freee</div>
        <div class="step-indicator">STEP %STEP% / 3</div>
        <form method="POST" action="%ACTION%">
            %CONTENT%
            <button type="submit">%BUTTON_TEXT%</button>
        </form>
    </div>
</body>
</html>
`

const authorizePageHTML = `
<!DOCTYPE html>
<html lang="ja">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>アプリ認証 (Emulator)</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            display: flex;
            justify-content: center;
            align-items: center;
            min-height: 100vh;
            margin: 0;
        }
        .auth-container {
            background: white;
            border-radius: 10px;
            box-shadow: 0 10px 40px rgba(0,0,0,0.2);
            padding: 40px;
            width: 100%;
            max-width: 400px;
        }
        .logo {
            text-align: center;
            margin-bottom: 30px;
            font-size: 32px;
            font-weight: bold;
            color: #667eea;
        }
        .app-info {
            background: #f3f4f6;
            padding: 20px;
            border-radius: 5px;
            margin-bottom: 20px;
        }
        .app-name {
            font-size: 18px;
            font-weight: bold;
            margin-bottom: 10px;
        }
        .permissions {
            font-size: 14px;
            color: #666;
        }
        .permissions li {
            margin: 5px 0;
        }
        button {
            width: 100%;
            padding: 12px;
            border: none;
            border-radius: 5px;
            font-size: 16px;
            font-weight: bold;
            cursor: pointer;
            transition: background 0.3s;
            margin-bottom: 10px;
        }
        .btn-authorize {
            background: #667eea;
            color: white;
        }
        .btn-authorize:hover {
            background: #5568d3;
        }
        .btn-cancel {
            background: #e5e7eb;
            color: #374151;
        }
        .btn-cancel:hover {
            background: #d1d5db;
        }
        .emulator-badge {
            background: #fbbf24;
            color: #78350f;
            padding: 5px 10px;
            border-radius: 5px;
            text-align: center;
            margin-bottom: 20px;
            font-size: 12px;
            font-weight: bold;
        }
    </style>
</head>
<body>
    <div class="auth-container">
        <div class="emulator-badge">🧪 EMULATOR MODE</div>
        <div class="logo">freee</div>
        <div class="app-info">
            <div class="app-name">Unbooked Checker</div>
            <div class="permissions">
                <p>このアプリは以下の権限を要求しています：</p>
                <ul>
                    <li>取引データの読み取り</li>
                    <li>明細データの読み取り</li>
                </ul>
            </div>
        </div>
        <form method="POST" action="/oauth/authorize/confirm">
            <input type="hidden" name="session_id" value="%SESSION_ID%">
            <button type="submit" class="btn-authorize">許可する</button>
        </form>
        <button class="btn-cancel" onclick="window.close()">キャンセル</button>
    </div>
</body>
</html>
`

const codePageHTML = `
<!DOCTYPE html>
<html lang="ja">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>認証コード (Emulator)</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            display: flex;
            justify-content: center;
            align-items: center;
            min-height: 100vh;
            margin: 0;
        }
        .code-container {
            background: white;
            border-radius: 10px;
            box-shadow: 0 10px 40px rgba(0,0,0,0.2);
            padding: 40px;
            width: 100%;
            max-width: 400px;
            text-align: center;
        }
        .logo {
            margin-bottom: 30px;
            font-size: 32px;
            font-weight: bold;
            color: #667eea;
        }
        .success-icon {
            font-size: 64px;
            margin-bottom: 20px;
        }
        .message {
            font-size: 18px;
            margin-bottom: 20px;
            color: #333;
        }
        .code {
            background: #f3f4f6;
            padding: 20px;
            border-radius: 5px;
            font-size: 24px;
            font-weight: bold;
            font-family: monospace;
            color: #667eea;
            margin: 20px 0;
            user-select: all;
        }
        .hint {
            font-size: 14px;
            color: #666;
        }
        .emulator-badge {
            background: #fbbf24;
            color: #78350f;
            padding: 5px 10px;
            border-radius: 5px;
            margin-bottom: 20px;
            font-size: 12px;
            font-weight: bold;
        }
    </style>
</head>
<body>
    <div class="code-container">
        <div class="emulator-badge">🧪 EMULATOR MODE</div>
        <div class="logo">freee</div>
        <div class="success-icon">✅</div>
        <div class="message">認証が完了しました</div>
        <div class="hint">以下の認証コードをアプリケーションに入力してください</div>
        <div class="code" id="auth-code">%CODE%</div>
        <div class="hint">このコードは1回のみ有効です</div>
    </div>
</body>
</html>
`

// HandleLoginPage displays the email/password login page
func (h *Handler) HandleLoginPage(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("client_id")
	redirectURI := r.URL.Query().Get("redirect_uri")
	state := r.URL.Query().Get("state")

	// Store in session (simplified)
	sessionID := fmt.Sprintf("%s:%s:%s", clientID, redirectURI, state)

	content := fmt.Sprintf(`
		<div class="form-group">
			<label for="email">メールアドレス</label>
			<input type="email" id="email" name="email" required autofocus>
		</div>
		<div class="form-group">
			<label for="password">パスワード</label>
			<input type="password" id="password" name="password" required>
		</div>
		<input type="hidden" name="session_id" value="%s">
	`, sessionID)

	html := loginPageHTML
	html = replaceAll(html, "%STEP%", "1")
	html = replaceAll(html, "%ACTION%", "/oauth/authorize/login")
	html = replaceAll(html, "%CONTENT%", content)
	html = replaceAll(html, "%BUTTON_TEXT%", "ログイン")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// HandleLogin processes the login form
func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")
	sessionID := r.FormValue("session_id")

	// Simple validation (test@example.com / password)
	if email != "test@example.com" || password != "password" {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	// Show 2FA page
	content := fmt.Sprintf(`
		<div class="form-group">
			<label for="otp">認証コード（6桁）</label>
			<input type="text" id="otp" name="otp" required autofocus maxlength="6" placeholder="123456">
		</div>
		<input type="hidden" name="session_id" value="%s">
	`, sessionID)

	html := loginPageHTML
	html = replaceAll(html, "%STEP%", "2")
	html = replaceAll(html, "%ACTION%", "/oauth/authorize/2fa")
	html = replaceAll(html, "%CONTENT%", content)
	html = replaceAll(html, "%BUTTON_TEXT%", "認証")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// Handle2FA processes the 2FA form
func (h *Handler) Handle2FA(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	otp := r.FormValue("otp")
	sessionID := r.FormValue("session_id")

	// Simple validation (any 6 digits, or specifically "123456")
	if len(otp) != 6 || otp != "123456" {
		http.Error(w, "Invalid OTP code", http.StatusUnauthorized)
		return
	}

	// Show authorization page
	html := authorizePageHTML
	html = replaceAll(html, "%SESSION_ID%", sessionID)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// HandleAuthorizeConfirm processes the authorization
func (h *Handler) HandleAuthorizeConfirm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	_ = r.FormValue("session_id") // sessionID would be used for production validation

	// Generate authorization code
	code := "AUTH_CODE_" + generateSimpleToken(16)

	// Parse session to get redirect_uri and state
	// sessionID format: clientID:redirectURI:state
	// For OOB flow, show the code directly

	html := codePageHTML
	html = replaceAll(html, "%CODE%", code)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func replaceAll(s, old, new string) string {
	result := ""
	for {
		idx := indexOf(s, old)
		if idx == -1 {
			result += s
			break
		}
		result += s[:idx] + new
		s = s[idx+len(old):]
	}
	return result
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func generateSimpleToken(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := ""
	for i := 0; i < length; i++ {
		result += string(chars[i%len(chars)])
	}
	return result
}

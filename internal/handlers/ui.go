package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"

	"github.com/virtuos/languagetool-user-proxy/internal/apikey"
	"github.com/virtuos/languagetool-user-proxy/internal/config"
	"github.com/virtuos/languagetool-user-proxy/internal/oidc"
	"github.com/virtuos/languagetool-user-proxy/internal/session"
)

type UIHandler struct {
	OIDCProvider     *oidc.Provider
	SessionManager   *session.Manager
	APIKeyManager    *apikey.Manager
	AccentColorStart string
	AccentColorEnd   string
	FrontendURL      string
}

type DashboardData struct {
	APIKey           string
	HasAPIKey        bool
	RegenError       string
	AccentColorStart string
	AccentColorEnd   string
	FrontendURL      string
}

func NewUIHandler(oidcProvider *oidc.Provider, sessionManager *session.Manager, apiKeyManager *apikey.Manager, cfg *config.Config) *UIHandler {
	return &UIHandler{
		OIDCProvider:     oidcProvider,
		SessionManager:   sessionManager,
		APIKeyManager:    apiKeyManager,
		AccentColorStart: cfg.UIAccentColorStart,
		AccentColorEnd:   cfg.UIAccentColorEnd,
		FrontendURL:      cfg.FrontendURL,
	}
}

func (h *UIHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sessionToken := h.SessionManager.GetTokenFromRequest(r)
	if sessionToken == "" {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	sess, err := h.SessionManager.GetSessionByToken(ctx, sessionToken)
	if err != nil {
		h.SessionManager.ClearSessionCookie(w)
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	apiKey, err := h.APIKeyManager.GetAPIKeyByUserID(ctx, sess.UserID)
	if err != nil {
		apiKey = ""
	}

	data := DashboardData{
		APIKey:           apiKey,
		HasAPIKey:        apiKey != "",
		AccentColorStart: h.AccentColorStart,
		AccentColorEnd:   h.AccentColorEnd,
		FrontendURL:      h.FrontendURL,
	}

	tmpl := template.Must(template.New("dashboard").Parse(dashboardHTML))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, data)
}

func (h *UIHandler) RegenerateKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	sessionToken := h.SessionManager.GetTokenFromRequest(r)
	if sessionToken == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	sess, err := h.SessionManager.GetSessionByToken(ctx, sessionToken)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	newKey, err := h.APIKeyManager.RegenerateAPIKey(ctx, sess.UserID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to regenerate key: %v", err), http.StatusInternalServerError)
		return
	}

	// Return the new key as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"key": newKey,
	})
}

func (h *UIHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	h.SessionManager.ClearSessionCookie(w)
	http.Redirect(w, r, "/", http.StatusFound)
}

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>LanguageTool Proxy - Dashboard</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            background: linear-gradient(135deg, {{.AccentColorStart}} 0%, {{.AccentColorEnd}} 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
        .container {
            background: white;
            border-radius: 16px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            padding: 40px;
            max-width: 600px;
            width: 100%;
        }
        h1 {
            color: #333;
            margin-bottom: 10px;
            font-size: 28px;
        }
        .subtitle {
            color: #666;
            margin-bottom: 30px;
            font-size: 14px;
        }
        .api-key-section {
            background: #f8f9fa;
            border-radius: 8px;
            padding: 20px;
            margin-bottom: 20px;
        }
        .api-key-label {
            font-size: 12px;
            color: #666;
            text-transform: uppercase;
            letter-spacing: 1px;
            margin-bottom: 8px;
        }
        .api-key {
            font-family: 'Courier New', monospace;
            font-size: 16px;
            color: #333;
            word-break: break-all;
            background: white;
            padding: 12px;
            border-radius: 4px;
            border: 1px solid #ddd;
        }
        .no-key {
            color: #999;
            font-style: italic;
        }
        .button-group {
            display: flex;
            gap: 10px;
            flex-wrap: wrap;
        }
        button {
            padding: 12px 24px;
            border: none;
            border-radius: 8px;
            font-size: 14px;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.2s;
        }
        .btn-primary {
            background: linear-gradient(135deg, {{.AccentColorStart}} 0%, {{.AccentColorEnd}} 100%);
            color: white;
        }
        .btn-primary:hover {
            transform: translateY(-2px);
            box-shadow: 0 4px 12px {{.AccentColorStart}}cc;
        }
        .btn-secondary {
            background: #f0f0f0;
            color: #333;
        }
        .btn-secondary:hover {
            background: #e0e0e0;
        }
        .btn-danger {
            background: #e74c3c;
            color: white;
        }
        .btn-danger:hover {
            background: #c0392b;
        }
        .info-box {
            background: #e8f4fd;
            border-left: 4px solid {{.AccentColorStart}};
            padding: 15px;
            margin-bottom: 20px;
            border-radius: 4px;
        }
        .info-box p {
            color: {{.AccentColorStart}};
            font-size: 14px;
            line-height: 1.5;
        }
        .info-box code {
            background: {{.AccentColorStart}}1a;
            padding: 2px 6px;
            border-radius: 3px;
            font-family: 'Courier New', monospace;
        }
        .error {
            background: #fdeaea;
            border-left: 4px solid #e74c3c;
            padding: 15px;
            margin-bottom: 20px;
            border-radius: 4px;
        }
        .error p {
            color: #c0392b;
            font-size: 14px;
        }
        .new-key-modal {
            display: none;
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background: rgba(0,0,0,0.5);
            align-items: center;
            justify-content: center;
            z-index: 1000;
        }
        .new-key-modal.active {
            display: flex;
        }
        .modal-content {
            background: white;
            padding: 30px;
            border-radius: 12px;
            max-width: 500px;
            width: 90%;
        }
        .modal-content h2 {
            margin-bottom: 15px;
            color: #333;
        }
        .modal-content .new-key {
            font-family: 'Courier New', monospace;
            font-size: 14px;
            background: #f8f9fa;
            padding: 15px;
            border-radius: 8px;
            word-break: break-all;
            margin: 15px 0;
        }
        .modal-content .warning {
            color: #e74c3c;
            font-size: 13px;
            margin-bottom: 20px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>LanguageTool Proxy</h1>
        <p class="subtitle">Your personal API endpoint</p>

        <div class="info-box">
            <p>Your API endpoint:<br>
            <code>{{.FrontendURL}}/{{.APIKey}}/v2/check</code></p>
        </div>

        <div class="api-key-section">
            <div class="api-key-label">Your API Key</div>
            {{if .HasAPIKey}}
                <div class="api-key">{{.APIKey}}</div>
            {{else}}
                <div class="api-key no-key">No API key generated yet</div>
            {{end}}
        </div>

        <div class="button-group">
            <button class="btn-primary" onclick="showRegenerateModal()">Regenerate API Key</button>
            <form method="POST" action="/logout" style="display: inline;">
                <button type="submit" class="btn-secondary">Logout</button>
            </form>
        </div>
    </div>

    <div class="new-key-modal" id="regenModal">
        <div class="modal-content">
            <h2>Generate New API Key</h2>
            <p class="warning">⚠️ Your old API key will be invalidated immediately. Make sure to update your applications with the new key.</p>
            <div class="new-key" id="newKeyDisplay"></div>
            <div class="button-group">
                <button class="btn-primary" onclick="copyKey()">Copy to Clipboard</button>
                <button class="btn-secondary" onclick="closeModal()">Close</button>
            </div>
        </div>
    </div>

    <script>
        function showRegenerateModal() {
            fetch('/regenerate-key', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
            })
            .then(response => response.json())
            .then(data => {
                document.getElementById('newKeyDisplay').textContent = data.key;
                document.getElementById('regenModal').classList.add('active');
            })
            .catch(error => {
                alert('Failed to regenerate key: ' + error);
            });
        }

        function closeModal() {
            document.getElementById('regenModal').classList.remove('active');
        }

        function copyKey() {
            const key = document.getElementById('newKeyDisplay').textContent;
            navigator.clipboard.writeText(key).then(() => {
                alert('API key copied to clipboard!');
            });
        }

        document.getElementById('regenModal').addEventListener('click', function(e) {
            if (e.target === this) {
                closeModal();
            }
        });
    </script>
</body>
</html>`

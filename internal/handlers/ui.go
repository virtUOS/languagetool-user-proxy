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
	Username         string
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

	apiKey, err := h.APIKeyManager.GetAPIKeyByUserID(ctx, sess.Session.UserID)
	if err != nil {
		apiKey = ""
	}

	data := DashboardData{
		APIKey:           apiKey,
		HasAPIKey:        apiKey != "",
		Username:         sess.User.Name.String,
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

	newKey, err := h.APIKeyManager.RegenerateAPIKey(ctx, sess.Session.UserID)
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
            padding-top: 10px;
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
        .button-group {
            display: flex;
            gap: 10px;
            flex-wrap: wrap;
            justify-content: center;
            margin: 20px;
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
        .btn-logout {
            background: #f0f0f0;
            color: #333;
            padding: 8px 16px;
            font-size: 13px;
        }
        .btn-logout:hover {
            background: #e0e0e0;
        }
        .user-bar {
            display: flex;
            justify-content: flex-end;
            align-items: center;
            margin-bottom: 30px;
        }
        .user-bar-text {
            color: #333;
            font-size: 14px;
            margin-right: 12px;
        }
        .info-box {
            background: #e8f4fd;
            border-left: 4px solid {{.AccentColorStart}};
            padding: 15px;
            margin: 20px;
            border-radius: 4px;
            color: {{.AccentColorStart}};
            font-size: 14px;
            line-height: 1.5;
        }
        .info-box code {
            background: {{.AccentColorStart}}1a;
            padding: 2px 6px;
            border-radius: 3px;
            font-family: 'Courier New', monospace;
            word-break: break-all;
            display: block;
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
        .user-bar {
            display: flex;
            justify-content: flex-end;
            align-items: center;
            margin-bottom: 30px;
        }
        .user-bar-text {
            color: #333;
            font-size: 14px;
            margin-right: 12px;
        }
        .header {
            display: flex;
            justify-content: center;
            align-items: center;
            margin-bottom: 20px;
        }
        .header-title {
            color: #333;
            margin-bottom: 5px;
            font-size: 24px;
            text-align: center;
        }
        .instructions-section {
            margin-top: 30px;
            padding-top: 20px;
            border-top: 1px solid #e0e0e0;
            font-size: 14px;
            color: #555;
        }
        .instructions-section h2 {
            color: #333;
            font-size: 18px;
            margin-bottom: 15px;
        }
        .instructions-section ol, .instructions-section ul, .instruction-section p {
            margin-left: 20px;
            margin-bottom: 15px;
        }
        .instructions-section li {
            line-height: 1.7;
        }
        .instructions-section a {
            color: {{.AccentColorStart}};
            text-decoration: none;
        }
        .instructions-section a:hover {
            text-decoration: underline;
        }
        .instructions-section > code {
        	display: block;
            background: #e8f4fd;
            padding: 5px 10px;
            margin: 20px;
            border-radius: 3px;
            font-family: 'Courier New', monospace;
            border-left: 2px solid gray;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="user-bar">
            <span class="user-bar-text">Logged in as {{.Username}}</span>
            <form method="POST" action="/logout">
                <button type="submit" class="btn-logout">Logout</button>
            </form>
        </div>
        <div class="header">
            <div>
                <h1 class="header-title">LanguageTool Proxy</h1>
            </div>
        </div>

        <div class="info-box">
            Your API Key:
            {{if .HasAPIKey}}
                <code id="apiKeyDisplay">{{.APIKey}}</code>
            {{else}}
                <code id="apiKeyDisplay">No API key generated yet</code>
            {{end}}
        </div>

        <div class="button-group">
            <button class="btn-primary" id="regenerateBtn" onclick="handleRegenerate()">Generate new API key</button>
        </div>

        <p>
        LanguageTool will need an API endpoint to use this private server.
        For this, every user gets a personal API endpoint.
        </p>

        <div class="info-box">
            Your personal API endpoint:
            {{if .HasAPIKey}}
                <code id="endpointDisplay">{{.FrontendURL}}/{{.APIKey}}/v2/</code>
            {{else}}
                <code id="endpointDisplay">No API key generated yet</code>
            {{end}}
        </div>

        <div class="instructions-section">
            <h2>Setting up LanguageTool in your browser</h2>
            <ol>
                <li>Install the LanguageTool Addon for <a href="https://addons.mozilla.org/en-US/firefox/addon/languagetool/" target="_blank">Firefox</a> or <a href="https://chromewebstore.google.com/detail/ai-grammar-checker-paraph/oldceeleldhonbafppcapldpdifcinji" target="_blank">Chrome</a>.</li>
                <li>Open LanguageTool Addon Settings</li>
                <li>Scroll down to "Advanced settings (only for professional users)"</li>
                <li>In "LanguageTool server"
                    <ul>
                        <li>Select "Other server"</li>
                        <li>Paste in your personal API endpoint</li>
                    </ul>
                </li>
            </ol>

            <h2>Disable LanguageTool (optional)</h2>
            <p>
                To prevent accidentally sending data to the public servers,
                you can disable all access from your local machine to that server.
                For that, under Linux or macOS, edit <code>/etc/hosts</code>
                and configure:
            </p>
            <code>127.0.0.2   languagetool.org</code>
        </div>
    </div>

    <script>
        function handleRegenerate() {
            if (!confirm('Generating a new API key will invalidate your current key. Continue?')) {
                return;
            }

            fetch('/regenerate-key', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
            })
            .then(response => response.json())
            .then(data => {
                // Update API key display with full key
                document.getElementById('apiKeyDisplay').textContent = data.key;
                // Update endpoint display with full key
                const endpoint = '{{.FrontendURL}}/' + data.key + '/v2/';
                document.getElementById('endpointDisplay').textContent = endpoint;
            })
            .catch(error => {
                alert('Failed to regenerate key: ' + error);
            });
        }
    </script>
</body>
</html>`

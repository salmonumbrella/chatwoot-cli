package auth

// chatwootLogoSVG is the Chatwoot logo used in both templates.
// Note: The viewBox and basic SVG attributes are included here; wrapper attributes
// (class, width/height for sizing) are added by each template.
const chatwootLogoSVG = `<circle cx="256" cy="256" fill="#47A7F6" r="256"/><path d="M362.807947,368.807947 L244.122956,368.807947 C178.699407,368.807947 125.456954,315.561812 125.456954,250.12177 C125.456954,184.703089 178.699407,131.456954 244.124143,131.456954 C309.565494,131.456954 362.807947,184.703089 362.807947,250.12177 L362.807947,368.807947 Z" fill="#FFFFFF" stroke="#FFFFFF" stroke-width="6"/>`

// githubIconSVG is the GitHub icon used in the footer of both templates.
const githubIconSVG = `<svg width="16" height="16" viewBox="0 0 16 16" fill="currentColor"><path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z"/></svg>`

// footerCSS contains the shared footer styles used in both templates.
const footerCSS = `
        .footer {
            text-align: center;
            margin-top: 2rem;
            font-size: 0.8125rem;
            color: var(--text-dim);
        }

        .footer a {
            color: var(--text-muted);
            text-decoration: none;
            transition: color 0.2s;
        }

        .footer a:hover {
            color: var(--chatwoot-blue);
        }

        .github-link {
            display: inline-flex;
            align-items: center;
            gap: 0.5rem;
        }

        .github-link svg {
            opacity: 0.7;
            transition: opacity 0.2s;
        }

        .github-link:hover svg {
            opacity: 1;
        }
`

// fadeUpAnimationCSS contains the shared fadeUp animation used in both templates.
const fadeUpAnimationCSS = `
        @keyframes fadeUp {
            from { opacity: 0; transform: translateY(10px); }
            to { opacity: 1; transform: translateY(0); }
        }
`

// baseCSSVars contains the CSS custom properties shared by both templates.
const baseCSSVars = `
        :root {
            --bg-deep: #06060a;
            --bg-card: #0d0d14;
            --bg-input: #12121a;
            --border: #1a1a2e;
            --text: #e4e4eb;
            --text-muted: #6b6b7a;
            --text-dim: #3d3d4a;
            --chatwoot-blue: #47A7F6;
            --success: #22c55e;
            --success-glow: rgba(34, 197, 94, 0.15);
        }
`

const setupTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Chatwoot CLI Setup</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600&family=Space+Grotesk:wght@400;500;600;700&display=swap" rel="stylesheet">
    <style>
` + baseCSSVars + `
        :root {
            --border-focus: #1f93ff;
            --accent: #1f93ff;
            --accent-glow: rgba(31, 147, 255, 0.15);
            --accent-hover: #3da3ff;
            --error: #ef4444;
            --error-glow: rgba(239, 68, 68, 0.15);
            --warning: #f59e0b;
        }

        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: 'Space Grotesk', -apple-system, sans-serif;
            background: var(--bg-deep);
            color: var(--text);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 2rem;
            position: relative;
            overflow-x: hidden;
        }

        /* Animated background grid */
        body::before {
            content: '';
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background-image:
                linear-gradient(rgba(71, 167, 246, 0.03) 1px, transparent 1px),
                linear-gradient(90deg, rgba(71, 167, 246, 0.03) 1px, transparent 1px);
            background-size: 60px 60px;
            pointer-events: none;
            z-index: 0;
        }

        /* Gradient orbs */
        body::after {
            content: '';
            position: fixed;
            top: -50%;
            left: -50%;
            width: 200%;
            height: 200%;
            background:
                radial-gradient(ellipse at 20% 30%, rgba(71, 167, 246, 0.08) 0%, transparent 50%),
                radial-gradient(ellipse at 80% 70%, rgba(139, 92, 246, 0.05) 0%, transparent 50%);
            pointer-events: none;
            z-index: 0;
            animation: orbFloat 20s ease-in-out infinite;
        }

        @keyframes orbFloat {
            0%, 100% { transform: translate(0, 0); }
            50% { transform: translate(-5%, 5%); }
        }

        .container {
            width: 100%;
            max-width: 520px;
            position: relative;
            z-index: 1;
        }

        /* Terminal header */
        .terminal-header {
            display: flex;
            align-items: center;
            gap: 0.75rem;
            margin-bottom: 2rem;
            padding-bottom: 1.5rem;
            border-bottom: 1px solid var(--border);
        }

        .terminal-prompt {
            font-family: 'JetBrains Mono', monospace;
            font-size: 0.875rem;
            color: var(--text-muted);
            display: flex;
            align-items: center;
            gap: 0.5rem;
        }

        .terminal-prompt::before {
            content: '$';
            color: var(--chatwoot-blue);
        }

        /* Logo */
        .logo-section {
            text-align: center;
            margin-bottom: 2.5rem;
        }

        .logo {
            width: 64px;
            height: 64px;
            margin-bottom: 1.25rem;
            display: inline-block;
        }

        .logo svg {
            width: 100%;
            height: 100%;
        }

        h1 {
            font-size: 1.75rem;
            font-weight: 600;
            letter-spacing: -0.02em;
            margin-bottom: 0.5rem;
        }

        .subtitle {
            color: var(--text-muted);
            font-size: 0.9375rem;
        }

        /* Card */
        .card {
            background: var(--bg-card);
            border: 1px solid var(--border);
            border-radius: 16px;
            padding: 2rem;
            box-shadow: 0 4px 24px rgba(0, 0, 0, 0.4);
        }

        /* Form */
        .form-group {
            margin-bottom: 1.5rem;
        }

        label {
            display: block;
            font-size: 0.8125rem;
            font-weight: 500;
            color: var(--text-muted);
            margin-bottom: 0.5rem;
            text-transform: uppercase;
            letter-spacing: 0.05em;
        }

        input {
            width: 100%;
            padding: 0.875rem 1rem;
            font-family: 'JetBrains Mono', monospace;
            font-size: 0.9375rem;
            background: var(--bg-input);
            border: 1px solid var(--border);
            border-radius: 10px;
            color: var(--text);
            transition: all 0.2s ease;
        }

        input::placeholder {
            color: var(--text-dim);
        }

        input:focus {
            outline: none;
            border-color: var(--border-focus);
            box-shadow: 0 0 0 3px var(--accent-glow);
        }

        input:hover:not(:focus) {
            border-color: #2a2a3e;
        }

        input.error {
            border-color: var(--error);
            box-shadow: 0 0 0 3px var(--error-glow);
        }

        input.error:focus {
            border-color: var(--error);
            box-shadow: 0 0 0 3px var(--error-glow);
        }

        /* Hide number input spinners */
        input[type="number"]::-webkit-outer-spin-button,
        input[type="number"]::-webkit-inner-spin-button {
            -webkit-appearance: none;
            margin: 0;
        }

        input[type="number"] {
            -moz-appearance: textfield;
        }

        .input-hint {
            font-size: 0.75rem;
            color: var(--text-dim);
            margin-top: 0.375rem;
            font-family: 'JetBrains Mono', monospace;
        }

        /* Buttons */
        .btn-group {
            display: flex;
            gap: 0.75rem;
            margin-top: 2rem;
        }

        button {
            flex: 1;
            padding: 0.875rem 1.5rem;
            font-family: 'Space Grotesk', sans-serif;
            font-size: 0.9375rem;
            font-weight: 500;
            border-radius: 10px;
            cursor: pointer;
            transition: all 0.2s ease;
            border: none;
        }

        .btn-secondary {
            background: transparent;
            border: 1px solid var(--border);
            color: var(--text-muted);
        }

        .btn-secondary:hover {
            background: var(--bg-input);
            border-color: #2a2a3e;
            color: var(--text);
        }

        .btn-primary {
            background: var(--chatwoot-blue);
            color: white;
            box-shadow: 0 4px 16px rgba(71, 167, 246, 0.25);
        }

        .btn-primary:hover {
            background: #5ab3f7;
            transform: translateY(-1px);
            box-shadow: 0 6px 20px rgba(71, 167, 246, 0.3);
        }

        .btn-primary:active {
            transform: translateY(0);
        }

        button:disabled {
            opacity: 0.5;
            cursor: not-allowed;
            transform: none !important;
        }

        /* Status messages */
        .status {
            margin-top: 1.5rem;
            padding: 1rem;
            border-radius: 10px;
            font-size: 0.875rem;
            display: none;
            align-items: center;
            gap: 0.75rem;
            font-family: 'JetBrains Mono', monospace;
        }

        .status.show {
            display: flex;
        }

        .status.loading {
            background: var(--accent-glow);
            border: 1px solid rgba(71, 167, 246, 0.2);
            color: var(--chatwoot-blue);
        }

        .status.success {
            background: var(--success-glow);
            border: 1px solid rgba(34, 197, 94, 0.2);
            color: var(--success);
        }

        .status.error {
            background: var(--error-glow);
            border: 1px solid rgba(239, 68, 68, 0.2);
            color: var(--error);
        }

        .spinner {
            width: 16px;
            height: 16px;
            border: 2px solid currentColor;
            border-top-color: transparent;
            border-radius: 50%;
            animation: spin 0.8s linear infinite;
        }

        @keyframes spin {
            to { transform: rotate(360deg); }
        }

        /* Help section */
        .help-section {
            margin-top: 2rem;
            padding-top: 1.5rem;
            border-top: 1px solid var(--border);
        }

        .help-title {
            font-size: 0.75rem;
            font-weight: 500;
            color: var(--text-dim);
            text-transform: uppercase;
            letter-spacing: 0.08em;
            margin-bottom: 1rem;
        }

        .help-item {
            display: flex;
            align-items: flex-start;
            gap: 0.75rem;
            margin-bottom: 0.875rem;
            font-size: 0.8125rem;
            color: var(--text-muted);
        }

        .help-item:last-child {
            margin-bottom: 0;
        }

        .help-icon {
            flex-shrink: 0;
            width: 20px;
            height: 20px;
            background: var(--bg-input);
            border-radius: 5px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-family: 'JetBrains Mono', monospace;
            font-size: 0.625rem;
            color: var(--text-dim);
        }

        .help-item code {
            font-family: 'JetBrains Mono', monospace;
            background: var(--bg-input);
            padding: 0.125rem 0.375rem;
            border-radius: 4px;
            font-size: 0.75rem;
            color: var(--chatwoot-blue);
        }

        /* Dynamic links */
        .hint-link, .help-link {
            color: var(--chatwoot-blue);
            text-decoration: none;
            border-bottom: 1px dashed rgba(71, 167, 246, 0.4);
            transition: all 0.2s ease;
        }

        .hint-link:hover, .help-link:hover {
            color: var(--accent-hover);
            border-bottom-color: var(--accent-hover);
        }

        .hint-link.disabled, .help-link.disabled {
            color: var(--text-dim);
            border-bottom-color: transparent;
            cursor: default;
            pointer-events: none;
        }

        /* Footer */
` + footerCSS + `
        /* Animations */
        .fade-in {
            animation: fadeIn 0.5s ease forwards;
        }

        @keyframes fadeIn {
            from { opacity: 0; transform: translateY(10px); }
            to { opacity: 1; transform: translateY(0); }
        }

        .card { animation-delay: 0.1s; opacity: 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="terminal-header">
            <div class="terminal-prompt">
                cw auth login
            </div>
        </div>

        <div class="logo-section fade-in">
            <div class="logo">
                <svg viewBox="0 0 512 512" xmlns="http://www.w3.org/2000/svg">
                    ` + chatwootLogoSVG + `
                </svg>
            </div>
            <h1>Connect to Chatwoot</h1>
            <p class="subtitle">Configure your CLI to interact with Chatwoot</p>
        </div>

        <div class="card fade-in">
            <form id="setupForm" autocomplete="off">
                <div class="form-group">
                    <label for="baseUrl">Instance URL</label>
                    <input
                        type="url"
                        id="baseUrl"
                        name="baseUrl"
                        placeholder="https://app.chatwoot.com"
                        value="https://app.chatwoot.com"
                        required
                    >
                    <div class="input-hint">Your Chatwoot instance URL (cloud or self-hosted)</div>
                </div>

                <div class="form-group">
                    <label for="accountId">Account ID</label>
                    <input
                        type="number"
                        id="accountId"
                        name="accountId"
                        placeholder="1"
                        min="1"
                        required
                    >
                    <div class="input-hint">Found in your URL: <a href="#" id="accountsLink" target="_blank" class="hint-link">/app/accounts/<strong>ID</strong>/...</a></div>
                </div>

                <div class="form-group">
                    <label for="apiToken">API Token</label>
                    <input
                        type="password"
                        id="apiToken"
                        name="apiToken"
                        placeholder="Enter your API access token"
                        required
                    >
                    <div class="input-hint">
                        <a href="#" id="profileSettingsLink" target="_blank" class="hint-link">Profile Settings</a> &rarr; Access Token
                    </div>
                </div>

                <div class="btn-group">
                    <button type="button" id="testBtn" class="btn-secondary">Test Connection</button>
                    <button type="submit" id="submitBtn" class="btn-primary">Save & Connect</button>
                </div>

                <div id="status" class="status"></div>
            </form>

            <div class="help-section">
                <div class="help-title">Where to find your credentials</div>
                <div class="help-item">
                    <span class="help-icon">1</span>
                    <span>Log in to your Chatwoot dashboard</span>
                </div>
                <div class="help-item">
                    <span class="help-icon">2</span>
                    <span>Go to <a href="#" id="helpSettingsLink" target="_blank" class="help-link">Profile Settings</a></span>
                </div>
                <div class="help-item">
                    <span class="help-icon">3</span>
                    <span>Copy your <a href="#" id="helpTokenLink" target="_blank" class="help-link">Access Token</a> from the page</span>
                </div>
            </div>
        </div>

        <div class="footer fade-in" style="animation-delay: 0.2s; opacity: 0;">
            <a href="https://github.com/salmonumbrella/chatwoot-cli" target="_blank" class="github-link">
                ` + githubIconSVG + `
                View on GitHub
            </a>
        </div>
    </div>

    <script>
        const form = document.getElementById('setupForm');
        const testBtn = document.getElementById('testBtn');
        const submitBtn = document.getElementById('submitBtn');
        const status = document.getElementById('status');
        const csrfToken = '{{.CSRFToken}}';

        function showStatus(type, message) {
            status.className = 'status show ' + type;
            if (type === 'loading') {
                status.innerHTML = '<div class="spinner"></div><span>' + message + '</span>';
            } else {
                const icon = type === 'success' ? '&#10003;' : '&#10007;';
                status.innerHTML = '<span>' + icon + '</span><span>' + message + '</span>';
            }
        }

        function hideStatus() {
            status.className = 'status';
        }

        function getFormData() {
            return {
                base_url: document.getElementById('baseUrl').value.trim().replace(/\/$/, ''),
                api_token: document.getElementById('apiToken').value.trim(),
                account_id: parseInt(document.getElementById('accountId').value, 10)
            };
        }

        // Dynamic links
        const baseUrlInput = document.getElementById('baseUrl');
        const accountIdInput = document.getElementById('accountId');
        const profileSettingsLink = document.getElementById('profileSettingsLink');
        const helpSettingsLink = document.getElementById('helpSettingsLink');
        const helpTokenLink = document.getElementById('helpTokenLink');
        const accountsLink = document.getElementById('accountsLink');

        function updateDynamicLinks() {
            const baseUrl = baseUrlInput.value.trim().replace(/\/$/, '');
            const accountId = accountIdInput.value.trim();
            const profileLinks = [profileSettingsLink, helpSettingsLink, helpTokenLink];

            // Profile settings links (need both URL and account ID)
            if (baseUrl && accountId && !isNaN(parseInt(accountId))) {
                const settingsUrl = baseUrl + '/app/accounts/' + accountId + '/profile/settings';
                profileLinks.forEach(link => {
                    link.href = settingsUrl;
                    link.classList.remove('disabled');
                });
            } else {
                profileLinks.forEach(link => {
                    link.href = '#';
                    link.classList.add('disabled');
                });
            }

            // Accounts link (just needs URL - takes user to dashboard to see account ID)
            if (baseUrl) {
                // If they have an account ID, go to that account's dashboard
                // Otherwise go to app root which will redirect to their default account
                if (accountId && !isNaN(parseInt(accountId))) {
                    accountsLink.href = baseUrl + '/app/accounts/' + accountId + '/dashboard';
                } else {
                    accountsLink.href = baseUrl + '/app';
                }
                accountsLink.classList.remove('disabled');
            } else {
                accountsLink.href = '#';
                accountsLink.classList.add('disabled');
            }
        }

        baseUrlInput.addEventListener('input', updateDynamicLinks);
        accountIdInput.addEventListener('input', updateDynamicLinks);
        updateDynamicLinks(); // Initialize on load

        function validateFields() {
            const fields = [
                { el: document.getElementById('baseUrl'), value: document.getElementById('baseUrl').value.trim() },
                { el: document.getElementById('accountId'), value: document.getElementById('accountId').value.trim() },
                { el: document.getElementById('apiToken'), value: document.getElementById('apiToken').value.trim() }
            ];
            let valid = true;
            fields.forEach(f => {
                if (!f.value) {
                    f.el.classList.add('error');
                    valid = false;
                } else {
                    f.el.classList.remove('error');
                }
            });
            return valid;
        }

        // Clear error state on input
        document.querySelectorAll('input').forEach(input => {
            input.addEventListener('input', () => input.classList.remove('error'));
        });

        testBtn.addEventListener('click', async () => {
            hideStatus();
            if (!validateFields()) return;

            const data = getFormData();
            testBtn.disabled = true;
            submitBtn.disabled = true;
            showStatus('loading', 'Testing connection...');

            try {
                const response = await fetch('/validate', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'X-CSRF-Token': csrfToken
                    },
                    body: JSON.stringify(data)
                });

                const result = await response.json();

                if (result.success) {
                    showStatus('success', 'Connected as ' + result.user_name + ' (' + result.user_email + ')');
                } else {
                    showStatus('error', result.error);
                }
            } catch (err) {
                showStatus('error', 'Request failed: ' + err.message);
            } finally {
                testBtn.disabled = false;
                submitBtn.disabled = false;
            }
        });

        form.addEventListener('submit', async (e) => {
            e.preventDefault();
            hideStatus();
            if (!validateFields()) return;

            const data = getFormData();
            testBtn.disabled = true;
            submitBtn.disabled = true;
            showStatus('loading', 'Saving credentials...');

            try {
                const response = await fetch('/submit', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'X-CSRF-Token': csrfToken
                    },
                    body: JSON.stringify(data)
                });

                const result = await response.json();

                if (result.success) {
                    showStatus('success', 'Credentials saved! Redirecting...');
                    setTimeout(() => {
                        window.location.href = '/success?name=' + encodeURIComponent(result.user_name) + '&email=' + encodeURIComponent(result.user_email);
                    }, 1000);
                } else {
                    showStatus('error', result.error);
                    testBtn.disabled = false;
                    submitBtn.disabled = false;
                }
            } catch (err) {
                showStatus('error', 'Request failed: ' + err.message);
                testBtn.disabled = false;
                submitBtn.disabled = false;
            }
        });
    </script>
</body>
</html>`

const successTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Setup Complete - Chatwoot CLI</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600&family=Space+Grotesk:wght@400;500;600;700&display=swap" rel="stylesheet">
    <style>
` + baseCSSVars + `
        :root {
            --chatwoot-glow: rgba(71, 167, 246, 0.2);
        }

        * { margin: 0; padding: 0; box-sizing: border-box; }

        body {
            font-family: 'Space Grotesk', sans-serif;
            background: var(--bg-deep);
            color: var(--text);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 2rem;
            position: relative;
        }

        body::before {
            content: '';
            position: fixed;
            top: 0; left: 0; right: 0; bottom: 0;
            background-image:
                linear-gradient(rgba(71, 167, 246, 0.02) 1px, transparent 1px),
                linear-gradient(90deg, rgba(71, 167, 246, 0.02) 1px, transparent 1px);
            background-size: 60px 60px;
            pointer-events: none;
        }

        body::after {
            content: '';
            position: fixed;
            top: -50%; left: -50%;
            width: 200%; height: 200%;
            background: radial-gradient(ellipse at 50% 50%, var(--chatwoot-glow) 0%, transparent 50%);
            pointer-events: none;
            animation: pulse 4s ease-in-out infinite;
        }

        @keyframes pulse {
            0%, 100% { opacity: 0.5; transform: scale(1); }
            50% { opacity: 0.8; transform: scale(1.05); }
        }

        .container {
            width: 100%;
            max-width: 560px;
            position: relative;
            z-index: 1;
            text-align: center;
        }

        .logo {
            width: 80px;
            height: 80px;
            margin: 0 auto 2rem;
            display: block;
            animation: scaleIn 0.5s cubic-bezier(0.34, 1.56, 0.64, 1) forwards;
            filter: drop-shadow(0 8px 40px var(--chatwoot-glow));
        }

        @keyframes scaleIn {
            from { transform: scale(0); }
            to { transform: scale(1); }
        }

        h1 {
            font-size: 2rem;
            font-weight: 600;
            letter-spacing: -0.02em;
            margin-bottom: 0.5rem;
            animation: fadeUp 0.5s ease 0.2s both;
        }

        .subtitle {
            color: var(--text-muted);
            font-size: 1rem;
            margin-bottom: 2.5rem;
            animation: fadeUp 0.5s ease 0.3s both;
        }

        .user-badge {
            display: inline-flex;
            align-items: center;
            gap: 0.5rem;
            background: var(--bg-card);
            border: 1px solid var(--border);
            border-radius: 100px;
            padding: 0.5rem 1rem;
            font-size: 0.875rem;
            color: var(--text-muted);
            margin-bottom: 2.5rem;
            animation: fadeUp 0.5s ease 0.35s both;
        }

        .user-badge .dot {
            width: 8px;
            height: 8px;
            background: var(--success);
            border-radius: 50%;
            animation: dotPulse 2s ease-in-out infinite;
        }

        @keyframes dotPulse {
            0%, 100% { opacity: 1; }
            50% { opacity: 0.5; }
        }

` + fadeUpAnimationCSS + `
        /* Terminal card */
        .terminal {
            background: var(--bg-card);
            border: 1px solid var(--border);
            border-radius: 16px;
            overflow: hidden;
            text-align: left;
            animation: fadeUp 0.5s ease 0.4s both;
            box-shadow: 0 4px 32px rgba(0, 0, 0, 0.4);
        }

        .terminal-bar {
            background: var(--bg-input);
            padding: 0.75rem 1rem;
            display: flex;
            align-items: center;
            gap: 0.5rem;
            border-bottom: 1px solid var(--border);
        }

        .terminal-dot {
            width: 12px;
            height: 12px;
            border-radius: 50%;
        }

        .terminal-dot.red { background: #ff5f57; }
        .terminal-dot.yellow { background: #febc2e; }
        .terminal-dot.green { background: #28c840; }

        .terminal-title {
            flex: 1;
            text-align: center;
            font-family: 'JetBrains Mono', monospace;
            font-size: 0.75rem;
            color: var(--text-dim);
        }

        .terminal-body {
            padding: 1.5rem;
        }

        .terminal-line {
            display: flex;
            align-items: center;
            gap: 0.5rem;
            font-family: 'JetBrains Mono', monospace;
            font-size: 0.875rem;
            margin-bottom: 1rem;
        }

        .terminal-line:last-child {
            margin-bottom: 0;
        }

        .terminal-prompt {
            color: var(--chatwoot-blue);
            user-select: none;
        }

        .terminal-text {
            color: var(--text);
        }

        .terminal-cursor {
            display: inline-block;
            width: 10px;
            height: 20px;
            background: var(--chatwoot-blue);
            animation: cursorBlink 1.2s step-end infinite;
            margin-left: 2px;
            vertical-align: middle;
        }

        @keyframes cursorBlink {
            0%, 50% { opacity: 1; }
            50.01%, 100% { opacity: 0; }
        }

        .terminal-output {
            color: var(--success);
            padding-left: 1.25rem;
            margin-top: -0.5rem;
            margin-bottom: 1rem;
        }

        .terminal-comment {
            color: var(--text-dim);
            font-style: italic;
        }

        /* Message */
        .message {
            margin-top: 2rem;
            padding: 1.25rem;
            background: rgba(71, 167, 246, 0.08);
            border: 1px solid rgba(71, 167, 246, 0.15);
            border-radius: 12px;
            animation: fadeUp 0.5s ease 0.5s both;
        }

        .message-icon {
            font-size: 1.5rem;
            margin-bottom: 0.5rem;
        }

        .message-title {
            font-weight: 600;
            margin-bottom: 0.25rem;
            color: var(--text);
        }

        .message-text {
            font-size: 0.875rem;
            color: var(--text-muted);
        }

        .message-text code {
            font-family: 'JetBrains Mono', monospace;
            background: var(--bg-input);
            padding: 0.2rem 0.5rem;
            border-radius: 6px;
            font-size: 0.8125rem;
            color: var(--chatwoot-blue);
            border: 1px solid var(--border);
        }

` + footerCSS + `
        /* Override footer with animation for success page */
        .footer {
            animation: fadeUp 0.5s ease 0.6s both;
        }
    </style>
</head>
<body>
    <div class="container">
        <svg class="logo" viewBox="0 0 512 512" xmlns="http://www.w3.org/2000/svg">
            ` + chatwootLogoSVG + `
        </svg>

        <h1>You're all set!</h1>
        <p class="subtitle">Chatwoot CLI is now connected and ready to use</p>

        {{if .UserEmail}}
        <div class="user-badge">
            <span class="dot"></span>
            <span>{{.UserEmail}}</span>
        </div>
        {{end}}

        <div class="terminal">
            <div class="terminal-bar">
                <span class="terminal-dot red"></span>
                <span class="terminal-dot yellow"></span>
                <span class="terminal-dot green"></span>
                <span class="terminal-title"></span>
            </div>
            <div class="terminal-body">
                <div class="terminal-line">
                    <span class="terminal-prompt">$</span>
                    <span class="terminal-text">chatwoot conversations list</span>
                </div>
                <div class="terminal-output">Fetching conversations...</div>
                <div class="terminal-line">
                    <span class="terminal-prompt">$</span>
                    <span class="terminal-text">chatwoot contacts search "john"</span>
                </div>
                <div class="terminal-output">Found 3 contacts</div>
                <div class="terminal-line">
                    <span class="terminal-prompt">$</span>
                    <span class="terminal-cursor"></span>
                </div>
            </div>
        </div>

        <div class="message">
            <div class="message-icon">&#8592;</div>
            <div class="message-title">Return to your terminal</div>
            <div class="message-text">You can close this window and start using the CLI. Try running <code>chatwoot --help</code> to see all available commands.</div>
        </div>

        <div class="footer">
            <a href="https://github.com/salmonumbrella/chatwoot-cli" target="_blank" class="github-link">
                ` + githubIconSVG + `
                View on GitHub
            </a>
        </div>
    </div>

    <script>
        // Signal completion to server
        fetch('/complete', { method: 'POST' }).catch(() => {});
    </script>
</body>
</html>`

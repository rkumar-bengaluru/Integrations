# Gmail Authorization Code Flow

> Complete guide for implementing OAuth 2.0 Authorization Code Flow with Gmail API

---

## 📋 Table of Contents

- [Architecture Overview](#architecture-overview)
- [Prerequisites](#prerequisites)
- [Implementation Steps](#implementation-steps)
- [Security Best Practices](#security-best-practices)
- [Code Examples](#code-examples)
- [Troubleshooting](#troubleshooting)

---

## 🏗️ Architecture Overview

```
┌─────────────┐         ┌─────────────┐         ┌─────────────┐
│   Client    │         │    User     │         │   Google    │
│ Application │         │   Browser   │         │   OAuth     │
│   (Your     │         │             │         │   Server    │
│    App)     │         │             │         │             │
└──────┬──────┘         └──────┬──────┘         └──────┬──────┘
       │                       │                       │
       │ 1️⃣ Redirect to Google │                       │
       │──────────────────────>│                       │
       │                       │ 2️⃣ User logs in       │
       │                       │    & consents         │
       │                       │──────────────────────>│
       │                       │                       │
       │                       │ 3️⃣ Authorization Code │
       │                       │<──────────────────────│
       │                       │                       │
       │ 4️⃣ Code + Redirect    │                       │
       │<─────────────────────│                       │
       │                       │                       │
       │ 5️⃣ Exchange Code      │                       │
       │    for Tokens         │                       │
       │──────────────────────────────────────────────>│
       │                       │                       │
       │ 6️⃣ Access + Refresh   │                       │
       │    Tokens             │                       │
       │<──────────────────────────────────────────────│
       │                       │                       │
       │ 7️⃣ API Calls with     │                       │
       │    Access Token       │                       │
       │──────────────────────────────────────────────>│
```

### Flow Description

| Step | Action | Description |
|------|--------|-------------|
| 1 | **Authorization Request** | App redirects user to Google's OAuth server |
| 2 | **User Authentication** | User logs in and grants permissions |
| 3 | **Authorization Code** | Google issues temporary authorization code |
| 4 | **Callback** | User redirected back to app with code |
| 5 | **Token Exchange** | App exchanges code for tokens (server-side) |
| 6 | **Token Response** | Google returns access & refresh tokens |
| 7 | **API Access** | App uses access token to call Gmail API |

---

## 🚀 Prerequisites

### 1. Google Cloud Project Setup

#### Enable APIs
1. Navigate to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select existing one
3. Enable the **Gmail API**:
   ```
   APIs & Services → Library → Search "Gmail API" → Enable
   ```

#### Configure OAuth Consent Screen
```
APIs & Services → OAuth consent screen
```

| Setting | Value |
|---------|-------|
| **User Type** | External (for public apps) / Internal (for Workspace orgs) |
| **App Name** | Your application name |
| **User Support Email** | Support contact email |
| **Developer Contact** | Technical contact email |

#### Create OAuth 2.0 Credentials
```
APIs & Services → Credentials → Create Credentials → OAuth client ID
```

**Required Information:**
- **Application type**: Web application
- **Authorized redirect URIs**: `https://yourapp.com/oauth/callback`
- **Download**: `client_secrets.json`

> ⚠️ **Security Warning**: Never commit `client_secrets.json` to version control. Add it to `.gitignore`.

---

## 🔧 Implementation Steps

### Step 1: Build Authorization URL

Construct the authorization endpoint URL with required parameters:

```http
GET https://accounts.google.com/o/oauth2/v2/auth
```

**Query Parameters:**

| Parameter | Required | Description |
|-----------|----------|-------------|
| `client_id` | ✅ | Your OAuth 2.0 Client ID |
| `redirect_uri` | ✅ | Must match registered URI exactly |
| `response_type` | ✅ | Always `code` for this flow |
| `scope` | ✅ | Space-delimited permissions (see [Scopes](#available-scopes)) |
| `access_type` | ❌ | `offline` to receive refresh token |
| `state` | ❌ | **Recommended** - CSRF protection token |
| `include_granted_scopes` | ❌ | `true` to include previously granted scopes |
| `prompt` | ❌ | `consent` to force re-consent |

**Example URL:**
```http
https://accounts.google.com/o/oauth2/v2/auth?
  scope=https://www.googleapis.com/auth/gmail.readonly&
  access_type=offline&
  include_granted_scopes=true&
  response_type=code&
  state=random_security_token_12345&
  redirect_uri=https://yourapp.com/oauth/callback&
  client_id=123456789-abc123.apps.googleusercontent.com
```

#### Available Scopes

| Scope | Access Level | Use Case |
|-------|--------------|----------|
| `gmail.readonly` | Read-only | Read messages, threads, labels |
| `gmail.modify` | Read/Write | All read operations + modify/delete messages |
| `gmail.compose` | Send only | Send messages, create drafts |
| `gmail.send` | Send only | Send messages only |
| `gmail.labels` | Labels management | Create/update/delete labels |
| `gmail.metadata` | Metadata only | Access message metadata (no body) |
| `gmail.settings.basic` | Settings | Manage basic mail settings |
| `gmail.settings.sharing` | Sharing settings | Manage delegates and forwarding |

> 🔒 **Best Practice**: Request minimal scopes. You can request additional scopes later.

---

### Step 2: Handle User Consent

After authorization, Google redirects to your callback URL:

**Success Response:**
```http
GET https://yourapp.com/oauth/callback?
  code=4/P7q7W91a-oMsCeLvIaQm6bTrgtp7&
  state=random_security_token_12345
```

**Error Response:**
```http
GET https://yourapp.com/oauth/callback?
  error=access_denied&
  state=random_security_token_12345
```

**Validate State Parameter:**
```python
# Verify state matches the one you sent
if request.params.get('state') != session['oauth_state']:
    raise SecurityError("Invalid state parameter - possible CSRF attack")
```

---

### Step 3: Exchange Code for Tokens

Make a server-side POST request to exchange the authorization code:

```http
POST https://oauth2.googleapis.com/token
Content-Type: application/x-www-form-urlencoded
```

**Request Body:**
```http
code=4/P7q7W91a-oMsCeLvIaQm6bTrgtp7&
client_id=YOUR_CLIENT_ID&
client_secret=YOUR_CLIENT_SECRET&
redirect_uri=https://yourapp.com/oauth/callback&
grant_type=authorization_code
```

**Success Response (200 OK):**
```json
{
  "access_token": "ya29.a0AfH6SMB...",
  "expires_in": 3599,
  "refresh_token": "1//0d7Xl5eThD7qCgYIARAAGA0SNwF-L9Ir...",
  "scope": "https://www.googleapis.com/auth/gmail.readonly",
  "token_type": "Bearer"
}
```

**Token Details:**

| Token | Duration | Purpose |
|-------|----------|---------|
| `access_token` | 1 hour (3599 seconds) | Short-lived token for API calls |
| `refresh_token` | Until revoked by user | Long-lived token to get new access tokens |

> ⚠️ **Important**: Store the refresh token securely. It's only returned on the first authorization or when `prompt=consent` is used.

---

### Step 4: Make Gmail API Calls

Include the access token in the Authorization header:

```http
GET https://gmail.googleapis.com/gmail/v1/users/me/messages
Authorization: Bearer ya29.a0AfH6SMB...
```

**Example API Endpoints:**

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/users/me/messages` | GET | List messages |
| `/users/me/messages/{id}` | GET | Get specific message |
| `/users/me/threads` | GET | List threads |
| `/users/me/labels` | GET | List labels |
| `/users/me/messages/{id}/modify` | POST | Modify message labels |

---

### Step 5: Refresh Access Token

When the access token expires (or before), use the refresh token:

```http
POST https://oauth2.googleapis.com/token
Content-Type: application/x-www-form-urlencoded
```

**Request Body:**
```http
client_id=YOUR_CLIENT_ID&
client_secret=YOUR_CLIENT_SECRET&
refresh_token=1//0d7Xl5eThD7qCgYIARAAGA0SNwF-L9Ir...&
grant_type=refresh_token
```

**Response:**
```json
{
  "access_token": "ya29.a0AfH6SMC...",
  "expires_in": 3599,
  "token_type": "Bearer",
  "scope": "https://www.googleapis.com/auth/gmail.readonly"
}
```

> 📝 **Note**: A new refresh token is not returned in this response. Continue using the original refresh token.

---

## 🛡️ Security Best Practices

### Token Storage

| Do ✅ | Don't ❌ |
|-------|----------|
| Store tokens encrypted at rest | Store tokens in localStorage/sessionStorage |
| Use secure, httpOnly cookies for session | Expose tokens in client-side JavaScript |
| Implement token rotation | Hardcode tokens in source code |
| Use environment variables for secrets | Commit credentials to version control |

### CSRF Protection

Always implement state parameter validation:

```python
import secrets

# Generate state
state = secrets.token_urlsafe(32)
session['oauth_state'] = state

# Validate on callback
if request.args.get('state') != session.pop('oauth_state', None):
    abort(403, "Invalid state parameter")
```

### PKCE Extension (Recommended for SPAs/Mobile)

For public clients (SPAs, mobile apps), use PKCE:

```
1. Generate code_verifier (random string)
2. Create code_challenge = BASE64URL(SHA256(code_verifier))
3. Send code_challenge with authorization request
4. Send code_verifier with token exchange
```

---

## 💻 Code Examples

### Python (Flask + Google Auth Library)

```python
from flask import Flask, redirect, request, session, abort
from google_auth_oauthlib.flow import Flow
import os

app = Flask(__name__)
app.secret_key = os.environ['SECRET_KEY']

# Path to client_secrets.json downloaded from Google Cloud
CLIENT_SECRETS_FILE = "client_secrets.json"
SCOPES = ['https://www.googleapis.com/auth/gmail.readonly']
REDIRECT_URI = 'https://yourapp.com/oauth/callback'

@app.route('/auth/gmail')
def authorize_gmail():
    """Step 1: Redirect to Google OAuth"""
    flow = Flow.from_client_secrets_file(
        CLIENT_SECRETS_FILE,
        scopes=SCOPES,
        redirect_uri=REDIRECT_URI
    )
    
    authorization_url, state = flow.authorization_url(
        access_type='offline',
        include_granted_scopes='true',
        prompt='consent'  # Force to get refresh token every time
    )
    
    session['state'] = state
    return redirect(authorization_url)

@app.route('/oauth/callback')
def oauth_callback():
    """Step 2-3: Handle callback and exchange code"""
    # Validate state
    if request.args.get('state') != session.get('state'):
        abort(403, "Invalid state parameter")
    
    flow = Flow.from_client_secrets_file(
        CLIENT_SECRETS_FILE,
        scopes=SCOPES,
        redirect_uri=REDIRECT_URI,
        state=session['state']
    )
    
    # Exchange authorization code for tokens
    flow.fetch_token(authorization_response=request.url)
    credentials = flow.credentials
    
    # Store tokens securely (example: database)
    store_tokens(
        user_id=session['user_id'],
        access_token=credentials.token,
        refresh_token=credentials.refresh_token,
        expires_at=credentials.expiry
    )
    
    return "Authorization successful!"

def get_gmail_service(user_id):
    """Step 4: Use stored tokens to make API calls"""
    tokens = get_tokens_from_db(user_id)
    
    from google.oauth2.credentials import Credentials
    from googleapiclient.discovery import build
    
    creds = Credentials(
        token=tokens['access_token'],
        refresh_token=tokens['refresh_token'],
        token_uri="https://oauth2.googleapis.com/token",
        client_id=os.environ['GOOGLE_CLIENT_ID'],
        client_secret=os.environ['GOOGLE_CLIENT_SECRET'],
        scopes=SCOPES
    )
    
    # Auto-refresh if expired
    if creds.expired and creds.refresh_token:
        creds.refresh(Request())
        update_access_token(user_id, creds.token)
    
    return build('gmail', 'v1', credentials=creds)

@app.route('/emails')
def list_emails():
    """Example: List user's emails"""
    service = get_gmail_service(session['user_id'])
    results = service.users().messages().list(userId='me', maxResults=10).execute()
    messages = results.get('messages', [])
    return {'messages': messages}
```

### Node.js (Express + googleapis)

```javascript
const express = require('express');
const { google } = require('googleapis');
const crypto = require('crypto');

const app = express();

const oauth2Client = new google.auth.OAuth2(
  process.env.GOOGLE_CLIENT_ID,
  process.env.GOOGLE_CLIENT_SECRET,
  'https://yourapp.com/oauth/callback'
);

const SCOPES = ['https://www.googleapis.com/auth/gmail.readonly'];

// Step 1: Redirect to Google
app.get('/auth/gmail', (req, res) => {
  const state = crypto.randomBytes(32).toString('hex');
  req.session.state = state;
  
  const authUrl = oauth2Client.generateAuthUrl({
    access_type: 'offline',
    scope: SCOPES,
    state: state,
    include_granted_scopes: true,
    prompt: 'consent'
  });
  
  res.redirect(authUrl);
});

// Step 2-3: Handle callback
app.get('/oauth/callback', async (req, res) => {
  // Validate state
  if (req.query.state !== req.session.state) {
    return res.status(403).send('Invalid state parameter');
  }
  
  const { tokens } = await oauth2Client.getToken(req.query.code);
  
  // Store tokens securely
  await storeTokens(req.session.userId, {
    accessToken: tokens.access_token,
    refreshToken: tokens.refresh_token,
    expiryDate: tokens.expiry_date
  });
  
  res.send('Authorization successful!');
});

// Step 4: Make API calls
async function getGmailService(userId) {
  const tokens = await getTokensFromDb(userId);
  
  oauth2Client.setCredentials({
    access_token: tokens.accessToken,
    refresh_token: tokens.refreshToken
  });
  
  // Auto-refresh handler
  oauth2Client.on('tokens', (newTokens) => {
    if (newTokens.access_token) {
      updateAccessToken(userId, newTokens.access_token);
    }
  });
  
  return google.gmail({ version: 'v1', auth: oauth2Client });
}

app.get('/emails', async (req, res) => {
  const gmail = await getGmailService(req.session.userId);
  const response = await gmail.users.messages.list({ userId: 'me', maxResults: 10 });
  res.json(response.data);
});
```

---

## 🔍 Troubleshooting

### Common Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `invalid_grant` | Code expired (valid for 10 mins) or already used | Request new authorization code |
| `redirect_uri_mismatch` | Redirect URI doesn't match registered URI | Check exact match including protocol and path |
| `invalid_client` | Wrong client ID or secret | Verify credentials from Google Cloud Console |
| `unauthorized_client` | App not authorized or consent screen not configured | Complete OAuth consent screen setup |
| `access_denied` | User denied permission | Handle gracefully, allow retry |
| `insufficient_permissions` | Scope not granted or token lacks permission | Check requested scopes, re-authorize if needed |

### Debug Checklist

- [ ] Client ID and Secret are correct and from the same project
- [ ] Gmail API is enabled in Google Cloud Console
- [ ] OAuth consent screen is configured and published (if external)
- [ ] Redirect URI matches exactly (including trailing slashes)
- [ ] Authorization code is used within 10 minutes
- [ ] HTTPS is used for all redirect URIs in production

---

## 📚 Additional Resources

- [Google OAuth 2.0 Documentation](https://developers.google.com/identity/protocols/oauth2)
- [Gmail API Reference](https://developers.google.com/gmail/api/reference/rest)
- [OAuth 2.0 Playground](https://developers.google.com/oauthplayground)
- [Google Cloud Console](https://console.cloud.google.com/)

---

## 📄 License

This documentation is provided as-is for educational purposes.

---

*Last updated: 2026-03-19*

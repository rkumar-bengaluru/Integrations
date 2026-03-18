┌─────────────────┐         ┌─────────────────────────┐         ┌─────────────────┐
│   GitHub App    │ ──JWT──►│  GitHub API (App-level) │ ──Token─►│  Installation   │
│  (Private Key)  │         │  /app/installations/... │         │  Access Token   │
└─────────────────┘         └─────────────────────────┘         └─────────────────┘
                                                                          │
                                                                          ▼
                                                                 ┌─────────────────┐
                                                                 │  Repository/Org │
                                                                 │    Resources    │
                                                                 └─────────────────┘


Step 1: Create a GitHub App
Registration
Go to Settings → Developer settings → GitHub Apps → New GitHub App
Configure:
App Name: Your app's identifier
Homepage URL: Your app's website
Webhook URL: Optional, for receiving events
Permissions: Select specific scopes (e.g., Contents: Read, Issues: Write)
Subscribe to events: Check relevant webhooks
Critical: Generate Private Key
After creation, generate a private key (PEM format). This is your app's identity.

github-app-private-key.pem
-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA...
-----END RSA PRIVATE KEY-----

Step 2: JWT Authentication (App-Level)
The JWT proves your app's identity to GitHub. It's used for app-level operations:
List installations
Generate installation tokens
Access app-level APIs

| Claim | Description | Value                            |
| ----- | ----------- | -------------------------------- |
| `iat` | Issued at   | Unix timestamp (now)             |
| `exp` | Expiration  | `iat` + max 600 seconds (10 min) |
| `iss` | Issuer      | Your GitHub App ID (numeric)     |


Step 3: Get Installation Access Token
The JWT alone cannot access repository data. You must exchange it for an installation token.
What is an Installation?
When someone installs your app on their repo/org, GitHub creates an installation (identified by installation_id).

Step 4: Use Installation Token
Now use the ghs_ token like a regular Personal Access Token:



┌─────────────────────────────────────────────────────────────────────────┐
│                         YOUR SERVER/APPLICATION                         │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  1. GENERATE JWT (every 10 min max)                                     │
│     ├── Load private key (github-app-private-key.pem)                   │
│     ├── Create payload: {iat, exp, iss}                                 │
│     └── Sign with RS256 → JWT string                                    │
│                                                                         │
│  2. DISCOVER INSTALLATION (if needed)                                   │
│     └── GET /app/installations                                          │
│         Authorization: Bearer {jwt}                                     │
│         → Returns list of installations with IDs                        │
│                                                                         │
│  3. GET INSTALLATION TOKEN (every hour or per-request)                  │
│     └── POST /app/installations/{id}/access_tokens                      │
│         Authorization: Bearer {jwt}                                     │
│         → Returns {token: "ghs_xxx", expires_at: "..."}                 │
│                                                                         │
│  4. MAKE API CALLS                                                      │
│     └── GET/POST/PUT/DELETE /repos/owner/repo/...                       │
│         Authorization: token {ghs_token}                                │
│         → Access granted based on app's permissions                     │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘



┌─────────────────────────────────────────┐
│         YOUR GITHUB APP                 │
│  ┌─────────────────────────────────┐    │
│  │  App ID: 123456 (yours alone)   │◄───┼── Created once by you
│  │  Private Key: *.pem               │    │
│  │  Permissions: issues:write, etc.  │  │
│  └─────────────────────────────────┘    │
└─────────────────────────────────────────┘
                    │
    ┌───────────────┼───────────────┐
    ▼               ▼               ▼
┌─────────┐    ┌─────────┐    ┌─────────┐
│Install #1│    │Install #2│    │Install #3│
│on your   │    │on Acme   │    │on Tech   │
│personal  │    │Corp org  │    │Corp org  │
│repo      │    │(Enterprise)│    │(Enterprise)│
│          │    │          │    │          │
│install_  │    │install_  │    │install_  │
│id: 111   │    │id: 222   │    │id: 333   │
└─────────┘    └─────────┘    └─────────┘



┌─────────────────────────────────────────────────────────────────┐
│                     YOUR PLATFORM                                │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  PLATFORM CREDENTIALS (Your GitHub App - Created Once)   │   │
│  │  • app_id: 123456                                        │   │
│  │  • private_key: ████████████████████ (stored securely)   │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        ▼                     ▼                     ▼
┌───────────────┐    ┌───────────────┐    ┌───────────────┐
│  ENTERPRISE A │    │  ENTERPRISE B │    │  ENTERPRISE C │
│               │    │               │    │               │
│ Step 1: Install│    │ Step 1: Install│    │ Step 1: Install│
│   your app    │    │   your app    │    │   your app    │
│   on org/repo │    │   on org/repo │    │   on org/repo │
│               │    │               │    │               │
│   install_id: │    │   install_id: │    │   install_id: │
│     111       │    │     222       │    │     333       │
│               │    │               │    │               │
│ Step 2: Create│    │ Step 2: Create│    │ Step 2: Create│
│   Code Review │    │   Code Review │    │   Code Review │
│   Agent in    │    │   Agent in    │    │   Agent in    │
│   your platform│   │   your platform│   │   your platform│
│               │    │               │    │               │
│   Agent A1    │    │   Agent B1    │    │   Agent C1    │
│   • install:  │    │   • install:  │    │   • install:  │
│     111       │    │     222       │    │     333       │
│   • owner:    │    │   • owner:    │    │   • owner:    │
│     acme-corp │    │     tech-giant│    │     startup   │
│   • repo:     │    │   • repo:     │    │   • repo:     │
│     web-app   │    │     api-core  │    │     mobile    │
│               │    │               │    │               │
│   Agent A2    │    │   Agent B2    │    │               │
│   • install:  │    │   • install:  │    │               │
│     111       │    │     222       │    │               │
│   • owner:    │    │   • owner:    │    │               │
│     acme-corp │    │     tech-giant│    │               │
│   • repo:     │    │   • repo:     │    │               │
│     backend   │    │     frontend  │    │               │
└───────────────┘    └───────────────┘    └───────────────┘




┌─────────────────┐     PR opened/edited     ┌─────────────────┐
│   GitHub Repo   │ ─────────────────────────►│  Your Platform  │
│  (Enterprise)   │  X-GitHub-Event: pull_request│  Webhook Handler │
└─────────────────┘                            └─────────────────┘
                                                        │
                                                        ▼
                                               ┌─────────────────┐
                                               │ Find Agent by:   │
                                               │ • installation_id│
                                               │ • repo name      │
                                               └─────────────────┘
                                                        │
                                                        ▼
                                               ┌─────────────────┐
                                               │ Get token via:   │
                                               │ oauthServer2Server│
                                               │ (JWT → Install)  │
                                               └─────────────────┘
                                                        │
                                                        ▼
                                               ┌─────────────────┐
                                               │ Review PR via API│
                                               │ POST /repos/.../ │
                                               │ pulls/{n}/reviews│
                                               └─────────────────┘




┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│  Enterprise │───►│   Install   │───►│   Copy      │───►│   Create    │
│   lands on  │    │   GitHub    │    │ Installation│    │   Agents    │
│   your docs │    │   App       │    │   ID from   │    │   in your   │
│             │    │   (GitHub UI)│   │   URL or    │    │   platform  │
│             │    │             │    │   webhook   │    │             │
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
                                                               │
                                                               ▼
                                                        ┌─────────────┐
                                                        │  Agent auto │
                                                        │  reviews PRs│
                                                        │  in their   │
                                                        │  repo 🎉    │
                                                        └─────────────┘




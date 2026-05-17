# 🔭 OpenScout

> Finds the **actually solvable** issues that match your level among thousands of GitHub issues and sends them to Telegram before your morning coffee is ready.

---

## What It Does

It runs automatically every day at 08:00:

1. The **Go backend** queries the GitHub GraphQL API and collects issues tagged `good first issue` / `help wanted`
2. It applies repository quality filters (README, license, contributor count, recent activity)
3. It sends issues to the **Python analyzer service**, which uses Gemini for complexity analysis
4. It selects the best 5 issues and sends them to **Telegram** with complexity scores
5. It skips issues already seen using records stored in PostgreSQL

---

## Architecture

```
GitHub GraphQL API
        │
        ▼
backend/cmd/openscout/main.go
        │
        ├── internal/adapter/github      ← issue collection and repo quality filters
        ├── internal/usecase             ← collection, analysis, notification flow
        ├── internal/adapter/http        ← Python analyzer client
        ├── internal/adapter/notification ← Telegram / email notifications
        └── internal/adapter/postgres    ← user, preference, and notification records
        │
        ▼
ai/service.py      ← Gemini-based issue analysis (JSON output)
        │
        ▼
PostgreSQL         ← skip issues seen before
```

**Why these technologies?**

- **GraphQL**: Fetches issues and repository quality data in a single query, while REST would need 3-4 requests
- **Gemini**: Fast and cost-effective analysis model for a daily workflow
- **Go backend**: Keeps scheduling, notifications, and data flow in one service
- **Python analyzer**: Isolates the LLM call and keeps the JSON output simple
- **PostgreSQL**: Central database for persistent cache and user data

---

## Setup

### 1. Clone the repository

```bash
git clone https://github.com/kullaniciadi/openscout.git
cd openscout
```

### 2. Create a virtual environment

```bash
python -m venv venv
source venv/bin/activate      # Windows: venv\Scripts\activate
pip install -r requirements.txt
```

### 3. Configure API keys

```bash
cp .env.example .env
```

Fill in the `.env` file:

```env
GITHUB_TOKEN=ghp_xxx          # github.com → Settings → Developer Settings → Tokens
GEMINI_API_KEY=xxx            # Google AI Studio / Gemini API
TELEGRAM_BOT_TOKEN=xxx        # @BotFather → /newbot
TELEGRAM_CHAT_ID=xxx          # See step 4
POSTGRES_USER=openscout
POSTGRES_PASSWORD=openscoutpass
POSTGRES_DB=openscout
```

### 4. Set up the Telegram bot

1. Find **@BotFather** on Telegram
2. Send `/newbot` and give it a name (for example, `OpenScout`)
3. Copy the token and add it to `.env`
4. Send any message to the bot
5. Find the chat ID:
   ```bash
   curl "https://api.telegram.org/bot<TOKEN>/getUpdates"
        # "chat":{"id": COPY_THIS_NUMBER
   ```
6. Add the chat ID to `.env`

### 5. Start the services

```bash
docker compose up --build
```

This command starts three services:

- `openscout` - Go backend and HTTP API
- `analyzer` - Python Gemini analyzer service
- `db` - PostgreSQL

### 6. Test it

```bash
# Backend health check
curl http://localhost:8080/health

# Analyzer health check
curl http://localhost:8000/health
```

If you want, you can also run the services locally one by one:

```bash
# Start the backend in its own terminal
cd backend && go run ./cmd/openscout

# Start the analyzer in a separate terminal
cd ai && uvicorn service:app --host 0.0.0.0 --port 8000
```

---

## Example Telegram Message

```
🔭 OpenScout — 15 January 2025
Today's 5 contribution opportunities:

1. Fix nil pointer in HTTP middleware
   📦 gin-gonic/gin  ⭐ 77,000
   🟢 Complexity: 2/5  ⚡ ~2h
   🔧 Go · HTTP
   💡 The error message is specific, and a single-file change is enough.
   🔗 View issue

────────────────────────────────
2. Add TypeScript types for config
   📦 vitejs/vite  ⭐ 65,000
   ...
```

---

## Roadmap

- [x] Go backend + Python analyzer split
- [x] PostgreSQL cache and user data
- [x] Cron-based daily collection and notification flow
- [ ] Web interface (language selection + email subscription)
- [ ] Supabase integration
- [ ] Email notifications (Resend)

---

## Contributing

See CONTRIBUTING.md if you want to open an issue or send a PR.

## License

MIT

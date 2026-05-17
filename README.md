# 🔭 OpenScout

> GitHub'daki binlerce issue arasından senin seviyene uygun, **gerçekten çözülebilir** olanları sabah kahven hazır olana kadar bulur ve Telegram'a atar.

---

## Ne Yapar?

Her sabah saat 08:00'de otomatik olarak çalışır:

1. **GitHub GraphQL API**'yi sorgular — Go, Python ve TypeScript projelerindeki `good first issue` ve `help wanted` etiketli issue'ları çeker
2. Repo kalite filtresi uygular (README, lisans, 10+ katkıcı, son 90 gün aktif)
3. **LLM (Gemini)** ile her issue'yu analiz eder: "Bu gerçekten yeni biri için çözülebilir mi?"
4. En uygun 5 issue'yu seçer, karmaşıklık skoruyla birlikte **Telegram'a** gönderir
5. Gönderilen issue'ları PostgreSQL cache'e kaydeder — ertesi gün tekrar göndermez

---

## Mimari

```
GitHub GraphQL API
        │
        ▼
github_client.py   ← repo kalite filtresi (README, lisans, katkıcı, aktiflik)
        │
        ▼
analyzer.py        ← Gemini ile karmaşıklık analizi (JSON çıktı)
        │
        ▼
cache.py           ← PostgreSQL — daha önce görülen issue'ları atla
        │
        ▼
notifier.py        ← Telegram bot bildirimi
        │
        ▼
scheduler.py       ← APScheduler cron (her gün 08:00 İstanbul saati)
```

**Neden bu teknolojiler?**

- **GraphQL**: Tek sorguda issue + repo kalite kriterleri — REST'te 3-4 istek gerekirdi
- **Gemini**: Hızlı ve ucuz analiz modeli — günlük çalışma için ideal
- **PostgreSQL**: Kalıcı cache ve kullanıcı verileri için merkezi veritabanı
- **APScheduler**: Hafif, Docker gerektirmez — basit cron alternatifi

---

## Kurulum

### 1. Repoyu klonla

```bash
git clone https://github.com/kullaniciadi/openscout.git
cd openscout
```

### 2. Sanal ortam oluştur

```bash
python -m venv venv
source venv/bin/activate      # Windows: venv\Scripts\activate
pip install -r requirements.txt
```

### 3. API anahtarlarını ayarla

```bash
cp .env.example .env
```

`.env` dosyasını doldur:

```env
GITHUB_TOKEN=ghp_xxx          # github.com → Settings → Developer Settings → Tokens
GEMINI_API_KEY=xxx            # Google AI Studio / Gemini API
TELEGRAM_BOT_TOKEN=xxx        # @BotFather → /newbot
TELEGRAM_CHAT_ID=xxx          # Adım 4'e bak
```

### 4. Telegram bot kurulumu

1. Telegram'da **@BotFather**'ı bul
2. `/newbot` yaz, isim ver (örn: `OpenScout`)
3. Token'ı kopyala → `.env`'e yaz
4. Bota bir mesaj at (herhangi bir şey)
5. Chat ID'yi bul:
   ```bash
   curl "https://api.telegram.org/bot<TOKEN>/getUpdates"
   # "chat":{"id": BU_SAYIYI_KOPYAlA
   ```
6. Chat ID'yi `.env`'e yaz

### 5. Test et

```bash
cd python-brain

# Bot bağlantısını test et
python main.py --test-bot

# GitHub'dan issue çek (analiz yapmadan)
python main.py --fetch-only

# Issue'ları analiz etmeden Telegram'a gönder
python main.py --send-raw-issues

# Tam pipeline'ı hemen çalıştır
python main.py --now

# Scheduler'ı başlat (her sabah 08:00)
python main.py
```

---

## Örnek Telegram Mesajı

```
🔭 OpenScout — 15 Ocak 2025
Bugün için 5 katkı fırsatı:

1. Fix nil pointer in HTTP middleware
   📦 gin-gonic/gin  ⭐ 77,000
   🟢 Zorluk: 2/5  ⚡ ~2h
   🔧 Go · HTTP
   💡 Hata mesajı tam olarak belirtilmiş, tek dosya değişikliği yeterli.
   🔗 Issue'yu gör

────────────────────────────────
2. Add TypeScript types for config
   📦 vitejs/vite  ⭐ 65,000
   ...
```

---

## Geliştirme Planı

- [x] Python pipeline (GitHub → LLM → Telegram)
- [x] PostgreSQL cache
- [x] APScheduler
- [ ] Go veri toplayıcı (paralel GraphQL sorguları)
- [ ] Web arayüzü (dil seçimi + mail abonelik)
- [ ] Supabase entegrasyonu
- [ ] Mail bildirimi (Resend)

---

## Katkı

Issue açmak veya PR göndermek için CONTRIBUTING.md'ye bak.

## Lisans

MIT

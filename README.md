# qb-auto
Automation for qBittorrent to manage torrent files, specifically Japan anime.

## Background

I periodically download anime that marked on anime list (watch list) using qBittorrent and tidy up them in my NAS.

### Components

- **Anime list** — [self-hosted watch list](https://github.com/ReinforceZwei/anime-list). Records have a Traditional Chinese title and flags like `downloaded`.
- **qBittorrent** — BT client. After files are copied to NAS, torrent gets a `done` tag and continues seeding while disk space allows.
- **qui** — Better frontend for qBittorrent, provides the API qb-auto uses.
- **NAS** — Stores anime under `anime\<Chinese title>\<original torrent folder name>`. Multiple seasons share the same title folder.

## Challenge

Downloaded anime folder names are arbitrary — they can contain release group tags, romanized or abbreviated titles, codec info, episode numbers, etc. The target NAS folder must use the full Traditional Chinese title (TW translation preferred).

Examples:
- `[LoliHouse] Acro Trip [01-12][WebRip 1080p HEVC-10bit AAC]`
- `[BDrip] Dan Da Dan S01 [Sakurato&7³ACG]`
- `[DBD-Raws][超超超超超喜欢你的100个女朋友 第二季][01-12TV全集]...`

## Automated workflow

```
qBittorrent ──► GET /api/torrent-complete?hash=&category=
                │
                ├─ [anime] ──► title_worker
                │                  ├─ qui: get torrent details
                │                  └─ DetermineAnimeTitle (see below)
                │              rsync_worker
                │                  ├─ rsync: copy files to NAS
                │                  ├─ qui: add "done" tag
                │                  └─ animelist: mark downloaded
                │              [status: pending_notify]
                │
                └─ [other] ──► [status: pending_notify]
                                    │
                                    ▼
                              notify_worker
                                  ├─ webhook: send notification
                                  └─ [status: done]
```

### Determine anime title

**Primary path (TMDb)**

1. LLM extracts the bare title from the torrent folder name
2. Search TMDb with the extracted title
3. LLM picks the best match from TMDb results
4. Fetch the zh-TW title from TMDb for the chosen show

**Wikipedia fallback** (triggered when TMDb returns no results, LLM finds no match, or the matched show has no zh-TW title)

5. Brave Search `"wikipedia <title>"` to find a Wikipedia page
6. Parse the Wikipedia URL to get the page lang and title
7. If not already a zh page, fetch language links to find the zh equivalent
8. Fetch the zh Wikipedia page content (wikitext)
9. LLM extracts Chinese title, original Japanese title, and official TW translation from wikitext
10. Retry TMDb with the original title; LLM confirms the best match
11. Return zh-TW title from TMDb (if available) or the TW/Chinese title from Wikipedia

**Anime list confirmation** (both paths)

12. Search anime list by the resolved title; LLM picks the best record
13. If no match found, iterate all unwatched+undownloaded records in chunks until matched
14. If still no match → stop, leave for human review

## Application design

- **Language**: Go
- **Framework**: [PocketBase](https://pocketbase.io) — embedded backend + SQLite DB
- **LLM**: [eino](https://github.com/cloudwego/eino) with OpenAI-compatible endpoint
- **HTTP client**: [resty v3](https://github.com/go-resty/resty)
- **TMDb**: [golang-tmdb](https://github.com/cyruzin/golang-tmdb)

### Project structure

```
qb-auto/
├── main.go                    # Entry point, PocketBase setup, client wiring
├── config/config.go           # Config from JSON file + env var overrides
├── models/job.go              # Job struct and status constants
│
├── migrations/                # PocketBase DB migrations
│
├── routes/
│   ├── torrent.go             # GET /api/torrent-complete?hash=&category=
│   └── anime_title.go         # POST /api/resolve-anime-title (debug/manual)
│
├── workers/
│   ├── title_worker.go        # Goroutine pool: determine anime title (anime jobs)
│   ├── rsync_worker.go        # Single worker: copy files to NAS via rsync
│   └── notify_worker.go       # Single worker: send webhook, mark job done
│
├── services/
│   ├── job_service.go         # Create/update job records in DB
│   └── anime_title.go         # DetermineAnimeTitle: full title resolution flow
│
├── clients/
│   ├── qui/qui.go             # qui API: get torrent info, add "done" tag
│   ├── animelist/animelist.go # Anime list API: search & mark downloaded
│   ├── tmdb/tmdb.go           # TMDb API wrapper
│   ├── brave/brave.go         # Brave Search API (Wikipedia fallback)
│   ├── wikipedia/wikipedia.go # Wikipedia Action API (langlinks + page content)
│   ├── rsync/rsync.go         # rsync daemon protocol wrapper
│   └── webhook/webhook.go     # Discord webhook notifications
│
└── llm/
    ├── llm.go                 # eino chat model client + all LLM helper methods
    └── prompts.go             # System prompt constants
```

### Job status flow

```
pending → processing_title → pending_rsync → pending_notify → done
                                                    ↑
                             (non-anime jobs start here)

any stage → error  (requires human intervention)
```

### Configuration

Config is loaded from `~/.config/qb-auto/config.json` with environment variable overrides. Run once to generate the template file.

| Key | Env var | Required | Description |
|-----|---------|----------|-------------|
| `llm_base_url` | `LLM_BASE_URL` | ✓ | OpenAI-compatible base URL |
| `llm_api_key` | `LLM_API_KEY` | ✓ | |
| `llm_model_name` | `LLM_MODEL_NAME` | ✓ | |
| `tmdb_api_key` | `TMDB_API_KEY` | ✓ | |
| `qui_base_url` | `QUI_BASE_URL` | ✓ | |
| `qui_api_key` | `QUI_API_KEY` | | |
| `qui_instance_id` | `QUI_INSTANCE_ID` | | Default: 1 |
| `animelist_base_url` | `ANIMELIST_BASE_URL` | ✓ | |
| `animelist_username` | `ANIMELIST_USERNAME` | | |
| `animelist_password` | `ANIMELIST_PASSWORD` | | |
| `rsync_host` | `RSYNC_HOST` | ✓ | |
| `rsync_module` | `RSYNC_MODULE` | ✓ | rsync daemon module (NAS share root) |
| `rsync_user` | `RSYNC_USER` | ✓ | |
| `rsync_password_file` | `RSYNC_PASSWORD_FILE` | ✓ | Path to plaintext password file |
| `rsync_port` | `RSYNC_PORT` | | Default: 873 |
| `nas_anime_base_path` | `NAS_ANIME_BASE_PATH` | ✓ | Folder inside rsync module, e.g. `anime` |
| `webhook_url` | `WEBHOOK_URL` | | Discord webhook URL |
| `brave_api_key` | `BRAVE_API_KEY` | | Enables Wikipedia fallback |
| `title_worker_count` | `TITLE_WORKER_COUNT` | | Default: 1 |
| `http_addr` | `HTTP_ADDR` | | Default: `127.0.0.1:8090` |

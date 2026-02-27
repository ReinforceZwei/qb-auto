# qb-auto
Automation for qBittorrent to manage torrent files, specifically Japan anime.

## Background

I periodically download anime that marked on anime list (watch list) using qBittorrent and tidy up them in my NAS.

### Anime list

[Anime list repositary](https://github.com/ReinforceZwei/anime-list)

My personal legacy project (self-hosted) for saving anime watch list. Provides basic API access. Record can have flags like completed or downloaded. Record title is arbitrary input, and I prefer full title in Traditional Chinese.

### qBittorrent

BT client for downloading anime. All downloading anime _should_ have a record exists in anime list. After the torrent download completed and all files had copied to NAS, torrent will be applied a tag "done" to indicate it can be deleted safely.

Torrent marked as done will continue seeding as long as there are enough disk free space.

### qui

Better frontend for qBittorrent. Provides API access to manage qBittorrent instance.

### NAS

Store all anime video files with specific folder structure.

```
anime\<Full anime title in Chinese>\<downloaded anime with folder name as is>

# For multiple seasons, they share same full anime title folder
anime\<Full anime title in Chinese>\<downloaded anime with folder name as is S1>
anime\<Full anime title in Chinese>\<downloaded anime with folder name as is S2>
anime\<Full anime title in Chinese>\<downloaded anime with folder name as is S3>
```

## Current workflow

1. qBittorrent all download completed
2. me copy all downloaded files to NAS specific temporary folder
3. me mark all copied files (torrent) as "done" in qBittorrent
4. for each item in temporary folder:
5. me determine which anime is it by folder name
6. me create folder with anime title in dest folder if not exist
7. me move the anime files in temporary folder to dest folder
8. me mark anime as downloaded in anime list
9. repeat until all items in temporary folder is handled

Example of downloaded anime folder name:
- `[LoliHouse] Acro Trip [01-12][WebRip 1080p HEVC-10bit AAC]`
- `[Lilith-Raws] Ikenaikyo`
- `[DBD-Raws][超超超超超喜欢你的100个女朋友][01-12TV全集+特典映像][1080P][BDRip][HEVC-10bit][简繁日双语外挂][FLAC][MKV]`
- `[DBD-Raws][超超超超超喜欢你的100个女朋友 第二季][01-12TV全集][1080P][BDRip][HEVC-10bit][简繁外挂][FLAC][MKV]`
- `[BDrip] Dan Da Dan S01 [Sakurato&7³ACG]`

Example of final folder structure:
- `anime\三者三葉\[VCB-Studio] Sansha Sanyou [Ma10p_1080p]`
- `anime\SPY×FAMILY\[BDrip] SPYxFAMILY S01 [Sakurato&7³ACG]`
- `anime\只要長得可愛，即使是變態你也喜歡嗎？\[Sakurato.sub][HenSuki][1-12 END][BIG5][1080P]`
- `anime\調教咖啡廳\[Snow-Raws] ブレンド・S`

From the example, we can see anime folder name format can be arbitrary:
- Have production team name
- Anime title can be abbreviation, romanization, in different language
- May have video/audio format information, subtitle information
- May have season or episode infomation

## Challenge point

### No standard folder naming for downloaded anime

Need to use LLM to extract and search full title online

### Chinese title can have different translation 

e.g. simplified chinese vs traditional chinese

For example 負けヒロインが多すぎる！
- English: Too Many LOSING Heroines!
- Traditional Chinese (TW): 敗北女角太多了！
- Simplified Chinese (CN): 敗北女主太多了！

Another example ワンルーム、日当たり普通、天使つき。
- Traditional Chinese (TW): 單人房、日照一般、附天使。
- Simplified Chinese (CN): 單間、光照尚好、附帶天使

Title in anime list usually use TW translation, but there might be edge cases.

For anime "負けヒロインが多すぎる！" in anime list, TW translation is used. That mean searching "敗北女主太多了" (CN translation) will return no result.

For anime final title that will be used in NAS folder, it should reference anime list. So in above example it will be "敗北女角太多了！"


## Automated workflow

1. qBittorrent download completed
2. qBittorrent trigger external program (e.g. shell script)
3. shell script call qb-auto API with torrent info
4. qb-auto call qui API to get torrent details
5. qb-auto determine destination folder name (follow existing naming pattern)
6. qb-auto execute `rsync` and copy torrent files to NAS
7. qb-auto call qui API to mark torrent as copied (apply tag `done`)
8. qb-auto call anime list API to mark anime as downloaded
9. qb-auto send webhook to notify torrent job completed

### Determine anime title

1. Ask LLM to extract title part from the downloaded anime folder name
2. Search through TMDb with the extracted title (could be abbreviation)
3a. If have results:
  i. Ask LLM to confirm which result is best match
  ii. Get full title in Traditional Chinese from TMDb
3b. If no results (or result not match what we search):
  i. Ask LLM to search online, targeting zh.wikipedia.org with language Traditional Chinese (TW)
  ii. Ask LLM to extract the full title from wikipedia
4. Search record in anime list using full title
5a. If have result:
  i. Title is confirmed correct, continue remaining workflow
5b. If no result:
  i. Title might be mismatched, stop processing and leave it to human (me)

## Application design

Golang

Use Pocketbase as framework for qb-auto API and database.

API:
- receive torrent download complete event from qBittorrent

Database:
- Store processing job, worklog

[eino](https://github.com/cloudwego/eino) for LLM framework

[golang-tmdb](https://github.com/cyruzin/golang-tmdb) for TMDb API

[resty (v3 beta)](https://github.com/go-resty/resty) for API call

### Suggested project structure

```
qb-auto/
├── main.go                    # Entry point, Pocketbase setup
├── go.mod
├── go.sum
│
├── migrations/                # Pocketbase DB migrations (already exists)
│   └── 1771918154_created_jobs.go
│
├── routes/                    # API route handlers (Pocketbase hooks/custom routes)
│   └── torrent.go             # GET /api/torrent-complete?hash=
│
├── workers/                   # Background worker logic
│   ├── title_worker.go        # Determine anime title & folder name (parallelizable)
│   └── rsync_worker.go        # Copy files to NAS via rsync (single worker)
│
├── services/                  # Orchestration / business logic
│   ├── job_service.go         # Create/update job records in DB
│   └── anime_title.go         # The "determine anime title" flow (LLM + TMDb + anime list)
│
├── clients/                   # External API clients (thin wrappers)
│   ├── qui/
│   │   └── qui.go             # qui API: get torrent info, add "done" tag
│   ├── animelist/
│   │   └── animelist.go       # anime-list API: search & mark downloaded
│   ├── tmdb/
│   │   └── tmdb.go            # TMDb API via golang-tmdb
│   ├── rsync/
│   │   └── rsync.go           # rsync binary wrapper (daemon protocol)
│   └── webhook/
│       └── webhook.go         # Send webhook notification
│
├── llm/                       # LLM integration via eino
│   ├── agent.go               # eino agent/chain setup
│   └── prompts.go             # Prompt templates (extract title, confirm match, etc.)
│
└── models/                    # Shared data structs / domain types
    └── job.go                 # Job struct, status constants
```

Key decisions explained:
- `routes/` — Pocketbase lets you register custom routes via app.OnBeforeServe(). Keep each route's handler here, separate from business logic.
- `workers/` — The two workers have different concurrency characteristics (title workers = multiple, rsync worker = single queue). Each gets its own file with its goroutine/channel logic.
- `services/` — Sits between routes/workers and external clients. anime_title.go encapsulates the full multi-step title determination flow (LLM extract → TMDb search → LLM confirm → anime list lookup) so workers just call one function.
- `clients/` — One sub-package per external system. Each is a thin wrapper focused only on talking to that system, no business logic. This makes them easy to mock/test. tmdb/ wraps golang-tmdb to hide its API surface. rsync/ wraps the rsync binary via os/exec using the rsync daemon protocol.
- `llm/` — Isolates all eino-specific code. prompts.go keeps prompt strings in one place so they're easy to iterate on.
- `models/` — Shared structs like Job (with status constants like pending, processing, done, error) that are referenced across layers without circular imports.

### Flow

```
qBittorrent → routes/torrent.go
            → services/job_service.go  (create job in DB)
            → workers/title_worker.go
                → clients/qui/         (get torrent details)
                → services/anime_title.go
                    → llm/             (extract title)
                    → clients/tmdb/    (search TMDb)
                    → llm/             (confirm match)
                    → clients/animelist/ (search record)
            → workers/rsync_worker.go
                → clients/rsync/       (copy files to NAS)
                → clients/qui/         (add "done" tag)
                → clients/animelist/   (mark downloaded)
                → clients/webhook/     (notify)
```

1. qBittorrent invoke shell script and call qb-auto API with torrent hash

GET /api/torrent-complete?hash=

2. Create job record in database

job status, torrent hash

3. Start worker (title worker and rsync worker)

Title worker determine anime title and final folder name (allow multiple workers)

Worker get torrent details from qui API (content path, name)

*invoke function* to get final anime title

update job status and title result to database

invoke rsync worker 

rsync worker copy file from local to remote NAS (single worker)

mark torrent as done with tag

mark downloaded in anime list

update job status to done

send webhook notification


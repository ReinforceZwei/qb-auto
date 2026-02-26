Authentication: API key
In header: `X-API-Key`

Add tag

```sh
curl 'http://192.168.1.29:7476/api/instances/1/torrents/bulk-action' \
  -X POST \
  -H 'User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:147.0) Gecko/20100101 Firefox/147.0' \
  -H 'Accept: */*' \
  -H 'Accept-Language: zh-TW,zh;q=0.9,en-US;q=0.8,en;q=0.7' \
  -H 'Accept-Encoding: gzip, deflate' \
  -H 'Referer: http://192.168.1.29:7476/instances/1' \
  -H 'X-Requested-With: XMLHttpRequest' \
  -H 'Content-Type: application/json' \
  -H 'Origin: http://192.168.1.29:7476' \
  -H 'Sec-GPC: 1' \
  -H 'Connection: keep-alive' \
  -H 'Cookie: qui_user_session=xxxxxxxxxxxxxxxxxxxxxxxxxxxx' \
  -H 'Priority: u=0' \
  --data-raw '{"hashes":["f9317dcffcf23bbca4e70cbe7545a960bfc2ca88"],"action":"addTags","tags":"done","selectAll":false}'
```

payload
```json
{"hashes":["f9317dcffcf23bbca4e70cbe7545a960bfc2ca88"],"action":"addTags","tags":"done","selectAll":false}
```

response
```json
{"message":"Bulk action completed successfully"}
```

Get torrent info

```sh
curl 'http://192.168.1.29:7476/api/instances/1/torrents?limit=1&filters=%7B%22expr%22%3A%22Hash+%3D%3D+%5C%22affa1c8196296e3f645d0b252468d98504699c1a%5C%22%22%2C%22status%22%3A%5B%5D%2C%22excludeStatus%22%3A%5B%5D%2C%22categories%22%3A%5B%5D%2C%22excludeCategories%22%3A%5B%5D%2C%22tags%22%3A%5B%5D%2C%22excludeTags%22%3A%5B%5D%2C%22trackers%22%3A%5B%5D%2C%22excludeTrackers%22%3A%5B%5D%7D' \
  --compressed \
  -H 'User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:147.0) Gecko/20100101 Firefox/147.0' \
  -H 'Accept: */*' \
  -H 'Accept-Language: zh-TW,zh;q=0.9,en-US;q=0.8,en;q=0.7' \
  -H 'Accept-Encoding: gzip, deflate' \
  -H 'Referer: http://192.168.1.29:7476/instances/1' \
  -H 'X-Requested-With: XMLHttpRequest' \
  -H 'Content-Type: application/json' \
  -H 'Sec-GPC: 1' \
  -H 'Connection: keep-alive' \
  -H 'Cookie: qui_user_session=33YH_aZiOSfC9HyR_3QzCUNJVHJAYmIkugH-K7rgrs8' \
  -H 'Priority: u=4'
```

response json
```json
{
  "torrents": [
    {
      "added_on": 1771339743,
      "amount_left": 0,
      "auto_tmm": true,
      "availability": -1,
      "category": "toxic",
      "comment": "",
      "completed": 2323657732,
      "completion_on": 1771339893,
      "created_by": "",
      "content_path": "/mnt/4tb/pool/toxic/[小丁(Fantasy Factory)] Fern.zip",
      "dl_limit": 0,
      "dlspeed": 0,
      "download_path": "",
      "downloaded": 2324383313,
      "downloaded_session": 2324383313,
      "eta": 8640000,
      "f_l_piece_prio": false,
      "force_start": false,
      "hash": "f9317dcffcf23bbca4e70cbe7545a960bfc2ca88",
      "infohash_v1": "f9317dcffcf23bbca4e70cbe7545a960bfc2ca88",
      "infohash_v2": "",
      "popularity": 0,
      "private": false,
      "last_activity": 1772004172,
      "magnet_uri": "magnet:?xt=urn:btih:f9317dcffcf23bbca4e70cbe7545a960bfc2ca88&dn=%5B%E5%B0%8F%E4%B8%81%28Fantasy%20Factory%29%5D%20Fern&tr=udp%3A%2F%2Ftracker.filemail.com%3A6969%2Fannounce&tr=https%3A%2F%2Ftracker.moeblog.cn%3A443%2Fannounce&tr=https%3A%2F%2Ftracker.pmman.tech%3A443%2Fannounce&tr=https%3A%2F%2Ftracker.zhuqiy.com%3A443%2Fannounce&tr=udp%3A%2F%2F6ahddutb1ucc3cp.ru%3A6969%2Fannounce&tr=udp%3A%2F%2Fleet-tracker.moe%3A1337%2Fannounce&tr=udp%3A%2F%2Fopen.dstud.io%3A6969%2Fannounce&tr=udp%3A%2F%2Ft.overflow.biz%3A6969%2Fannounce&tr=udp%3A%2F%2Ftracker-udp.gbitt.info%3A80%2Fannounce&tr=udp%3A%2F%2Ftracker.alaskantf.com%3A6969%2Fannounce&tr=udp%3A%2F%2Ftracker.bittor.pw%3A1337%2Fannounce&tr=http%3A%2F%2Fsukebei.tracker.wf%3A8888%2Fannounce&tr=udp%3A%2F%2Ftracker.qu.ax%3A6969%2Fannounce&tr=udp%3A%2F%2Ftracker.wepzone.net%3A6969%2Fannounce&tr=udp%3A%2F%2Fwepzone.net%3A6969%2Fannounce&tr=udp%3A%2F%2Fzer0day.ch%3A1337%2Fannounce&tr=udp%3A%2F%2Fopen.demonii.com%3A1337%2Fannounce&tr=udp%3A%2F%2Fopen.stealth.si%3A80%2Fannounce&tr=udp%3A%2F%2Ftracker.opentrackr.org%3A1337%2Fannounce&tr=udp%3A%2F%2Fexodus.desync.com%3A6969%2Fannounce&tr=udp%3A%2F%2Ftracker.torrent.eu.org%3A451%2Fannounce",
      "max_ratio": -1,
      "max_seeding_time": -1,
      "max_inactive_seeding_time": -1,
      "name": "[小丁(Fantasy Factory)] Fern",
      "num_complete": 20,
      "num_incomplete": 5,
      "num_leechs": 1,
      "num_seeds": 0,
      "priority": 0,
      "progress": 1,
      "ratio": 22.12194926130069,
      "ratio_limit": -2,
      "reannounce": 0,
      "save_path": "/mnt/4tb/pool/toxic",
      "seeding_time": 664305,
      "seeding_time_limit": -2,
      "inactive_seeding_time_limit": -2,
      "seen_complete": 1772003908,
      "seq_dl": false,
      "size": 2323657732,
      "state": "uploading",
      "super_seeding": false,
      "tags": "done",
      "time_active": 664454,
      "total_size": 2323657732,
      "tracker": "udp://exodus.desync.com:6969/announce",
      "trackers_count": 21,
      "up_limit": 0,
      "uploaded": 51419889714,
      "uploaded_session": 51419889714,
      "upspeed": 8,
      "trackers": null
    }
  ],
  "total": 1,
  "stats": {
    "total": 1,
    "downloading": 0,
    "seeding": 1,
    "paused": 0,
    "error": 0,
    "checking": 0,
    "totalDownloadSpeed": 0,
    "totalUploadSpeed": 8,
    "totalSize": 2323657732,
    "totalRemainingSize": 0,
    "totalSeedingSize": 2323657732
  },
  "counts": {
    "status": {
      "active": 10,
      "all": 34,
      "checking": 0,
      "completed": 34,
      "downloading": 0,
      "errored": 0,
      "inactive": 24,
      "moving": 0,
      "paused": 0,
      "resumed": 34,
      "running": 34,
      "seeding": 34,
      "stalled": 24,
      "stalled_downloading": 0,
      "stalled_uploading": 24,
      "stopped": 0,
      "tracker_down": 0,
      "unregistered": 0,
      "uploading": 34
    },
    "categories": {
      "anime": 15,
      "toxic": 19
    },
    "categorySizes": {
      "anime": 175049831494,
      "toxic": 81029064804
    },
    "tags": {
      "": 18,
      "done": 16
    },
    "tagSizes": {
      "": 78705407072,
      "done": 177373489226
    },
    "trackers": {
      "exodus.desync.com": 3,
      "nyaa.tracker.wf": 2,
      "open.stealth.si": 5,
      "sparkle.ghostchu-services.top": 1,
      "t.nyaatracker.com": 2,
      "tr.nyacat.pw": 3,
      "tracker.bt4g.com": 1,
      "tracker.dler.com": 1,
      "tracker.moeblog.cn": 12,
      "tracker.opentrackr.org": 1,
      "tracker.torrent.eu.org": 3
    },
    "trackerTransfers": {
      "exodus.desync.com": {
        "uploaded": 575520353876,
        "downloaded": 9157619132,
        "totalSize": 9151915936,
        "count": 3
      },
      "nyaa.tracker.wf": {
        "uploaded": 236832093602,
        "downloaded": 31405850392,
        "totalSize": 31402581028,
        "count": 2
      },
      "open.stealth.si": {
        "uploaded": 2141336566480,
        "downloaded": 60537982425,
        "totalSize": 60497395101,
        "count": 5
      },
      "sparkle.ghostchu-services.top": {
        "uploaded": 107237158004,
        "downloaded": 11596236822,
        "totalSize": 11592856022,
        "count": 1
      },
      "t.nyaatracker.com": {
        "uploaded": 50288966943,
        "downloaded": 39715222704,
        "totalSize": 39706195947,
        "count": 2
      },
      "tr.nyacat.pw": {
        "uploaded": 137508995230,
        "downloaded": 25262000675,
        "totalSize": 25258765164,
        "count": 3
      },
      "tracker.bt4g.com": {
        "uploaded": 50013986662,
        "downloaded": 7544884087,
        "totalSize": 7544125108,
        "count": 1
      },
      "tracker.dler.com": {
        "uploaded": 121854670737,
        "downloaded": 24379796582,
        "totalSize": 12999192155,
        "count": 1
      },
      "tracker.moeblog.cn": {
        "uploaded": 1153026643602,
        "downloaded": 53146620356,
        "totalSize": 53125619286,
        "count": 12
      },
      "tracker.opentrackr.org": {
        "uploaded": 26433868407,
        "downloaded": 602033234,
        "totalSize": 600967473,
        "count": 1
      },
      "tracker.torrent.eu.org": {
        "uploaded": 175139045924,
        "downloaded": 4202914197,
        "totalSize": 4199283078,
        "count": 3
      }
    },
    "total": 34
  },
  "categories": {
    "anime": {
      "name": "anime",
      "savePath": ""
    },
    "toxic": {
      "name": "toxic",
      "savePath": ""
    }
  },
  "tags": [
    "done"
  ],
  "serverState": {
    "alltime_dl": 434434489463,
    "alltime_ul": 4950732193747,
    "average_time_queue": 439,
    "connection_status": "connected",
    "dht_nodes": 1474,
    "dl_info_data": 218527787330,
    "dl_info_speed": 0,
    "dl_rate_limit": 0,
    "free_space_on_disk": 3409362808832,
    "global_ratio": "11.39",
    "queued_io_jobs": 0,
    "queueing": false,
    "read_cache_hits": "0",
    "read_cache_overload": "0",
    "refresh_interval": 1500,
    "total_buffers_size": 655360,
    "total_peer_connections": 45,
    "total_queued_size": 0,
    "total_wasted_session": 89361750,
    "up_info_data": 4763357826739,
    "up_info_speed": 978099,
    "up_rate_limit": 10240000,
    "use_alt_speed_limits": false,
    "use_subcategories": false,
    "write_cache_overload": "0"
  },
  "useSubcategories": false,
  "hasMore": false,
  "cacheMetadata": {
    "source": "fresh",
    "age": 0,
    "isStale": false,
    "nextRefresh": "2026-02-25T07:23:20Z"
  },
  "trackerHealthSupported": false,
  "isCrossInstance": false,
  "partialResults": false
}
```
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
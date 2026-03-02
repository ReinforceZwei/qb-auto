# 1. [planned, low priority] orphan torrent

What if a torrent didnt trigger torrent-complete event (for whatever reason)?

for non-anime category -> acceptable, _let it go~_

for anime category -> need to be handled

## Possible solution

### schedule scan orphan torrent (anime category)

call qui API and look for category is anime, tags not contain done, torrent status is download complete, hash not in jobs collection

then create job for them

# 2. [done] hacky way to config pocketbase listen address

we currently injecting listen address into args, but there should a better way to do that.

another AI agent has figured out how to config pb by looking at pb source code.

see its result in [how to config pocketbase](./HOW_TO_CONFIG_POCKETBASE.md)

follow the study result and update qb-auto to config pb in a correct way

# 3. [done] pb_data need to follow default config location

currently pb_data folder will stick with binary, but config json is stored in user config directory.

need to update default pb_data location to the same as config json

see [how to config pocketbase](./HOW_TO_CONFIG_POCKETBASE.md) for how to set default pb_data location

# 4. [done] one-click setup systemd service

qb-auto is intended to run directly on linux host machine (no docker, no vm)

i want a command (e.g. `qb-auto install`) to help me setup systemd service and auto-startup

the service is installed as systemd template unit, run under my user account.

i can control it as `sudo systemctl start qb-auto@myuser`

# 5. [done] save tmdb id into anime list

we have tmdb id determined for an anime list record. 

save the tmdb id into anime list `url` property

this is not related to qb-auto itself, but anime list is planned to be re-written and integrate with tmdb (i.e. record will have tmdb id linked). saving tmdb id here can reduce future migration work

## Implementation note

there is existing `MarkDownloaded` function. avoid two seperate update request call in worker. combine multiple properties update into single request. anime list update API support multiple properties

# 6. [TBC] replace qui API with qbittorrent API

qui API wraps around qbittorrent API with additional advanced feature. however qb-auto doesnt require advanced qui feature. consider direct API call to qbittorrent

pros: no extra dependency on thrid party API (qui) - qb-auto deal with qbittorrent, not qui

cons: qbittorrent uses username + password and cookie based session id, while qui API provide clean API key based authentication
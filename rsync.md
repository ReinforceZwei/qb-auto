# rsync client selection

dest NAS support both ssh and rsync daemon

both require authentication

ssh support password and key authentication

## 1. go package https://github.com/gokrazy/rsync

pros: no system binary required, all done in golang

after looking at the source code, it doesnt support authentication in rsync daemon mode (not implemented). it use system ssh client to open ssh connection, so same pros and cons as 2.

## 2. rsync binary + ssh

pros: better security, passwordless login (using ssh key)

cons: require sshpass for password authentication, else need ssh key setup

## 3. rsync binary + rsync protocol (rsync daemon)

pros: easier setup for password authentication (password file)

cons: less security (but use in LAN only, should be acceptable)

# winner: 3. rsync binary + rsync protocol (rsync daemon)

# rsync usage

copy local folder to remote NAS.

the source path will look like (will contain CJK characters):
```
/mnt/4tb/pool/anime/[LoliHouse] Acro Trip [01-12][WebRip 1080p HEVC-10bit AAC]
```

the dest and final structure should look like:
```
# first `anime` segment is rsync module (NAS shared folder)
# second `anime` segment is actul folder on the NAS shared folder
anime/anime/Acro Trip 頂尖惡路/[LoliHouse] Acro Trip [01-12][WebRip 1080p HEVC-10bit AAC]
```

more example:
source folders
```
/mnt/4tb/pool/anime/[DBD-Raws][超超超超超喜欢你的100个女朋友][01-12TV全集+特典映像][1080P][BDRip][HEVC-10bit][简繁日双语外挂][FLAC][MKV]
/mnt/4tb/pool/anime/[DBD-Raws][超超超超超喜欢你的100个女朋友 第二季][01-12TV全集][1080P][BDRip][HEVC-10bit][简繁外挂][FLAC][MKV]
```

final result (note they share same `超超超超超喜歡你的100個女朋友` folder)
```
anime/anime/超超超超超喜歡你的100個女朋友/[DBD-Raws][超超超超超喜欢你的100个女朋友][01-12TV全集+特典映像][1080P][BDRip][HEVC-10bit][简繁日双语外挂][FLAC][MKV]
anime/anime/超超超超超喜歡你的100個女朋友/[DBD-Raws][超超超超超喜欢你的100个女朋友 第二季][01-12TV全集][1080P][BDRip][HEVC-10bit][简繁外挂][FLAC][MKV]
```
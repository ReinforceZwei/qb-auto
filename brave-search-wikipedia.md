# Brave Search API + Wikipeida API

Sometime the TMDb might return no result when seaching extracted title from downloaded torrent name (e.g. language variant, abbreviation). Or TMDb do not have a TW title for that TV show.

In manual workflow, I search the extracted title on search engine (Google) and visit Wikipedia page (Chinese, TW), take the anime title from either title (if it looks clean) or infobox (wikipedia template: Infobox animanga).

## API usage

### Brave Search

~_I think Brave Search API is popular enough to have tons of well-mantained API library to use_~

Im wrong, there is no popular golang library available.

Follow this offical [SKILL.md](https://github.com/brave/brave-search-skills/blob/main/skills/web-search/SKILL.md) from Brave to build one (only web-search API is needed)

### Wikipedia

_No existing golang API wrapper that cover the action/query i planned to use._

Brave will return English search result if the query is in English (or roman), even with search language set to Chinese. So we need to use Wikipedia API to "switch language".

1. Get the `title` from Brave Search result

e.g.
```
https://zh.wikipedia.org/zh-tw/%E5%A4%A9%E4%B9%85%E9%B7%B9%E5%A4%AE%E7%B3%BB%E5%88%97
# Decoded
https://zh.wikipedia.org/zh-tw/天久鷹央系列

# Take the title
天久鷹央系列

https://en.wikipedia.org/wiki/A_Ninja_and_an_Assassin_Under_One_Roof
# Take the title
A_Ninja_and_an_Assassin_Under_One_Roof
```

2. Get language list (only required for non-Chinese search result)

```
https://en.wikipedia.org/w/api.php?action=query&prop=langlinks&titles=A%20Ninja%20and%20an%20Assassin%20Under%20One%20Roof&lllimit=500&llprop=url&format=json
```

Using Python code as example:
```py
url = "https://en.wikipedia.org/w/api.php"

querystring = {"action":"query","prop":"langlinks","titles":"A_Ninja_and_an_Assassin_Under_One_Roof","lllimit":"500","llprop":"url","format":"json"}

response = requests.get(url, params=querystring)
```

Should response JSON:
```json
{
  "batchcomplete": "",
  "query": {
    "normalized": [
      {
        "from": "A_Ninja_and_an_Assassin_Under_One_Roof",
        "to": "A Ninja and an Assassin Under One Roof"
      }
    ],
    "pages": {
      "76712735": {
        "pageid": 76712735,
        "ns": 0,
        "title": "A Ninja and an Assassin Under One Roof",
        "langlinks": [
          {
            "lang": "de",
            "url": "https://de.wikipedia.org/wiki/Ninja_to_Koroshiya_no_Futarigurashi",
            "*": "Ninja to Koroshiya no Futarigurashi"
          },
          {
            "lang": "es",
            "url": "https://es.wikipedia.org/wiki/Ninja_to_Koroshiya_no_Futarigurashi",
            "*": "Ninja to Koroshiya no Futarigurashi"
          },
          {
            "lang": "fr",
            "url": "https://fr.wikipedia.org/wiki/Ninja_to_Koroshiya_no_Futarigurashi",
            "*": "Ninja to Koroshiya no Futarigurashi"
          },
          {
            "lang": "id",
            "url": "https://id.wikipedia.org/wiki/Ninkoro:_Duo_Ninja_dan_Pembunuh_Tinggal_Seatap",
            "*": "Ninkoro: Duo Ninja dan Pembunuh Tinggal Seatap"
          },
          {
            "lang": "ja",
            "url": "https://ja.wikipedia.org/wiki/%E5%BF%8D%E8%80%85%E3%81%A8%E6%AE%BA%E3%81%97%E5%B1%8B%E3%81%AE%E3%81%B5%E3%81%9F%E3%82%8A%E3%81%90%E3%82%89%E3%81%97",
            "*": "忍者と殺し屋のふたりぐらし"
          },
          {
            "lang": "ko",
            "url": "https://ko.wikipedia.org/wiki/%EB%8B%8C%EC%9E%90%EC%99%80_%EC%95%94%EC%82%B4%EC%9E%90%EC%9D%98_%EB%8F%99%EA%B1%B0",
            "*": "닌자와 암살자의 동거"
          },
          {
            "lang": "pl",
            "url": "https://pl.wikipedia.org/wiki/A_Ninja_and_an_Assassin_Under_One_Roof",
            "*": "A Ninja and an Assassin Under One Roof"
          },
          {
            "lang": "th",
            "url": "https://th.wikipedia.org/wiki/%E0%B8%AB%E0%B9%89%E0%B8%AD%E0%B8%87%E0%B8%9E%E0%B8%B1%E0%B8%81%E0%B8%9B%E0%B9%88%E0%B8%A7%E0%B8%99%E0%B8%82%E0%B8%AD%E0%B8%87%E0%B8%AA%E0%B8%AD%E0%B8%87%E0%B8%AA%E0%B8%B2%E0%B8%A7%E0%B8%99%E0%B8%B4%E0%B8%99%E0%B8%88%E0%B8%B2%E0%B8%81%E0%B8%B1%E0%B8%9A%E0%B8%99%E0%B8%B1%E0%B8%81%E0%B8%86%E0%B9%88%E0%B8%B2",
            "*": "ห้องพักป่วนของสองสาวนินจากับนักฆ่า"
          },
          {
            "lang": "uk",
            "url": "https://uk.wikipedia.org/wiki/A_Ninja_and_an_Assassin_Under_One_Roof",
            "*": "A Ninja and an Assassin Under One Roof"
          },
          {
            "lang": "zh",
            "url": "https://zh.wikipedia.org/wiki/%E5%BF%8D%E8%80%85%E8%88%87%E6%AE%BA%E6%89%8B%E7%9A%84%E5%90%8C%E4%BD%8F%E6%97%A5%E5%B8%B8",
            "*": "忍者與殺手的同住日常"
          }
        ]
      }
    }
  }
}
```

3. Get page content

Get page content in wiki text format so that it includes the infobox template which contain summarized information about the anime.

Python code example:
```py
# Note that language prefix in domain name, need to match title language
url = "https://zh.wikipedia.org/w/api.php"

querystring = {"action":"query","prop":"revisions","titles":"忍者與殺手的同住日常","rvprop":"content","format":"json","rvslots":"main"}

response = requests.get(url, headers=headers, params=querystring)
```

Should response: (content is truncated in this example)
```json
{
  "batchcomplete": "",
  "query": {
    "pages": {
      "9039014": {
        "pageid": 9039014,
        "ns": 0,
        "title": "忍者與殺手的同住日常",
        "revisions": [
          {
            "slots": {
              "main": {
                "contentmodel": "wikitext",
                "contentformat": "text/x-wiki",
                "*": "{{不是|NINJA SLAYER忍者殺手}}\n{{未完結}}\n{{Infobox animanga/Headerofja\n| 標題 = 忍者與殺手的同住日常\n| 日文名稱 = 忍者と殺し屋のふたりぐらし\n| 羅馬字 = Ninja to Koroshiya no Futarigurashi\n| image = File:Poster_of_Ninkoro.jpeg\n| caption = 动画主视觉图\n}}\n{{Infobox animanga/name\n| 正式譯名 = 忍者與殺手的同住日常（台灣角川）<br />忍者與殺手的兩人生活（回歸線娛樂）\n}}\n{{Infobox animanga/Manga\n| 作者 = {{tsl|ja|ハンバーガー (漫画家)|漢堡 (漫畫家)|漢堡}}\n| 作畫 = \n| 出版社 = 日本：[[KADOKAWA]]<br />台灣：[[台灣角川]]\n| 其他出版社 = \n| 連載雜誌 = {{tsl|ja|コミック電撃だいおうじ|漫畫電擊大王}}\n| 網路 = \n| 叢書 = 電撃漫畫NEXT\n| 開始 = Vol.90\n| 結束 = \n| 開始日 = 2021年2月26日\n| 結束日 = \n| 冊數 = 6卷（{{as of|2025|12|27}}）\n| 話數 = \n}}\n{{Infobox animanga/TVAnime\n| 標題 = 忍者與殺手的兩人生活\n| 原作 = 漢堡\n| 導演 = [[宮本幸裕]]\n| 系列構成 = [[SHAFT|東富耶子]]\n| 編劇 = 西部真帆 \n| 人物設定 = 潮月一也\n| 音樂 = 葛西龍之介\n| 音樂製作 = Heart Company\n| 動畫製作 = [[SHAFT]]\n| 製作 = 忍殺製作委員會\n| 代理發行 = 日本：[[KADOKAWA]]\n| 其他代理發行 = 中文授權：[[回歸線娛樂]]\n| 播放電視台 = [[AT-X]]等\n| 其他電視台 = \n| 播放開始 = [[2025年日本動畫列表|2025年]]4月10日\n| 播放結束 = 6月26日\n| 網絡 = 台港澳：[[巴哈姆特動畫瘋]]\n| 話數 = 全12话\n| 版權 = {{lang|ja|ハンバーガー / KADOKAWA / にんころ製作委員会}}\n}}\n{{Infobox animanga/Footerofja}}\n\n{{日本作品|忍者與殺手的同住日常|忍者と殺し屋のふたりぐらし}}，简称「忍殺」（{{lang|ja|にんころ}}），是{{tsl|ja|ハンバーガー (漫画家)|漢堡 (漫畫家)|漢堡}}創作的[[日本漫畫]]作品，於Vol.90起在雜誌《{{tsl|ja|コミック電撃だいおうじ|漫畫電擊大王}}》上連載<ref>{{Cite web|url=https://natalie.mu/comic/news/417956|script-title=ja:「ハンバーガーちゃん絵日記」のハンバーガー、女忍者マンガをだいおうじで開始|website=[[Comic Natalie]]|date=2021-02-26|accessdate=2025-03-16|language=ja|archive-date=2024-04-23|archive-url=https://web.archive.org/web/20240423090534/https://natalie.mu/comic/news/417956|dead-url=no}}</ref>。該作品"
              }
            }
          }
        ]
      }
    }
  }
}
```

## Automated workflow

Use this workflow when TMDb workflow return no result.

1. Search extracted title using brave search API

Prepend `wikipedia` in the query to get a higher chance search result will contain wikipedia link.

e.g. extracted title is `Ninja to Koroshiya no Futarigurashi`, search query will be `wikipeida Ninja to Koroshiya no Futarigurashi`

2. Find wikipedia URL in search result
3. Extract wikipedia title from URL
4. Get titles using wikipedia langlist API
5. Get page content in Chinese
6. Use LLM to extract: Chinese title, original title, official translation (if any)
7. Search TMDb again with original title (follow TMDb flow, confirm with LLM)
8a. If have result and have TW title -> return tmdb ID and anime title
8b. If have result but no TW title -> return tmdb ID and Chinese anime title from wikipedia
8c. If no result -> error
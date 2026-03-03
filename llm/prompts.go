package llm

// promptExtractTitle is the system prompt used to extract a bare anime title
// from a downloaded torrent folder name.
//
// Folder names are often noisy: they contain release group tags, resolution,
// codec, language, and episode information. The LLM should strip all of that
// and return only the core title (ideally the original Japanese or any single
// well-known title string that can be used to search TMDb).
//
// Response format: JSON object {"title": "<extracted title>"}
const promptExtractTitle = `You are a helpful assistant that extracts anime titles from torrent folder names.

Given a torrent folder name, extract only the core anime title. Remove:
- Release group names (e.g. [LoliHouse], [DBD-Raws], [BDrip])
- Episode or season numbers (e.g. [01-12], S01, 第二季)
- Video/audio format info (e.g. 1080P, HEVC, AAC, BDRip, WebRip)
- Subtitle language info (e.g. 简繁日双语, BIG5, FLAC)
- Any other metadata in square or round brackets

Return ONLY a JSON object in this exact format (no markdown, no explanation):
{"title": "<extracted title>"}`

// promptPickBestAnimeListMatch is the system prompt used to select the best
// anime list record for a given resolved Traditional Chinese anime title.
//
// The user message will contain the resolved title and a numbered list of anime
// list candidates (index and name only).
//
// Response format: JSON object {"index": <0-based index, or -1 if no match>}
const promptPickBestAnimeListMatch = `You are a helpful assistant that identifies the correct anime record from a watch list.

You will be given:
1. A resolved anime title in Traditional Chinese
2. A numbered list of anime watch list records (index and name)

Pick the record whose name best matches the given title.
The names in the list may use Traditional Chinese, Simplified Chinese, Japanese, or other representations of the same anime.
If none of the records match the title, return -1.

Return ONLY a JSON object in this exact format (no markdown, no explanation):
{"index": <0-based index of best match, or -1 if none match>}`

// promptPickBestMatch is the system prompt used to select the best TMDb TV show
// result for a given anime folder name.
//
// The user message will contain the original folder name, the extracted title,
// and a numbered list of TMDb candidates (index, name, original name, overview).
//
// Response format: JSON object {"index": <0-based index, or -1 if no match>}
const promptPickBestMatch = `You are a helpful assistant that identifies the correct anime from a list of search results.

You will be given:
1. The original torrent folder name
2. The extracted anime title
3. A numbered list of TMDb TV show search results

Pick the result that best matches the anime described by the folder name.
Consider the original name, localized name, and overview when deciding.
If none of the results match, return -1.

Return ONLY a JSON object in this exact format (no markdown, no explanation):
{"index": <0-based index of best match, or -1 if none match>}`

import { account, api, fault, mediaURL, publicAPI, requireAccount } from "./client.js";
import {
  albumEntry, artistEntry, commentEntry, episodeEntry, idOf, items, offsetOf, pageSize,
  playlistEntry, showEntry, songDetails, trackEntry, trackIdentity,
} from "./catalog.js";

export { account };
export { signIn, signOut } from "./client.js";

const searchTypes = {
  tracks: { type: 1, field: "songs", count: "songCount", convert: (value) => trackEntry(value) },
  albums: { type: 10, field: "albums", count: "albumCount", convert: albumEntry },
  artists: { type: 100, field: "artists", count: "artistCount", convert: artistEntry },
  playlists: { type: 1000, field: "playlists", count: "playlistCount", convert: (value) => playlistEntry(value) },
  shows: { type: 1009, field: "djRadios", count: "djRadiosCount", convert: showEntry },
};

export async function search({ query, kind = "tracks", offset }) {
  const text = query.trim();
  if (!text) return {};
  if (!Object.hasOwn(searchTypes, kind)) throw fault("unsupported", "网易云音乐不支持此搜索类型。");
  const { type, field, count, convert } = searchTypes[kind];
  offset = offsetOf(offset);
  const body = await api("/api/cloudsearch/pc", { s: text, type, offset, limit: pageSize, total: true });
  const rows = items(body.result?.[field]);
  const results = kind === "tracks" ? await detailedTracks(rows) : rows.map(convert);
  return { [kind]: results, more: offset + rows.length < Number(body.result?.[count] || 0) };
}

async function detailedTracks(songs) {
  try {
    return await songDetails(songs.map((song) => song.id), songs, { refresh: true });
  } catch (error) {
    if (error.code === "signIn") throw error;
    return songs.map((song) => trackEntry(song));
  }
}

export async function suggest({ query }) {
  if (!query.trim()) return [];
  const body = await api("/api/search/suggest/keyword", { s: query.trim(), limit: 8 });
  return [...new Set(items(body.result?.allMatch).map((item) => item.keyword).filter(Boolean))].slice(0, 8);
}

async function charts() {
  return items((await publicAPI("/api/toplist/detail")).list).map((value) => playlistEntry(value));
}

async function discoveryPlaylists() {
  const body = await publicAPI("/api/personalized/playlist", { limit: 24 });
  if (!Array.isArray(body.result)) throw new Error("网易云音乐未返回推荐歌单。");
  return body.result.map((value) => playlistEntry(value));
}

async function discoverySection() {
  const title = "推荐歌单";
  try {
    return { title, playlists: await discoveryPlaylists(), empty: "暂无推荐歌单。" };
  } catch (error) {
    if (error.code === "signIn") throw error;
    return { title, morePage: "discovery", empty: "推荐歌单暂时无法加载，可打开详情页重试。" };
  }
}

const myMusic = {
  title: "我的音乐",
  links: [
    { title: "最近一周", subtitle: "听歌排行", page: "record:week" },
    { title: "所有时间", subtitle: "听歌排行", page: "record:all" },
    { title: "音乐云盘", page: "cloud:0" },
  ],
};

const forYou = {
  title: "为你推荐",
  links: [
    { title: "每日推荐", page: "daily" },
    { title: "私人雷达", page: "radar" },
    { title: "个性化歌单", page: "recommendations" },
  ],
};

export function navigation() {
  return [
    { title: "发现音乐", builtin: "home" },
    { builtin: "liked" },
    { builtin: "playlists" },
    { builtin: "savedAlbums" },
    { builtin: "recent" },
    { builtin: "shows" },
  ];
}

export async function home() {
  return {
    title: "发现音乐",
    sections: [
      myMusic,
      forYou,
      { title: "私人漫游", stations: stationList() },
      await discoverySection(),
      { links: [{ title: "排行榜", page: "charts" }] },
    ],
  };
}

async function cloudPage(offset) {
  const response = await api("/api/v1/cloud/get", { offset, limit: pageSize });
  if (!Array.isArray(response.data) || (response.hasMore && response.data.length === 0)) {
    throw new Error("网易云音乐未返回有效的云盘列表。");
  }
  const songs = response.data.map(({ simpleSong: song, songId, songName, artist, album }) => ({
    ...song,
    id: songId ?? song?.id,
    name: songName || song?.name,
    ar: song?.ar || song?.artists || (artist ? [{ name: artist }] : []),
    al: song?.al || song?.album || { name: album },
  }));
  return { title: "音乐云盘", sections: [{
    tracks: songs.map((song) => trackEntry(song)),
    empty: "云盘暂无歌曲。",
    links: response.hasMore ? [{ title: "下一页", page: "cloud:" + (offset + songs.length) }] : [],
  }] };
}

async function recordPage(week) {
  const { id } = await requireAccount();
  const body = await api("/api/v1/play/record", { uid: id, type: week ? 1 : 0 });
  const entries = week ? body.weekData : body.allData;
  if (!Array.isArray(entries)) throw new Error("网易云音乐未返回有效的听歌排行。");
  return {
    title: week ? "最近一周 · 听歌排行" : "所有时间 · 听歌排行",
    sections: [{ tracks: entries.filter((entry) => entry.song?.id).map((entry) => trackEntry(entry.song)), empty: "暂无听歌记录。" }],
  };
}

export async function browse({ page }) {
  if (page === "my-music" || page === "for-you") {
    const { title, links } = page === "my-music" ? myMusic : forYou;
    return { title, sections: [{ links }] };
  }
  if (page === "charts") return { title: "排行榜", sections: [{ playlists: await charts() }] };
  if (page === "discovery") return { title: "推荐歌单", sections: [{ playlists: await discoveryPlaylists(), empty: "暂无推荐歌单。" }] };
  if (page === "record:week" || page === "record:all") return recordPage(page === "record:week");
  if (page === "radar") {
    const user = await requireAccount();
    const radar = await loadPlaylist("3136952023", user);
    return { title: "私人雷达", sections: [{ tracks: radar.tracks, empty: "网易云音乐暂未返回私人雷达歌曲。" }] };
  }
  if (page === "daily") {
    const tracks = await api("/api/v3/discovery/recommend/songs");
    const songs = items(tracks.data?.dailySongs || tracks.recommend);
    return { title: "每日推荐", sections: [{ tracks: await detailedTracks(songs), empty: "暂无每日推荐歌曲。" }] };
  }
  if (page === "recommendations") {
    const lists = await api("/api/v1/discovery/recommend/resource");
    return { title: "个性化歌单", sections: [{ playlists: items(lists.recommend).map((value) => playlistEntry(value)), empty: "暂无个性化歌单。" }] };
  }
  const cloud = /^cloud:(\d+)$/.exec(page);
  if (cloud) return cloudPage(offsetOf(Number(cloud[1])));
  throw fault("unavailable", "网易云音乐页面不存在。");
}

async function rawPlaylist(id, n = pageSize, request = api) {
  const response = await request("/api/v6/playlist/detail", { id: idOf(id), n, s: 0 });
  if (!response.playlist) throw fault("unavailable", "网易云音乐歌单不存在或无权访问。");
  return response.playlist;
}

export async function playlist({ id }) {
  return loadPlaylist(id, await account());
}

async function loadPlaylist(id, user) {
  const request = user ? api : publicAPI;
  const result = await rawPlaylist(id, pageSize, request);
  const songs = items(result.tracks);
  const ids = items(result.trackIds).map((track) => track.id);
  return {
    ...playlistEntry(result, user?.id),
    tracks: ids.length ? await songDetails(ids, songs, { request }) : songs.map((song) => trackEntry(song)),
  };
}

async function listAll(path, fields, key, more, recover) {
  const limit = 100;
  const rows = new Map();
  let stalled = 0;
  for (let offset = 0; offset < 10000;) {
    const body = await api(path, { ...fields, offset, limit, total: true });
    const page = body[key];
    if (!Array.isArray(page)) throw new Error("网易云音乐未返回有效的列表。");
    const previous = rows.size;
    for (const item of page) rows.set(String(item.id), item);
    if (!body[more]) return [...rows.values()];
    stalled = rows.size === previous ? stalled + 1 : 0;
    if (!page.length || stalled === 2) {
      if (recover) return recover(rows.size);
      break;
    }
    offset += page.length;
  }
  throw new Error("网易云音乐未返回完整的列表，请稍后重试。");
}

export async function playlists() {
  const user = await requireAccount();
  const rows = await listAll("/api/user/playlist", { uid: user.id, includeVideo: false }, "playlist", "more");
  return rows.map((value) => playlistEntry(value, user.id));
}

export async function album({ id }) {
  const body = await api("/api/v1/album/" + idOf(id));
  if (!body.album) throw fault("unavailable", "网易云音乐专辑不存在。");
  return { album: albumEntry(body.album), tracks: items(body.songs).map((song) => trackEntry(song, song.privilege, body.album)) };
}

export async function artist({ id }) {
  id = idOf(id);
  const [body, records] = await Promise.all([
    api("/api/v1/artist/" + id),
    listAll("/api/artist/albums/" + id, {}, "hotAlbums", "more"),
  ]);
  if (!body.artist) throw fault("unavailable", "网易云音乐歌手不存在。");
  return {
    artist: artistEntry(body.artist),
    tracks: items(body.hotSongs).map((song) => trackEntry(song)),
    albums: records.map(albumEntry),
  };
}

let likedCache;

async function likedIDs() {
  const { id } = await requireAccount();
  const now = Date.now();
  if (likedCache?.id === id && now - likedCache.time < 15000) return likedCache.result;
  const result = api("/api/song/like/get", { uid: id }).then((body) => items(body.ids).map(idOf));
  likedCache = { id, time: now, result };
  try {
    return await result;
  } catch (error) {
    if (likedCache?.result === result) likedCache = undefined;
    throw error;
  }
}

export async function liked({ offset }) {
  offset = offsetOf(offset);
  const ids = await likedIDs();
  return { tracks: await songDetails(ids.slice(offset, offset + pageSize)), more: offset + pageSize < ids.length };
}

export async function isLiked({ ids }) {
  const liked = new Set(await likedIDs());
  return Object.fromEntries(ids.map((id) => [id, liked.has(id)]));
}

export async function like({ id, on }) {
  await api("/api/radio/like", { trackId: idOf(id), like: !!on, alg: "itembased", time: 3 }, true);
  likedCache = undefined;
}

export async function recent() {
  const body = await api("/api/play-record/song/list", { limit: 100 });
  return items(body.data?.list).filter((entry) => entry.data?.id).map((entry) => trackEntry(entry.data));
}

export async function createPlaylist({ name }) {
  const { id } = await requireAccount();
  const body = await api("/api/playlist/create", { name: name.trim(), privacy: 0, type: "NORMAL" }, true);
  if (!body.playlist?.id) throw new Error("网易云音乐未返回新建的歌单。");
  return { ...playlistEntry(body.playlist, id), mine: true };
}

export async function renamePlaylist({ id, name }) {
  await api("/api/playlist/update/name", { id: idOf(id), name: name.trim() }, true);
}

export async function deletePlaylist({ id }) {
  await api("/api/playlist/remove", { ids: JSON.stringify([idOf(id)]) }, true);
}

async function editPlaylistTracks(id, ids, operation) {
  const response = await api("/api/playlist/manipulate/tracks",
    { pid: idOf(id), trackIds: JSON.stringify(ids.map(idOf)), op: operation, imme: true }, true);
  if (response.body?.code && response.body.code !== 200) {
    throw new Error(response.body.message || "网易云音乐暂时无法修改歌单。");
  }
}

export async function addToPlaylist({ id, trackIds }) {
  if (trackIds.length) await editPlaylistTracks(id, trackIds, "add");
}

export async function removeFromPlaylist({ id, trackIds }) {
  if (trackIds.length) await editPlaylistTracks(id, trackIds, "del");
}

export async function movePlaylistTrack({ id, from, to }) {
  const p = await rawPlaylist(id, 0);
  const ids = items(p.trackIds).map((item) => idOf(item.id));
  if (![from, to].every((i) => Number.isSafeInteger(i) && i >= 0 && i < ids.length)) throw new Error("歌曲位置已变化，请刷新歌单。");
  if (from === to) return;
  ids.splice(to, 0, ...ids.splice(from, 1));
  await editPlaylistTracks(id, ids, "update");
}

export async function savePlaylist({ id, on }) {
  await api("/api/playlist/" + (on ? "subscribe" : "unsubscribe"), { id: idOf(id) }, true);
}

export async function followArtist({ id, on }) {
  id = idOf(id);
  await api("/api/artist/" + (on ? "sub" : "unsub"), { artistId: id, artistIds: JSON.stringify([id]) }, true);
}

export async function saveAlbum({ id, on }) {
  await api("/api/album/" + (on ? "sub" : "unsub"), { id: idOf(id) }, true);
}

export async function savedAlbums({ offset }) {
  const body = await api("/api/album/sublist", { offset: offsetOf(offset), limit: pageSize, total: true });
  return { albums: items(body.data).map((item) => ({ ...albumEntry(item), saved: true })), more: !!body.hasMore };
}

export async function shows() {
  const path = "/api/djradio/get/subed";
  const rows = await listAll(path, {}, "djRadios", "hasMore", async (loaded) => {
    for (let limit = 200; limit <= 2000; limit = Math.min(limit * 2, 2000)) {
      if (limit <= loaded && limit < 2000) continue;
      const result = await api(path, { offset: 0, limit, total: true });
      if (Array.isArray(result.djRadios) && result.hasMore === false) {
        const unique = new Map(result.djRadios.map((item) => [String(item.id), item]));
        if (unique.size >= loaded) return [...unique.values()];
      }
      if (limit === 2000) break;
    }
    throw new Error("网易云音乐未返回完整的播客订阅列表，请稍后重试。");
  });
  return rows.map((item) => ({ ...showEntry(item), subscribed: true }));
}

export async function show({ id, offset }) {
  id = idOf(id);
  offset = offsetOf(offset);
  const [detail, list] = await Promise.all([
    api("/api/djradio/v2/get", { id }),
    api("/api/dj/program/byradio", { radioId: id, offset, limit: pageSize, asc: false }),
  ]);
  if (!detail.data) throw fault("unavailable", "网易云音乐播客不存在。");
  const info = showEntry(detail.data);
  const episodes = items(list.programs).map((entry) => episodeEntry(entry, info));
  return { show: info, episodes, more: !!list.more };
}

export async function subscribeShow({ id, on }) {
  await api("/api/djradio/" + (on ? "sub" : "unsub"), { id: idOf(id) }, true);
}

function stationList() {
  return [
    { id: "DEFAULT", name: "私人漫游", description: "根据你的听歌偏好持续推荐" },
    { id: "FAMILIAR", name: "熟悉的歌", description: "听听你熟悉的歌曲" },
    { id: "EXPLORE", name: "新发现", description: "探索还没听过的音乐" },
  ];
}

export function stations() {
  return stationList();
}

export async function stationNext({ id }) {
  const mode = id === "personal" ? "DEFAULT" : id;
  if (!stationList().some((station) => station.id === mode)) throw fault("unavailable", "网易云音乐电台不存在。");
  const body = await api("/api/v1/radio/get", { mode, limit: 3, timestamp: Date.now() });
  const songs = items(body.data);
  if (!songs.length) {
    const settings = await api("/api/user/setting").catch((error) => {
      if (error.code === "signIn") throw error;
      return null;
    });
    if (Number(settings?.setting?.personalServiceSetting) === 2) {
      throw fault("unavailable", "请在网易云音乐 App 的「设置 → 消息和隐私」开启「个性化服务」后重试。");
    }
    throw fault("unavailable", "网易云音乐暂未返回漫游歌曲，请稍后重试。");
  }
  return detailedTracks(songs);
}

export async function comments({ trackId, offset }) {
  const { thread, resourceId } = trackIdentity(trackId);
  offset = offsetOf(offset);
  const [body, hot] = await Promise.all([
    api("/api/v1/resource/comments/" + thread, { offset, limit: pageSize }),
    offset === 0 ? api("/api/v1/resource/hotcomments/" + thread, { rid: resourceId, limit: 20, offset: 0, beforeTime: 0 })
      .catch((error) => {
        if (error.code === "signIn") throw error;
        return null;
      }) : null,
  ]);
  const hotComments = offset === 0 ? items(hot?.hotComments ?? body.hotComments)
    .slice().sort((left, right) => Number(right.likedCount || 0) - Number(left.likedCount || 0)).slice(0, 20) : [];
  return {
    comments: items(body.comments).map(commentEntry), hotComments: hotComments.map(commentEntry),
    total: Number(body.total || 0), more: !!body.more,
  };
}

export async function likeComment({ trackId, commentId, on }) {
  await api("/api/v1/comment/" + (on ? "like" : "unlike"),
    { threadId: trackIdentity(trackId).thread, commentId: idOf(commentId) }, true);
}

export async function postComment({ trackId, text }) {
  const body = await api("/api/resource/comments/add", { threadId: trackIdentity(trackId).thread, content: text.trim() }, true);
  if (!body.comment?.commentId) throw new Error("网易云音乐未确认评论已发布。");
  return commentEntry(body.comment);
}

function timedLyrics(value) {
  const lines = [];
  for (const row of String(value || "").split(/\r?\n/)) {
    const line = /^\[(\d+),(\d+)\](.+)$/.exec(row);
    if (!line) continue;
    const words = [];
    for (const word of line[3].matchAll(/\((\d+),(\d+),\d+\)(.*?)(?=\(\d+,\d+,\d+\)|$)/g)) {
      const start = Number(word[1]);
      if (word[3]) words.push({ start, end: start + Number(word[2]), text: word[3] });
    }
    if (words.length) lines.push({ start: Number(line[1]), end: Number(line[1]) + Number(line[2]), words });
  }
  return lines;
}

export async function lyrics({ id }) {
  const { songId, thread } = trackIdentity(id);
  if (thread.startsWith("A_DJ_1_")) return {};
  let desktop, desktopError;
  try {
    desktop = await api("/api/song/lyric/v1", { id: songId, lv: 0, tv: 0, yv: 0, ytv: 0 });
  } catch (error) {
    if (error.code === "signIn") throw error;
    desktopError = error;
  }
  const lrc = desktop?.lrc?.lyric || "";
  const translation = desktop?.tlyric?.lyric || "";
  const lines = timedLyrics(desktop?.yrc?.lyric);
  if (lines.length) {
    const wordTranslation = desktop?.ytlrc?.lyric;
    return { lrc, translation: /^\[\d+:/m.test(wordTranslation || "") ? wordTranslation : translation, lines };
  }
  if (lrc.trim()) return { lrc, translation };
  const web = await publicAPI("/api/song/lyric", { id: songId, lv: 1, tv: -1 });
  if (!web.lrc?.lyric?.trim() && desktopError) throw desktopError;
  return { lrc: web.lrc?.lyric || "", translation: web.tlyric?.lyric || translation };
}

export async function stream({ id, quality = "high" }) {
  const { songId } = trackIdentity(id);
  if (!songId) throw fault("unavailable", "该播客节目没有可播放的音频。");
  const level = { standard: "standard", high: "exhigh", lossless: "lossless" }[quality] || "exhigh";
  const body = await api("/api/song/enhance/player/url/v1",
    { ids: "[" + songId + "]", level, encodeType: level === "lossless" ? "flac" : "mp3", timestamp: Date.now() });
  const audio = items(body.data).find((item) => String(item.id) === songId);
  const url = mediaURL(audio?.url);
  if (!url || (audio.code != null && Number(audio.code) !== 200) || audio.freeTrialInfo != null) {
    throw fault("unavailable", "网易云音乐当前账号无法完整播放这首曲目。");
  }
  const format = String(audio.type || "").toLowerCase();
  const type = format === "mp4" ? "m4a" : format;
  if (!["mp3", "flac", "aac", "m4a", "wav", "ogg"].includes(type)) {
    throw fault("unavailable", "网易云音乐返回了无法播放的音频格式。");
  }
  const actualQuality = { standard: "standard", higher: "high", exhigh: "high", lossless: "lossless", hires: "lossless" };
  return {
    url, type, quality: actualQuality[audio.level] || "", bitRate: Number(audio.br || 0), size: Number(audio.size || 0),
    gain: Number(audio.gain || 0), peak: Number(audio.peak || 0),
  };
}

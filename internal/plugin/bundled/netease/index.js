import { api, fault, mediaURL, requireAccount } from "./client.js";
import {
  albumEntry, artistEntry, commentEntry, episodeEntry, idOf, items, offsetOf, pageSize,
  playlistEntry, showEntry, songDetails, trackEntry, trackIdentity,
} from "./catalog.js";

export { account, signIn, signOut } from "./client.js";

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
  const results = items(body.result?.[field]).map(convert);
  return { [kind]: results, more: offset + results.length < Number(body.result?.[count] || 0) };
}

export async function suggest({ query }) {
  if (!query.trim()) return [];
  const body = await api("/api/search/suggest/keyword", { s: query.trim(), limit: 8 });
  return [...new Set(items(body.result?.allMatch).map((item) => item.keyword).filter(Boolean))].slice(0, 8);
}

async function charts() {
  return items((await api("/api/toplist")).list).map((value) => playlistEntry(value));
}

export async function home() {
  const [songs, lists] = await Promise.all([
    api("/api/v3/discovery/recommend/songs"),
    charts(),
  ]);
  return {
    title: "网易云音乐",
    sections: [
      { title: "每日推荐", links: [{ title: "今日推荐", page: "daily" }, { title: "音乐云盘", page: "cloud:0" }] },
      { title: "排行榜", playlists: lists.slice(0, 6), morePage: "charts" },
      { title: "今日歌曲", tracks: items(songs.data?.dailySongs || songs.recommend).slice(0, 12).map((song) => trackEntry(song)) },
    ],
  };
}

async function cloudPage(offset) {
  const response = await api("/api/v1/cloud/get", { offset, limit: pageSize });
  const songs = items(response.data).map((value) => value.simpleSong || {
    id: value.songId, name: value.songName, ar: [{ name: value.artist }], al: { name: value.album },
  });
  return { title: "音乐云盘", sections: [{
    tracks: songs.map((song) => trackEntry(song)),
    links: response.hasMore ? [{ title: "下一页", page: "cloud:" + (offset + songs.length) }] : [],
  }] };
}

export async function browse({ page }) {
  if (page === "charts") return { title: "排行榜", sections: [{ playlists: await charts() }] };
  if (page === "daily") {
    const [tracks, lists] = await Promise.all([
      api("/api/v3/discovery/recommend/songs"),
      api("/api/v1/discovery/recommend/resource"),
    ]);
    return { title: "每日推荐", sections: [
      { title: "今日歌曲", tracks: items(tracks.data?.dailySongs || tracks.recommend).map((song) => trackEntry(song)) },
      { title: "推荐歌单", playlists: items(lists.recommend).map((value) => playlistEntry(value)) },
    ] };
  }
  const cloud = /^cloud:(\d+)$/.exec(page);
  if (cloud) return cloudPage(offsetOf(Number(cloud[1])));
  throw fault("unavailable", "网易云音乐页面不存在。");
}

async function rawPlaylist(id, n = pageSize) {
  const response = await api("/api/v6/playlist/detail", { id: idOf(id), n, s: 0 });
  if (!response.playlist) throw fault("unavailable", "网易云音乐歌单不存在或无权访问。");
  return response.playlist;
}

export async function playlist({ id }) {
  const user = await requireAccount();
  const result = await rawPlaylist(id);
  const songs = items(result.tracks);
  const ids = items(result.trackIds).map((track) => track.id);
  return {
    ...playlistEntry(result, user.id),
    tracks: ids.length ? await songDetails(ids, songs) : songs.map((song) => trackEntry(song)),
  };
}

async function listAll(path, fields, key, more) {
  const limit = 100;
  const rows = new Map();
  for (let offset = 0; ; offset += limit) {
    const body = await api(path, { ...fields, offset, limit, total: true });
    const page = items(body[key]);
    let added = 0;
    for (const item of page) {
      if (rows.has(item.id)) continue;
      rows.set(item.id, item);
      added++;
    }
    if (!body[more] || page.length < limit || added === 0) return [...rows.values()];
  }
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
  const rows = await listAll("/api/djradio/get/subed", {}, "djRadios", "hasMore");
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
  const episodes = items(list.programs).filter((entry) => entry.mainSong?.id).map((entry) => episodeEntry(entry, info));
  return { show: info, episodes, more: !!list.more };
}

export async function subscribeShow({ id, on }) {
  await api("/api/djradio/" + (on ? "sub" : "unsub"), { id: idOf(id) }, true);
}

export async function stations() {
  await requireAccount();
  return [{ id: "personal", name: "私人 FM", description: "根据你的听歌偏好持续播放" }];
}

export async function stationNext({ id }) {
  if (id !== "personal") throw fault("unavailable", "网易云音乐电台不存在。");
  const body = await api("/api/v1/radio/get", { timestamp: Date.now() });
  return items(body.data).map((song) => trackEntry(song));
}

export async function comments({ trackId, offset }) {
  const { thread } = trackIdentity(trackId);
  const body = await api("/api/v1/resource/comments/" + thread, { offset: offsetOf(offset), limit: pageSize });
  return { comments: items(body.comments).map(commentEntry), total: Number(body.total || 0), more: !!body.more };
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
  const { songId } = trackIdentity(id);
  if (id.startsWith("episode:")) return {};
  const body = await api("/api/song/lyric/v1", { id: songId, lv: 0, tv: 0, yv: 0 });
  return { lrc: body.lrc?.lyric || "", translation: body.tlyric?.lyric || "", lines: timedLyrics(body.yrc?.lyric) };
}

export async function stream({ id, quality = "high" }) {
  const { songId } = trackIdentity(id);
  const level = { standard: "standard", high: "exhigh", lossless: "lossless" }[quality] || "exhigh";
  const body = await api("/api/song/enhance/player/url/v1",
    { ids: "[" + songId + "]", level, encodeType: "flac", timestamp: Date.now() });
  const audio = items(body.data).find((item) => String(item.id) === songId);
  const url = mediaURL(audio?.url);
  if (!url || audio.code !== 200 || audio.freeTrialInfo) {
    throw fault("unavailable", "网易云音乐当前账号无法完整播放这首曲目。");
  }
  const type = String(audio.type || "").toLowerCase();
  if (!["mp3", "flac", "aac", "m4a", "wav", "ogg"].includes(type)) {
    throw fault("unavailable", "网易云音乐返回了无法播放的音频格式。");
  }
  return { url, type, bitRate: Number(audio.br || 0), size: Number(audio.size || 0) };
}

// Heart-mode request and continuation behavior follows these AGPL sources:
// sogud/mygo-music-netease, index.js at 82696baa3f773d7661a568a430bde8d68f8d0f43 (AGPL-3.0-only).
// AkinoKaede/mygo-music-netease-cloud-music, src/radio.ts at
// b9bf374cce816e87ed3ca58e293ff4001842c1cb (Copyright 2026 Kaede Akino, AGPL-3.0-or-later).
// SPDX-License-Identifier: AGPL-3.0-only

import { fault, mediaURL, requireSession } from "./client.js";
import { albumEntry, idOf, items, pageSize, songDetails } from "./catalog.js";

let heartState;
let pendingHeart;

function unavailable(message = "网易云音乐暂未返回新的心动推荐，请稍后重试。") {
  return fault("unavailable", message);
}

async function heartSource(context) {
  const { user, request } = context;
  const seen = new Set();
  let favorite;
  for (let offset = 0; offset < 2000;) {
    const body = await request("/api/user/playlist", { uid: user.id, includeVideo: false, offset, limit: pageSize, total: true });
    if (!Array.isArray(body.playlist)) throw unavailable("网易云音乐未返回有效的喜欢歌单列表。");
    favorite = body.playlist.find((value) => value && Number(value.specialType) === 5 &&
      String(value.creator?.userId || value.userId) === user.id);
    if (favorite || !body.more) break;
    const previous = seen.size;
    for (const playlist of body.playlist) if (playlist?.id) seen.add(String(playlist.id));
    if (seen.size === previous) throw unavailable("网易云音乐未返回完整的喜欢歌单列表，请稍后重试。");
    offset += body.playlist.length;
  }
  if (!favorite) throw unavailable("未找到当前账号的喜欢歌单，暂时无法开启心动模式。");

  const playlistId = idOf(favorite.id);
  const body = await request("/api/v6/playlist/detail", { id: playlistId, n: pageSize, s: 0 });
  const detail = body.playlist;
  if (!detail || String(detail.id) !== playlistId || Number(detail.specialType) !== 5 ||
      String(detail.creator?.userId || detail.userId) !== user.id) {
    throw unavailable("无法确认当前账号的喜欢歌单，请稍后重试。");
  }
  const entries = detail.trackIds ?? detail.tracks;
  if (!Array.isArray(entries)) throw unavailable("网易云音乐未返回喜欢歌单的歌曲列表。");
  const ids = [...new Set(entries.filter((song) => song?.id).map((song) => idOf(song.id)))];
  if (!ids.length) throw unavailable("喜欢歌单为空，请先在网易云音乐喜欢几首歌曲。");
  const song = items(detail.tracks).find((song) => song?.id);
  return {
    playlistId, ids,
    imageUrl: albumEntry(song?.al || song?.album).imageUrl || mediaURL(detail.coverImgUrl || favorite.coverImgUrl),
  };
}

export async function heartPreview() {
  return heartSource(await requireSession());
}

async function loadHeart(context) {
  const { user, request, check } = context;
  let state = heartState?.key === context.key && heartState.generation === context.generation && heartState.uid === user.id
    ? heartState : null;
  let seed;
  if (!state) {
    const source = await heartSource(context);
    for (let offset = 0; offset < source.ids.length && !seed; offset += pageSize) {
      const tracks = await songDetails(source.ids.slice(offset, offset + pageSize), [], { request });
      seed = tracks.find((track) => !track.unavailable);
    }
    if (!seed) throw unavailable("喜欢歌单中暂无可播放的歌曲，暂时无法开启心动模式。");
    state = {
      key: context.key, generation: context.generation, uid: user.id,
      playlistId: source.playlistId, seed: seed.id, last: seed.id, seen: [seed.id],
    };
  }

  const seen = new Set(state.seen);
  let cursor = state.last;
  // A repeated batch may still carry a new cursor. Bound attempts if it stalls.
  for (let attempt = 0; attempt < 3; attempt++) {
    const body = await request("/api/playmode/intelligence/list", {
      songId: cursor, startMusicId: state.seed, playlistId: state.playlistId, type: "fromPlayOne", count: 1,
    });
    if (!Array.isArray(body.data)) throw unavailable("网易云音乐未返回有效的心动推荐列表。");
    if (!body.data.length) throw unavailable();
    const ids = new Set();
    let nextCursor = cursor;
    for (const row of body.data) {
      if (row == null || row.songInfo == null) continue;
      const id = idOf(row.songInfo.id);
      nextCursor = id;
      if (!seen.has(id)) ids.add(id);
    }
    const tracks = (await songDetails([...ids], [], { request })).filter((track) => !track.unavailable);
    check();
    for (const id of ids) seen.add(id);
    if (tracks.length) {
      heartState = { ...state, last: nextCursor, seen: [...seen].slice(-200) };
      return seed ? [seed, ...tracks] : tracks;
    }
    // Keep an established station's progress even when this batch has no new songs.
    if (!seed) heartState = { ...state, last: nextCursor, seen: [...seen].slice(-200) };
    if (nextCursor === cursor) break;
    cursor = nextCursor;
  }
  throw unavailable();
}

export async function heartNext() {
  const context = await requireSession();
  if (pendingHeart?.key === context.key && pendingHeart.generation === context.generation) return pendingHeart.promise;
  const pending = { key: context.key, generation: context.generation, promise: loadHeart(context) };
  pendingHeart = pending;
  try {
    return await pending.promise;
  } finally {
    if (pendingHeart === pending) pendingHeart = null;
  }
}

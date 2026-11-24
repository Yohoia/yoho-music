import { api, fault, mediaURL, origin } from "./client.js";

export const pageSize = 30;
export const items = (value) => Array.isArray(value) ? value : [];
export const offsetOf = (value) => Number.isSafeInteger(value) && value >= 0 ? value : 0;

export function idOf(value) {
  const id = String(value ?? "");
  if (!/^[1-9]\d{0,18}$/.test(id)) throw fault("unavailable", "网易云音乐内容标识无效。");
  return id;
}

export function trackIdentity(value) {
  const id = String(value);
  const episode = /^episode:([1-9]\d{0,18}):([1-9]\d{0,18})$/.exec(id);
  return episode ? { songId: episode[1], thread: "A_DJ_1_" + episode[2] }
    : { songId: idOf(id), thread: "R_SO_4_" + idOf(id) };
}

export function artistEntry(artist = {}) {
  return { id: String(artist.id || ""), name: artist.name || "", imageUrl: mediaURL(artist.picUrl || artist.img1v1Url), following: !!artist.followed };
}

export function albumEntry(album = {}) {
  const artists = items(album.artists);
  return {
    id: String(album.id || ""), name: album.name || "", imageUrl: mediaURL(album.picUrl || album.coverImgUrl),
    artists: (artists.length ? artists : album.artist?.id ? [album.artist] : []).map(artistEntry),
    saved: !!album.isSub,
  };
}

export function trackEntry(song, privilege = song.privilege, album = song.al || song.album) {
  const id = idOf(song.id);
  return {
    id, name: song.name || "歌曲 " + id, artists: items(song.ar || song.artists).map(artistEntry),
    album: albumEntry(album), duration: Number(song.dt || song.duration || 0),
    unavailable: Number(privilege?.st ?? song.st ?? 0) < 0,
    url: origin + "/#/song?id=" + id,
  };
}

export function playlistEntry(playlist, uid = "") {
  const mine = uid !== "" && String(playlist.creator?.userId || playlist.userId) === uid;
  return {
    id: idOf(playlist.id), name: playlist.name || "", imageUrl: mediaURL(playlist.coverImgUrl || playlist.picUrl),
    owner: playlist.creator?.nickname || "", description: playlist.description || "",
    trackCount: Number(playlist.trackCount || 0), mine, fixed: mine && playlist.specialType === 5, saved: !!playlist.subscribed,
  };
}

export function showEntry(show) {
  return {
    id: idOf(show.id), name: show.name || "", description: show.desc || "",
    imageUrl: mediaURL(show.picUrl), publisher: show.dj?.nickname || "",
    episodeCount: Number(show.programCount || 0), subscribed: !!(show.subed ?? show.subscribed),
  };
}

export function episodeEntry(program, show) {
  const song = program.mainSong;
  const track = trackEntry(song);
  const programId = idOf(program.id);
  return {
    ...track, id: "episode:" + track.id + ":" + programId, name: program.name || track.name, showId: show.id,
    album: { name: show.name, imageUrl: mediaURL(program.coverUrl) || show.imageUrl },
    description: program.description || "", url: origin + "/#/program?id=" + programId,
  };
}

export function commentEntry(comment) {
  return {
    id: idOf(comment.commentId), author: comment.user?.nickname || "网易云用户",
    avatarUrl: mediaURL(comment.user?.avatarUrl), text: comment.content || "",
    time: Number(comment.time || 0), likes: Number(comment.likedCount || 0), liked: !!comment.liked,
  };
}

export async function songDetails(values, originals = []) {
  const ids = values.map(idOf);
  const found = new Map(originals.map((song) => [String(song.id), trackEntry(song)]));
  const needed = [...new Set(ids.filter((id) => !found.has(id)))];
  const batchSize = 100;
  for (let start = 0; start < needed.length; start += 4 * batchSize) {
    const group = needed.slice(start, start + 4 * batchSize);
    const requests = [];
    for (let at = 0; at < group.length; at += batchSize) {
      const c = JSON.stringify(group.slice(at, at + batchSize).map((id) => ({ id })));
      requests.push(api("/api/v3/song/detail", { c }));
    }
    for (const result of await Promise.all(requests)) {
      const privileges = new Map(items(result.privileges).map((value) => [String(value.id), value]));
      for (const song of items(result.songs)) found.set(String(song.id), trackEntry(song, privileges.get(String(song.id))));
    }
  }
  // Keep unavailable entries in place so the host can advance by the original
  // page length and playlist moves still target the right position.
  return ids.map((id) => found.get(id) || { id, name: "歌曲 " + id, duration: 0, unavailable: true });
}

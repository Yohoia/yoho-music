const artist = { id: "a1", name: "The Examples" };
const album = { id: "al1", name: "First Album", artists: [artist] };
const tracks = [1, 2, 3].map((number) => ({
  id: `t${number}`,
  name: `Song ${number}`,
  artists: [artist],
  album,
  duration: 12000,
}));

export function search({ query, kind, offset }) {
  if (kind !== "tracks" || offset > 0) return {};
  const term = query.toLowerCase();
  return { tracks: tracks.filter((track) => track.name.toLowerCase().includes(term)) };
}

export async function stream({ id, quality }) {
  if (!tracks.some((track) => track.id === id)) {
    throw Object.assign(new Error("No such track"), { code: "unavailable" });
  }
  return { url: "http://127.0.0.1:8765/tone.mp3", type: "mp3", quality };
}

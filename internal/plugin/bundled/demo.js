import { Storage } from "nsgo";

const track = { id: "tone", name: "Yoho Sound Check", duration: 12000, artists: [{ id: "yoho", name: "Yoho Music" }] };
const listener = { id: "demo", name: "Yoho Demo Listener" };

export function home() {
  return { title: "Yoho Demo", sections: [{ tracks: [track] }] };
}

export function search({ query, kind, offset }) {
  const matches = /yoho|sound|check/i.test(query) || track.name.toLowerCase().includes(query.toLowerCase());
  return kind === "tracks" && offset === 0 && matches ? { tracks: [track] } : {};
}

export function stream({ id }) {
  return id === track.id ? { url: "yoho-demo://audio/tone.mp3", type: "mp3" } : null;
}

export async function account() {
  return (await Storage.getItem("demo-signed-in")) === "yes" ? listener : null;
}

export async function signIn() {
  await Storage.setItem("demo-signed-in", "yes");
  return listener;
}

export async function signOut() {
  await Storage.removeItem("demo-signed-in");
}

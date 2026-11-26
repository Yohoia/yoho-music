import { host, Secrets, Storage } from "nsgo";

export const origin = "https://music.163.com";
const sessionKey = "netease-session";
const desktopOrigin = "https://interfacepc.music.163.com";
const desktopVersion = "3.1.17.204416";
const cookieNames = ["MUSIC_U", "__csrf", "NMTID"];
const desktopRoutes = new Set([
  "/api/song/enhance/player/url/v1", "/api/song/lyric/v1", "/api/song/like/get",
  "/api/v1/radio/get", "/api/v3/song/detail", "/api/play-record/song/list",
  "/api/album/sublist",
  "/api/playmode/intelligence/list",
]);
let device;
let session;
let profile;
let checkedAt = 0;
let generation = 0;

export function fault(code, message) {
  return Object.assign(new Error(message), { code });
}

function signInRequired() {
  return fault("signIn", "请用网易云音乐 App 扫码登录。");
}

function form(fields) {
  return Object.entries(fields)
    .map(([name, value]) => encodeURIComponent(name) + "=" + encodeURIComponent(value))
    .join("&");
}

function sessionCookies(...values) {
  const cookies = {};
  for (const value of values) {
    for (const match of String(value || "").matchAll(/(?:^|[;,]\s*)(MUSIC_U|__csrf|NMTID)=([^;,\s]*)/g)) {
      cookies[match[1]] = match[2];
    }
  }
  return cookies;
}

async function storedCookies() {
  session ??= Secrets.get(sessionKey).then((value) => sessionCookies(value));
  return session;
}

function cookieHeader(cookies) {
  return cookieNames.filter((name) => cookies[name]).map((name) => name + "=" + cookies[name]).join("; ");
}

async function desktopHeaders(cookies) {
  device ??= (async () => {
    const saved = await Storage.getItem("desktop-id");
    if (saved) return saved;
    const id = Array.from({ length: 52 }, () => Math.floor(Math.random() * 16).toString(16)).join("").toUpperCase();
    await Storage.setItem("desktop-id", id);
    return id;
  })();
  const now = Date.now();
  return {
    os: "pc", osver: "Microsoft-Windows-10-Professional-build-19045-64bit",
    appver: desktopVersion, versioncode: "140", channel: "netease", resolution: "1920x1080",
    deviceId: await device, buildver: String(Math.floor(now / 1000)),
    requestId: String(now) + "_" + String(Math.floor(Math.random() * 10000)).padStart(4, "0"),
    __csrf: "", ...cookies,
  };
}

async function request(path, parameters, { cookies, desktop = false, codes = [200] }) {
  const headers = { Accept: "application/json", Referer: origin + "/" };
  const cookie = cookieHeader(cookies);
  if (cookie) headers.Cookie = cookie;
  const fields = cookies.__csrf ? { ...parameters, csrf_token: cookies.__csrf } : parameters;
  let url, body;
  if (desktop) {
    const header = await desktopHeaders(cookies);
    const payload = JSON.stringify({ ...fields, header, e_r: false });
    const params = await host.call("yoho.neteaseEncode", { path, payload });
    headers["Content-Type"] = "application/x-www-form-urlencoded";
    headers["User-Agent"] = "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 Chrome/91.0.4472.164 Safari/537.36 NeteaseMusicDesktop/" + desktopVersion;
    headers.Cookie = Object.entries(header).map(([key, value]) => key + "=" + encodeURIComponent(value)).join("; ");
    url = desktopOrigin + "/eapi/" + path.slice(5);
    body = form({ params });
  } else {
    const query = form(fields);
    url = origin + path + (query ? "?" + query : "");
  }
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), 12000);
  let response, reply;
  try {
    response = await fetch(url, { method: desktop ? "POST" : "GET", headers, body, signal: controller.signal, redirect: "error" });
    if (response.ok) reply = await response.json();
  } catch {
    throw new Error(controller.signal.aborted ? "连接网易云音乐超时，请重试。" : "无法读取网易云音乐，请检查网络后重试。");
  } finally {
    clearTimeout(timer);
  }
  if (response.status === 401) throw signInRequired();
  if (response.status === 429) throw new Error("网易云音乐请求过于频繁，请稍后重试。");
  if (!response.ok) throw new Error("网易云音乐请求失败（HTTP " + response.status + "）。");
  const code = Number(reply?.code);
  if (!codes.includes(code)) {
    if (code === 301 || code === 401) throw signInRequired();
    if (code === 404 || code === 403) throw fault("unavailable", "网易云音乐未开放此内容或操作。");
    const detail = reply?.message || reply?.msg;
    throw new Error(detail ? String(detail).slice(0, 160) : "网易云音乐返回了异常状态。");
  }
  return { body: reply, cookies: sessionCookies(...response.headers.getSetCookie(), reply.cookie) };
}

export async function publicAPI(path, fields = {}) {
  return query(path, fields);
}

export async function api(path, fields = {}, post = false) {
  return query(path, fields, { authenticated: true, desktop: post || desktopRoutes.has(path) });
}

async function query(path, fields, { authenticated = false, desktop = false } = {}) {
  const active = generation;
  const cookies = await storedCookies();
  if (active !== generation || (authenticated && !cookies.MUSIC_U)) throw signInRequired();
  const { body } = await request(path, fields, { cookies, desktop });
  if (active !== generation || cookies !== await storedCookies()) throw signInRequired();
  return body;
}

export function mediaURL(raw) {
  if (!raw) return "";
  try {
    const url = new URL(raw.startsWith("//") ? "https:" + raw : raw);
    if (url.username || url.password || url.port || !url.hostname.endsWith(".music.126.net")) return "";
    if (url.protocol !== "http:" && url.protocol !== "https:") return "";
    return "https:" + url.href.slice(url.protocol.length);
  } catch {
    return "";
  }
}

async function fetchAccount(cookies) {
  const { body } = await request("/api/nuser/account/get", { timestamp: Date.now() }, { cookies });
  const user = body.profile;
  return user?.userId ? { id: String(user.userId), name: user.nickname, imageUrl: mediaURL(user.avatarUrl) } : null;
}

export async function account() {
  const current = await storedCookies();
  if (!current.MUSIC_U) return null;
  if (profile && Date.now() - checkedAt < 60000) return profile;
  let next;
  try {
    next = await fetchAccount(current);
  } catch (error) {
    if (error.code !== "signIn") throw error;
  }
  if (current !== await storedCookies()) return profile;
  if (!next) {
    await signOut();
    return null;
  }
  profile = next;
  checkedAt = Date.now();
  return profile;
}

export async function requireAccount() {
  const current = await account();
  if (!current) throw signInRequired();
  return current;
}

// Bind a sequence of requests to the account and cookies that started it.
export async function requireSession() {
  const active = generation;
  const cookies = await storedCookies();
  const user = await requireAccount();
  if (active !== generation || cookies !== await storedCookies()) throw signInRequired();
  const current = session;
  const check = () => {
    if (active !== generation || current !== session || user.id !== profile?.id) throw signInRequired();
  };
  check();
  return {
    user, key: current, generation: active, check,
    request: async (path, fields = {}, post = false) => {
      check();
      const { body } = await request(path, fields, { cookies, desktop: post || desktopRoutes.has(path) });
      check();
      return body;
    },
  };
}

export async function signOut() {
  generation++;
  profile = null;
  checkedAt = 0;
  session = Promise.resolve({});
  await Secrets.delete(sessionKey);
}

export async function signIn({ requestId }) {
  const attempt = ++generation;
  const notify = async (text, qr = "") => {
    if (attempt !== generation) throw new Error("网易云音乐登录已取消。");
    await host.call("yoho.signInPrompt", { requestId, text, qr });
  };

  await notify("正在获取网易云音乐登录二维码…");
  const start = await request("/api/login/qrcode/unikey", { type: 3 }, { cookies: {}, desktop: true });
  const key = start.body.unikey;
  if (typeof key !== "string" || !key) throw new Error("网易云音乐未返回登录二维码。");
  const qr = origin + "/login?codekey=" + encodeURIComponent(key);
  let cookies = start.cookies;
  let message = "请用网易云音乐 App 扫码。";
  const deadline = Date.now() + 180000;
  while (Date.now() < deadline) {
    await notify(message, qr);
    await new Promise((resolve) => setTimeout(resolve, 2000));
    await notify(message, qr);
    const poll = await request("/api/login/qrcode/client/login", { key, type: 3 },
      { cookies, desktop: true, codes: [800, 801, 802, 803] });
    Object.assign(cookies, poll.cookies);
    const status = Number(poll.body.code);
    if (status === 800) throw new Error("二维码已过期，请重新登录。");
    if (status === 802) message = "已扫码，请在手机上确认登录。";
    if (status !== 803) continue;

    if (!cookies.MUSIC_U) throw new Error("网易云音乐未返回有效会话，请重新登录。");
    const signedIn = await fetchAccount(cookies);
    if (!signedIn) throw new Error("无法读取网易云音乐账号，请重新登录。");
    await notify("正在保存网易云音乐登录状态…");
    await Secrets.set(sessionKey, cookieHeader(cookies));
    try {
      await notify("网易云音乐登录成功。");
    } catch (error) {
      if (attempt === generation) await signOut();
      throw error;
    }
    session = Promise.resolve(cookies);
    profile = signedIn;
    checkedAt = Date.now();
    return signedIn;
  }
  throw new Error("网易云音乐扫码登录超时，请重新登录。");
}

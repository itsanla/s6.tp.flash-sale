// Klien API tipis untuk seluruh aplikasi. Semua permintaan diarahkan ke backend Go
// yang berjalan pada origin yang sama, sehingga tidak perlu konfigurasi alamat khusus.

const BASE = "/api/v1";

const ADMIN_TOKEN_KEY = "wahana_admin_token";
const USER_TOKEN_KEY = "wahana_user_token";

// Token admin dan token pengunjung disimpan terpisah supaya keduanya tidak saling
// menimpa bila seseorang memakai kedua peran pada peramban yang sama.
export function getToken() {
  return localStorage.getItem(ADMIN_TOKEN_KEY);
}
export function setToken(token) {
  localStorage.setItem(ADMIN_TOKEN_KEY, token);
}
export function clearToken() {
  localStorage.removeItem(ADMIN_TOKEN_KEY);
}
export function getUserToken() {
  return localStorage.getItem(USER_TOKEN_KEY);
}
export function setUserToken(token) {
  localStorage.setItem(USER_TOKEN_KEY, token);
}
export function clearUserToken() {
  localStorage.removeItem(USER_TOKEN_KEY);
}

async function request(path, { method = "GET", body, auth = false, user = false } = {}) {
  const headers = { "Content-Type": "application/json" };
  if (auth) {
    const token = getToken();
    if (token) headers.Authorization = `Bearer ${token}`;
  }
  if (user) {
    const token = getUserToken();
    if (token) headers.Authorization = `Bearer ${token}`;
  }
  const res = await fetch(BASE + path, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });
  const json = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(json.message || "Terjadi kesalahan pada server");
  }
  return json.data;
}

export const api = {
  // Katalog
  categories: () => request("/categories"),
  rides: (params = {}) => {
    const q = new URLSearchParams();
    if (params.category) q.set("category", params.category);
    if (params.date) q.set("date", params.date);
    const qs = q.toString();
    return request(`/rides${qs ? `?${qs}` : ""}`);
  },
  ride: (slug, date) => request(`/rides/${slug}${date ? `?date=${date}` : ""}`),

  // Akun pengunjung
  register: (payload) => request("/auth/register", { method: "POST", body: payload }),
  loginUser: (email, password) =>
    request("/auth/login", { method: "POST", body: { email, password } }),
  me: () => request("/me", { user: true }),
  updateMe: (payload) => request("/me", { method: "PUT", body: payload, user: true }),
  myOrders: () => request("/me/orders", { user: true }),

  // Pemesanan
  checkout: (payload) => request("/orders", { method: "POST", body: payload, user: true }),
  order: (code) => request(`/orders/${code}`),
  cancelOrder: (code) => request(`/orders/${code}/cancel`, { method: "POST", body: {} }),
  tickets: (code) => request(`/orders/${code}/tickets`),
  scanTicket: (code) => request(`/tickets/${code}/scan`, { method: "POST", body: {} }),

  // Halaman uji pembayaran
  pendingOrders: () => request("/test/pending-orders"),
  settle: (code) => request(`/test/orders/${code}/settle`, { method: "POST", body: {} }),
  system: () => request("/test/system"),

  // Admin
  login: (username, password) =>
    request("/auth/admin", { method: "POST", body: { username, password } }),
  stats: () => request("/admin/stats", { auth: true }),
  adminOrders: () => request("/admin/orders", { auth: true }),
  createRide: (payload) => request("/admin/rides", { method: "POST", body: payload, auth: true }),
  updateRide: (id, payload) =>
    request(`/admin/rides/${id}`, { method: "PUT", body: payload, auth: true }),
  deleteRide: (id) => request(`/admin/rides/${id}`, { method: "DELETE", auth: true }),
};

// ------------------------------------------------------------------ Pembantu

export function rupiah(value) {
  return "Rp" + Number(value || 0).toLocaleString("id-ID");
}

export function todayISO() {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(
    d.getDate()
  ).padStart(2, "0")}`;
}

export function formatDate(iso) {
  if (!iso) return "-";
  const [y, m, d] = iso.split("-").map(Number);
  const bulan = [
    "Januari", "Februari", "Maret", "April", "Mei", "Juni",
    "Juli", "Agustus", "September", "Oktober", "November", "Desember",
  ];
  return `${d} ${bulan[m - 1]} ${y}`;
}

export function formatTime(value) {
  if (!value) return "-";
  const d = new Date(value);
  return d.toLocaleString("id-ID", {
    day: "2-digit",
    month: "short",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function countdownText(seconds) {
  const s = Math.max(0, Math.floor(seconds));
  const m = String(Math.floor(s / 60)).padStart(2, "0");
  return `${m}:${String(s % 60).padStart(2, "0")}`;
}

// Label tampilan untuk tiap zona wahana.
export const CATEGORY_LABELS = {
  ekstrem: "Ekstrem",
  keluarga: "Keluarga",
  anak: "Anak",
  air: "Wahana Air",
  petualangan: "Petualangan",
  indoor: "Indoor",
};

export const STATUS_META = {
  PENDING: { label: "Menunggu pembayaran", cls: "badge-orange" },
  PAID: { label: "Lunas", cls: "badge-green" },
  EXPIRED: { label: "Kedaluwarsa", cls: "badge-gray" },
  CANCELLED: { label: "Dibatalkan", cls: "badge-red" },
  ISSUED: { label: "Siap dipakai", cls: "badge-green" },
  USED: { label: "Sudah dipakai", cls: "badge-gray" },
};

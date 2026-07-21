// Klien API tipis untuk seluruh aplikasi. Semua permintaan diarahkan ke backend Go
// yang berjalan pada origin yang sama, sehingga tidak perlu konfigurasi alamat khusus.

const BASE = "/api/v1";

const TOKEN_KEY = "wahana_admin_token";

export function getToken() {
  return localStorage.getItem(TOKEN_KEY);
}
export function setToken(token) {
  localStorage.setItem(TOKEN_KEY, token);
}
export function clearToken() {
  localStorage.removeItem(TOKEN_KEY);
}

async function request(path, { method = "GET", body, auth = false } = {}) {
  const headers = { "Content-Type": "application/json" };
  if (auth) {
    const token = getToken();
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

  // Pemesanan
  checkout: (payload) => request("/orders", { method: "POST", body: payload }),
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
    request("/auth/login", { method: "POST", body: { username, password } }),
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

export const STATUS_META = {
  PENDING: { label: "Menunggu pembayaran", cls: "badge-orange" },
  PAID: { label: "Lunas", cls: "badge-green" },
  EXPIRED: { label: "Kedaluwarsa", cls: "badge-gray" },
  CANCELLED: { label: "Dibatalkan", cls: "badge-red" },
  ISSUED: { label: "Siap dipakai", cls: "badge-green" },
  USED: { label: "Sudah dipakai", cls: "badge-gray" },
};

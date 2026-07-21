import { useCallback, useEffect, useState } from "react";
import { api, CATEGORY_LABELS, clearToken, formatTime, getToken, rupiah, setToken } from "../api";
import { Empty, Loading, StatusBadge } from "../components";
import { useApp } from "../store";

const KATEGORI = ["ekstrem", "keluarga", "anak", "air", "petualangan", "indoor"];

const KOSONG = {
  slug: "", name: "", category: "keluarga", tagline: "", description: "",
  emoji: "🎠", image_url: "", price: 30000, duration_min: 10, min_height_cm: 0,
  thrill_level: 1, daily_quota: 300, is_active: true,
};

export default function Admin() {
  const { toast } = useApp();
  const [masuk, setMasuk] = useState(Boolean(getToken()));
  const [kredensial, setKredensial] = useState({ username: "admin", password: "" });

  const [stats, setStats] = useState(null);
  const [rides, setRides] = useState(null);
  const [orders, setOrders] = useState([]);
  const [form, setForm] = useState(null);

  const muat = useCallback(async () => {
    try {
      const [s, r, o] = await Promise.all([api.stats(), api.rides(), api.adminOrders()]);
      setStats(s);
      setRides(r || []);
      setOrders(o || []);
    } catch (e) {
      if (String(e.message).toLowerCase().includes("sesi") || String(e.message).includes("login")) {
        clearToken();
        setMasuk(false);
      }
      toast(e.message, "err");
    }
  }, [toast]);

  useEffect(() => {
    if (masuk) muat();
  }, [masuk, muat]);

  const login = async (e) => {
    e.preventDefault();
    try {
      const data = await api.login(kredensial.username, kredensial.password);
      setToken(data.token);
      setMasuk(true);
      toast("Berhasil masuk sebagai admin");
    } catch (err) {
      toast(err.message, "err");
    }
  };

  const keluar = () => {
    clearToken();
    setMasuk(false);
    setStats(null);
    setRides(null);
    toast("Anda sudah keluar dari mode admin");
  };

  const simpan = async (e) => {
    e.preventDefault();
    try {
      if (form.id) {
        await api.updateRide(form.id, form);
        toast("Wahana diperbarui");
      } else {
        const slug = form.slug || form.name.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "");
        await api.createRide({ ...form, slug });
        toast("Wahana ditambahkan");
      }
      setForm(null);
      muat();
    } catch (err) {
      toast(err.message, "err");
    }
  };

  const hapus = async (ride) => {
    if (!confirm(`Hapus wahana ${ride.name}?`)) return;
    try {
      await api.deleteRide(ride.id);
      toast("Wahana dihapus");
      muat();
    } catch (err) {
      toast(err.message, "err");
    }
  };

  if (!masuk) {
    return (
      <div className="container page">
        <div className="card" style={{ maxWidth: 400, margin: "40px auto" }}>
          <div className="text-center mb-16">
            <div className="empty-emoji">🔐</div>
            <h1 className="title-sm">Masuk sebagai Admin</h1>
            <div className="caption">Kelola katalog wahana dan pantau penjualan tiket</div>
          </div>
          <form onSubmit={login} className="stack">
            <div className="field">
              <label className="label" htmlFor="user">Username</label>
              <input
                id="user"
                className="input"
                value={kredensial.username}
                onChange={(e) => setKredensial({ ...kredensial, username: e.target.value })}
              />
            </div>
            <div className="field">
              <label className="label" htmlFor="pass">Password</label>
              <input
                id="pass"
                type="password"
                className="input"
                value={kredensial.password}
                onChange={(e) => setKredensial({ ...kredensial, password: e.target.value })}
              />
            </div>
            <button className="btn btn-primary btn-block" type="submit">Masuk</button>
          </form>
        </div>
      </div>
    );
  }

  return (
    <div className="container page">
      <div className="section-head">
        <div>
          <h1 className="title-lg">Dasbor Admin</h1>
          <div className="subtitle">Ringkasan penjualan dan pengelolaan katalog wahana</div>
        </div>
        <div className="row">
          <button className="btn btn-gray btn-sm" onClick={muat}>Muat ulang</button>
          <button className="btn btn-danger btn-sm" onClick={keluar}>Keluar</button>
        </div>
      </div>

      {stats && (
        <div className="stat-grid mb-16">
          <div className="stat"><div className="stat-label">Total wahana</div><div className="stat-value">{stats.total_rides}</div></div>
          <div className="stat"><div className="stat-label">Total pesanan</div><div className="stat-value">{stats.total_orders}</div></div>
          <div className="stat"><div className="stat-label">Pesanan lunas</div><div className="stat-value" style={{ color: "var(--green)" }}>{stats.paid_orders}</div></div>
          <div className="stat"><div className="stat-label">Menunggu bayar</div><div className="stat-value" style={{ color: "var(--orange)" }}>{stats.pending_orders}</div></div>
          <div className="stat"><div className="stat-label">Tiket terbit</div><div className="stat-value">{stats.total_tickets}</div></div>
          <div className="stat"><div className="stat-label">Pendapatan</div><div className="stat-value" style={{ fontSize: 20, color: "var(--blue)" }}>{rupiah(stats.revenue)}</div></div>
        </div>
      )}

      <div className="section-head mt-32">
        <h2 className="title">Katalog wahana</h2>
        <button className="btn btn-primary btn-sm" onClick={() => setForm({ ...KOSONG })}>
          Tambah wahana
        </button>
      </div>

      {form && (
        <form className="card mb-16" onSubmit={simpan}>
          <div className="title-sm mb-16">{form.id ? "Ubah wahana" : "Wahana baru"}</div>
          <div className="form-grid">
            <div className="field">
              <label className="label">Nama wahana</label>
              <input className="input" required value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
            </div>
            <div className="field">
              <label className="label">Kategori</label>
              <select className="select" value={form.category} onChange={(e) => setForm({ ...form, category: e.target.value })}>
                {KATEGORI.map((k) => <option key={k} value={k}>{CATEGORY_LABELS[k]}</option>)}
              </select>
            </div>
            <div className="field">
              <label className="label">Emoji</label>
              <input className="input" value={form.emoji} onChange={(e) => setForm({ ...form, emoji: e.target.value })} />
            </div>
            <div className="field">
              <label className="label">Harga tiket</label>
              <input type="number" className="input" min="0" value={form.price} onChange={(e) => setForm({ ...form, price: Number(e.target.value) })} />
            </div>
            <div className="field">
              <label className="label">Durasi (menit)</label>
              <input type="number" className="input" min="0" value={form.duration_min} onChange={(e) => setForm({ ...form, duration_min: Number(e.target.value) })} />
            </div>
            <div className="field">
              <label className="label">Tinggi minimum (cm)</label>
              <input type="number" className="input" min="0" value={form.min_height_cm} onChange={(e) => setForm({ ...form, min_height_cm: Number(e.target.value) })} />
            </div>
            <div className="field">
              <label className="label">Tingkat tantangan (1 sampai 5)</label>
              <input type="number" className="input" min="1" max="5" value={form.thrill_level} onChange={(e) => setForm({ ...form, thrill_level: Number(e.target.value) })} />
            </div>
            <div className="field">
              <label className="label">Kuota harian</label>
              <input type="number" className="input" min="0" value={form.daily_quota} onChange={(e) => setForm({ ...form, daily_quota: Number(e.target.value) })} />
            </div>
          </div>
          <div className="field mt-16">
            <label className="label">Alamat gambar wahana</label>
            <input
              className="input"
              placeholder="https://..."
              value={form.image_url || ""}
              onChange={(e) => setForm({ ...form, image_url: e.target.value })}
            />
          </div>
          <div className="field mt-16">
            <label className="label">Deskripsi singkat</label>
            <input className="input" value={form.tagline} onChange={(e) => setForm({ ...form, tagline: e.target.value })} />
          </div>
          <div className="field mt-16">
            <label className="label">Deskripsi lengkap</label>
            <textarea className="textarea" value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
          </div>
          <label className="row mt-16" style={{ gap: 8, cursor: "pointer" }}>
            <input type="checkbox" checked={form.is_active} onChange={(e) => setForm({ ...form, is_active: e.target.checked })} />
            <span className="label">Wahana beroperasi</span>
          </label>
          <div className="row mt-16">
            <button className="btn btn-primary" type="submit">Simpan</button>
            <button className="btn btn-gray" type="button" onClick={() => setForm(null)}>Batal</button>
          </div>
        </form>
      )}

      {rides === null ? (
        <Loading />
      ) : (
        <div className="table-wrap mb-16">
          <table>
            <thead>
              <tr>
                <th>Wahana</th><th>Kategori</th><th>Harga</th><th>Kuota</th><th>Status</th><th>Aksi</th>
              </tr>
            </thead>
            <tbody>
              {rides.map((r) => (
                <tr key={r.id}>
                  <td><span style={{ marginRight: 8 }}>{r.emoji}</span>{r.name}</td>
                  <td><span className="badge">{CATEGORY_LABELS[r.category] || r.category}</span></td>
                  <td>{rupiah(r.price)}</td>
                  <td>{r.daily_quota}</td>
                  <td>
                    <span className={`badge ${r.is_active ? "badge-green" : "badge-gray"}`}>
                      {r.is_active ? "Beroperasi" : "Tutup"}
                    </span>
                  </td>
                  <td>
                    <div className="row" style={{ gap: 6 }}>
                      <button className="btn btn-tinted btn-sm" onClick={() => setForm({ ...r })}>Ubah</button>
                      <button className="btn btn-danger btn-sm" onClick={() => hapus(r)}>Hapus</button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <div className="section-head mt-32">
        <h2 className="title">Pesanan terbaru</h2>
      </div>
      {orders.length === 0 ? (
        <div className="card"><Empty emoji="🧾" title="Belum ada pesanan" /></div>
      ) : (
        <div className="table-wrap">
          <table>
            <thead>
              <tr><th>Kode</th><th>Pemesan</th><th>Kunjungan</th><th>Total</th><th>Status</th><th>Dibuat</th></tr>
            </thead>
            <tbody>
              {orders.map((o) => (
                <tr key={o.code}>
                  <td className="mono">{o.code}</td>
                  <td>{o.customer_name}</td>
                  <td>{o.visit_date}</td>
                  <td>{rupiah(o.total_amount)}</td>
                  <td><StatusBadge status={o.status} /></td>
                  <td className="caption">{formatTime(o.created_at)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

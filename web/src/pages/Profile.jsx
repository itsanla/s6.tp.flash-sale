import { useCallback, useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { api, formatDate, formatTime, rupiah } from "../api";
import { Empty, Loading, StatusBadge } from "../components";
import { useApp } from "../store";

export default function Profile() {
  const navigate = useNavigate();
  const { user, setUser, authReady, signOut, toast } = useApp();

  const [orders, setOrders] = useState(null);
  const [editing, setEditing] = useState(false);
  const [form, setForm] = useState({ name: "", phone: "" });
  const [busy, setBusy] = useState(false);

  // Halaman ini khusus pemilik akun, jadi pengunjung yang belum masuk diarahkan
  // ke halaman masuk sambil membawa tujuan semula.
  useEffect(() => {
    if (authReady && !user) navigate("/masuk?lanjut=/profil", { replace: true });
  }, [authReady, user, navigate]);

  useEffect(() => {
    if (user) setForm({ name: user.name, phone: user.phone || "" });
  }, [user]);

  const muat = useCallback(async () => {
    try {
      setOrders(await api.myOrders());
    } catch {
      setOrders([]);
    }
  }, []);

  useEffect(() => {
    if (user) muat();
  }, [user, muat]);

  const simpan = async (e) => {
    e.preventDefault();
    setBusy(true);
    try {
      const updated = await api.updateMe(form);
      setUser(updated);
      setEditing(false);
      toast("Profil diperbarui");
    } catch (err) {
      toast(err.message, "err");
    } finally {
      setBusy(false);
    }
  };

  const keluar = () => {
    signOut();
    toast("Anda sudah keluar dari akun");
    navigate("/");
  };

  if (!authReady || !user) return <div className="container page"><Loading text="Memuat akun" /></div>;

  const initial = user.name.trim().charAt(0).toUpperCase();
  const totalDibayar = (orders || [])
    .filter((o) => o.status === "PAID")
    .reduce((a, o) => a + o.total_amount, 0);

  return (
    <div className="container page">
      <div className="profile-head">
        <div className="avatar">{initial}</div>
        <div className="grow">
          <h1 className="title">{user.name}</h1>
          <div className="subtitle">{user.email}</div>
          <div className="caption">Bergabung sejak {formatTime(user.created_at)}</div>
        </div>
        <button className="btn btn-danger" onClick={keluar}>Keluar</button>
      </div>

      <div className="stat-grid mt-24">
        <div className="stat">
          <div className="stat-label">Total pesanan</div>
          <div className="stat-value">{orders ? orders.length : "-"}</div>
        </div>
        <div className="stat">
          <div className="stat-label">Pesanan lunas</div>
          <div className="stat-value" style={{ color: "var(--green)" }}>
            {orders ? orders.filter((o) => o.status === "PAID").length : "-"}
          </div>
        </div>
        <div className="stat">
          <div className="stat-label">Total dibelanjakan</div>
          <div className="stat-value" style={{ fontSize: 20, color: "var(--blue)" }}>
            {rupiah(totalDibayar)}
          </div>
        </div>
      </div>

      <div className="card mt-24">
        <div className="row-between mb-16">
          <div className="title-sm">Data akun</div>
          {!editing && (
            <button className="btn btn-tinted btn-sm" onClick={() => setEditing(true)}>
              Ubah data
            </button>
          )}
        </div>

        {editing ? (
          <form onSubmit={simpan} className="stack">
            <div className="form-grid">
              <div className="field">
                <label className="label" htmlFor="pnama">Nama lengkap</label>
                <input
                  id="pnama"
                  className="input"
                  required
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                />
              </div>
              <div className="field">
                <label className="label" htmlFor="ptelepon">Nomor telepon</label>
                <input
                  id="ptelepon"
                  className="input"
                  placeholder="08xxxxxxxxxx"
                  value={form.phone}
                  onChange={(e) => setForm({ ...form, phone: e.target.value })}
                />
              </div>
            </div>
            <div className="row">
              <button className="btn btn-primary" type="submit" disabled={busy}>
                {busy ? "Menyimpan..." : "Simpan"}
              </button>
              <button
                className="btn btn-gray"
                type="button"
                onClick={() => {
                  setEditing(false);
                  setForm({ name: user.name, phone: user.phone || "" });
                }}
              >
                Batal
              </button>
            </div>
          </form>
        ) : (
          <div className="chips">
            <div className="chip">👤 <b>{user.name}</b></div>
            <div className="chip">✉️ <b>{user.email}</b></div>
            <div className="chip">📱 <b>{user.phone || "belum diisi"}</b></div>
          </div>
        )}
      </div>

      <div className="section-head mt-32">
        <h2 className="title">Riwayat pemesanan</h2>
        <Link to="/wahana" className="btn btn-tinted btn-sm">Pesan lagi</Link>
      </div>

      {orders === null ? (
        <Loading text="Mengambil riwayat pemesanan" />
      ) : orders.length === 0 ? (
        <div className="card">
          <Empty
            emoji="🎠"
            title="Belum ada pemesanan"
            desc="Pesanan yang Anda buat saat masuk ke akun akan tersimpan di sini."
            action={<Link to="/wahana" className="btn btn-primary">Jelajahi wahana</Link>}
          />
        </div>
      ) : (
        <div className="stack">
          {orders.map((o) => (
            <div key={o.code} className="card">
              <div className="row-between wrap">
                <div>
                  <div className="mono" style={{ fontWeight: 700 }}>{o.code}</div>
                  <div className="caption">
                    Kunjungan {formatDate(o.visit_date)} · dipesan {formatTime(o.created_at)}
                  </div>
                </div>
                <div style={{ textAlign: "right" }}>
                  <div className="price">{rupiah(o.total_amount)}</div>
                  <StatusBadge status={o.status} />
                </div>
              </div>

              <div className="chips mt-16">
                {o.items.map((i) => (
                  <span key={i.ride_id} className="badge">
                    {i.ride_emoji} {i.ride_name} x{i.quantity}
                  </span>
                ))}
              </div>

              <div className="row mt-16 wrap">
                {o.status === "PAID" && (
                  <Link to={`/tiket/${o.code}`} className="btn btn-success btn-sm">
                    Lihat tiket
                  </Link>
                )}
                {o.status === "PENDING" && (
                  <Link to={`/pembayaran/${o.code}`} className="btn btn-primary btn-sm">
                    Selesaikan pembayaran
                  </Link>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

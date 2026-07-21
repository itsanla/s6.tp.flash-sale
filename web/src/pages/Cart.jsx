import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { api, formatDate, rupiah, todayISO } from "../api";
import { Empty, Stepper } from "../components";
import { useApp } from "../store";

export default function Cart() {
  const navigate = useNavigate();
  const { cart, totals, updateQuantity, removeItem, setVisitDate, clearCart, toast, user } = useApp();

  const [form, setForm] = useState({ name: "", email: "", phone: "" });
  const [submitting, setSubmitting] = useState(false);

  // Data pemesan diisikan otomatis dari akun supaya pengunjung tidak perlu mengetik ulang.
  useEffect(() => {
    if (user) {
      setForm((f) => ({
        name: f.name || user.name || "",
        email: f.email || user.email || "",
        phone: f.phone || user.phone || "",
      }));
    }
  }, [user]);

  const date = cart.visitDate || todayISO();
  const set = (k) => (e) => setForm((f) => ({ ...f, [k]: e.target.value }));

  const bayar = async () => {
    if (!form.name.trim()) {
      toast("Nama pemesan wajib diisi", "err");
      return;
    }
    setSubmitting(true);
    try {
      const data = await api.checkout({
        customer_name: form.name,
        customer_email: form.email,
        customer_phone: form.phone,
        visit_date: date,
        items: cart.items.map((i) => ({ ride_id: i.rideId, quantity: i.quantity })),
      });
      clearCart();
      navigate(`/pembayaran/${data.order.code}`);
    } catch (e) {
      toast(e.message, "err");
    } finally {
      setSubmitting(false);
    }
  };

  if (cart.items.length === 0) {
    return (
      <div className="container page">
        <h1 className="title-lg mb-16">Keranjang</h1>
        <div className="card">
          <Empty
            emoji="🛒"
            title="Keranjang Anda masih kosong"
            desc="Pilih wahana favorit terlebih dahulu, lalu kembali ke halaman ini untuk menyelesaikan pemesanan."
            action={<Link to="/wahana" className="btn btn-primary">Jelajahi wahana</Link>}
          />
        </div>
      </div>
    );
  }

  return (
    <div className="container page">
      <h1 className="title-lg mb-16">Keranjang</h1>

      <div style={{ display: "grid", gridTemplateColumns: "1.4fr 1fr", gap: 22, alignItems: "start" }} className="cart-layout">
        <div className="stack">
          <div className="list">
            {cart.items.map((item) => (
              <div key={item.rideId} className="list-row">
                <div className={`list-icon cat-${item.category}`}>{item.emoji}</div>
                <div className="grow">
                  <div style={{ fontWeight: 650 }}>{item.name}</div>
                  <div className="caption">{rupiah(item.price)} per tiket</div>
                </div>
                <Stepper
                  value={item.quantity}
                  onChange={(v) => updateQuantity(item.rideId, v)}
                  min={0}
                  max={20}
                />
                <div style={{ minWidth: 96, textAlign: "right", fontWeight: 700 }}>
                  {rupiah(item.price * item.quantity)}
                </div>
                <button
                  className="btn btn-danger btn-sm"
                  onClick={() => removeItem(item.rideId)}
                  aria-label={`Hapus ${item.name}`}
                >
                  Hapus
                </button>
              </div>
            ))}
          </div>

          <div className="card">
            <div className="row-between mb-16">
              <div className="title-sm">Data pemesan</div>
              {!user && (
                <Link to="/masuk?lanjut=/keranjang" className="btn btn-tinted btn-sm">
                  Masuk untuk menyimpan riwayat
                </Link>
              )}
            </div>
            <div className="stack">
              <div className="field">
                <label className="label" htmlFor="nama">Nama lengkap</label>
                <input id="nama" className="input" placeholder="Nama sesuai identitas" value={form.name} onChange={set("name")} />
              </div>
              <div className="form-grid">
                <div className="field">
                  <label className="label" htmlFor="email">Email (opsional)</label>
                  <input id="email" type="email" className="input" placeholder="nama@email.com" value={form.email} onChange={set("email")} />
                </div>
                <div className="field">
                  <label className="label" htmlFor="telepon">Nomor telepon (opsional)</label>
                  <input id="telepon" className="input" placeholder="08xxxxxxxxxx" value={form.phone} onChange={set("phone")} />
                </div>
              </div>
              <div className="field">
                <label className="label" htmlFor="tanggal-kunjungan">Tanggal kunjungan</label>
                <input
                  id="tanggal-kunjungan"
                  type="date"
                  className="input"
                  min={todayISO()}
                  value={date}
                  onChange={(e) => setVisitDate(e.target.value)}
                />
                <div className="caption">Seluruh tiket berlaku untuk {formatDate(date)}</div>
              </div>
            </div>
          </div>
        </div>

        <div className="card" style={{ position: "sticky", top: 78 }}>
          <div className="title-sm mb-16">Ringkasan pesanan</div>
          {cart.items.map((i) => (
            <div key={i.rideId} className="row-between" style={{ marginBottom: 10 }}>
              <span className="caption" style={{ maxWidth: "62%" }}>
                {i.emoji} {i.name} x{i.quantity}
              </span>
              <span style={{ fontWeight: 600, fontSize: 14 }}>{rupiah(i.price * i.quantity)}</span>
            </div>
          ))}
          <div className="divider" />
          <div className="row-between">
            <span className="label">Jumlah tiket</span>
            <span style={{ fontWeight: 700 }}>{totals.count}</span>
          </div>
          <div className="row-between mt-8">
            <span className="label">Total bayar</span>
            <span className="title" style={{ color: "var(--blue)" }}>{rupiah(totals.amount)}</span>
          </div>

          <button className="btn btn-primary btn-block btn-lg mt-24" onClick={bayar} disabled={submitting}>
            {submitting ? "Memproses..." : "Bayar dengan QRIS"}
          </button>
          <div className="caption text-center mt-8">
            Kode QRIS akan terbit setelah pesanan dibuat
          </div>
        </div>
      </div>

      <style>{`
        @media (max-width: 900px) {
          .cart-layout { grid-template-columns: 1fr !important; }
        }
        @media (max-width: 640px) {
          .list-row { flex-wrap: wrap; }
        }
      `}</style>
    </div>
  );
}

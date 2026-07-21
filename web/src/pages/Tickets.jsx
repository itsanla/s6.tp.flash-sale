import { useCallback, useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { api, formatDate, formatTime, rupiah } from "../api";
import { Empty, Loading, StatusBadge } from "../components";
import { useApp } from "../store";

export default function Tickets() {
  const { code } = useParams();
  const navigate = useNavigate();
  const { toast } = useApp();

  const [input, setInput] = useState(code || "");
  const [order, setOrder] = useState(null);
  const [tickets, setTickets] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const muat = useCallback(
    async (kode) => {
      if (!kode) return;
      setLoading(true);
      setError("");
      try {
        const data = await api.order(kode);
        setOrder(data.order);
        const t = await api.tickets(kode);
        setTickets(t || []);
      } catch (e) {
        setError(e.message);
        setOrder(null);
        setTickets([]);
      } finally {
        setLoading(false);
      }
    },
    []
  );

  useEffect(() => {
    if (code) muat(code);
  }, [code, muat]);

  // Tiket diterbitkan worker lewat antrean, sehingga daftar diperbarui berkala sampai
  // seluruh tiket muncul. Inilah bukti nyata pemrosesan asinkron pada aplikasi ini.
  useEffect(() => {
    if (!order || order.status !== "PAID" || tickets.length > 0) return;
    const timer = setInterval(() => muat(order.code), 1500);
    return () => clearInterval(timer);
  }, [order, tickets.length, muat]);

  const cari = (e) => {
    e.preventDefault();
    const kode = input.trim().toUpperCase();
    if (kode) navigate(`/tiket/${kode}`);
  };

  const pakaiTiket = async (kodeTiket) => {
    try {
      await api.scanTicket(kodeTiket);
      toast("Tiket berhasil diverifikasi di gerbang");
      muat(order.code);
    } catch (e) {
      toast(e.message, "err");
    }
  };

  return (
    <div className="container page">
      <h1 className="title-lg mb-16">Tiket Saya</h1>

      <form className="card mb-16" onSubmit={cari}>
        <div className="field">
          <label className="label" htmlFor="kode">Masukkan kode pesanan</label>
          <div className="row">
            <input
              id="kode"
              className="input grow"
              placeholder="Contoh: ORD-A1B2C3D4E5F6"
              value={input}
              onChange={(e) => setInput(e.target.value)}
            />
            <button className="btn btn-primary" type="submit">Cari</button>
          </div>
          <div className="caption">
            Kode pesanan diberikan setelah Anda menyelesaikan pemesanan tiket.
          </div>
        </div>
      </form>

      {loading && <Loading text="Mengambil data tiket" />}

      {error && !loading && (
        <div className="card">
          <Empty emoji="🎫" title={error} desc="Periksa kembali kode pesanan yang Anda masukkan." />
        </div>
      )}

      {order && !loading && (
        <>
          <div className="card mb-16">
            <div className="row-between wrap">
              <div>
                <div className="caption">Kode pesanan</div>
                <div className="title-sm mono">{order.code}</div>
              </div>
              <StatusBadge status={order.status} />
            </div>
            <div className="divider" />
            <div className="chips">
              <div className="chip">👤 <b>{order.customer_name}</b></div>
              <div className="chip">📅 <b>{formatDate(order.visit_date)}</b></div>
              <div className="chip">💳 <b>{rupiah(order.total_amount)}</b></div>
              {order.paid_at && <div className="chip">✅ Dibayar <b>{formatTime(order.paid_at)}</b></div>}
            </div>
          </div>

          {order.status === "PAID" && tickets.length === 0 && (
            <div className="card">
              <div className="row" style={{ justifyContent: "center" }}>
                <div className="spinner" />
                <div>
                  <div style={{ fontWeight: 650 }}>Tiket sedang diterbitkan</div>
                  <div className="caption">
                    Pesanan Anda sudah masuk antrean penerbitan tiket. Halaman ini akan
                    memperbarui sendiri begitu tiket selesai dibuat.
                  </div>
                </div>
              </div>
            </div>
          )}

          {order.status !== "PAID" && tickets.length === 0 && (
            <div className="card">
              <Empty
                emoji="⏳"
                title="Tiket belum terbit"
                desc="Tiket hanya diterbitkan setelah pembayaran pesanan diterima."
              />
            </div>
          )}

          {tickets.length > 0 && (
            <div className="stack">
              <div className="section-head">
                <h2 className="title-sm">{tickets.length} tiket elektronik</h2>
                <span className="caption">Tunjukkan kode ini di gerbang wahana</span>
              </div>
              {tickets.map((t) => (
                <div key={t.code} className={`ticket ${t.status === "USED" ? "used" : ""}`}>
                  <div className="ticket-emoji">{t.ride_emoji}</div>
                  <div className="grow">
                    <div style={{ fontWeight: 700 }}>{t.ride_name}</div>
                    <div className="mono caption">{t.code}</div>
                    <div className="caption">Berlaku {formatDate(t.visit_date)}</div>
                  </div>
                  <div className="stack" style={{ gap: 8, alignItems: "flex-end" }}>
                    <StatusBadge status={t.status} />
                    {t.status === "ISSUED" ? (
                      <button className="btn btn-tinted btn-sm" onClick={() => pakaiTiket(t.code)}>
                        Pindai di gerbang
                      </button>
                    ) : (
                      <span className="caption">Dipakai {formatTime(t.used_at)}</span>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}
        </>
      )}
    </div>
  );
}

import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { api, countdownText, formatDate, formatTime, rupiah } from "../api";
import { Empty, Loading } from "../components";
import { useApp } from "../store";

// Halaman uji pembayaran QRIS.
//
// Pada sistem sungguhan, status pembayaran diterima lewat notifikasi dari penyedia
// pembayaran. Karena aplikasi ini adalah demonstrasi, pelunasan dilakukan manual dari
// halaman ini. Menekan tombol selesaikan pembayaran akan menandai pesanan lunas lalu
// mengantrekan penerbitan tiket ke RabbitMQ.
export default function QrisTest() {
  const { toast } = useApp();
  const [orders, setOrders] = useState(null);
  const [system, setSystem] = useState(null);
  const [busy, setBusy] = useState("");

  const muat = useCallback(async () => {
    try {
      const data = await api.pendingOrders();
      setOrders(data || []);
    } catch {
      setOrders([]);
    }
    try {
      const s = await api.system();
      setSystem(s);
    } catch {
      setSystem(null);
    }
  }, []);

  useEffect(() => {
    muat();
    const timer = setInterval(muat, 4000);
    return () => clearInterval(timer);
  }, [muat]);

  const selesaikan = async (code) => {
    setBusy(code);
    try {
      await api.settle(code);
      toast(`Pembayaran ${code} berhasil, tiket masuk antrean penerbitan`);
      await muat();
    } catch (e) {
      toast(e.message, "err");
    } finally {
      setBusy("");
    }
  };

  return (
    <div className="container page">
      <div className="section-head">
        <div>
          <h1 className="title-lg">Uji Pembayaran QRIS</h1>
          <div className="subtitle">
            Daftar pesanan yang menunggu pembayaran beserta kode QRIS masing masing
          </div>
        </div>
        <button className="btn btn-gray btn-sm" onClick={muat}>Muat ulang</button>
      </div>

      <div className="note note-blue mb-16">
        Halaman ini menggantikan notifikasi dari penyedia pembayaran sungguhan. Kode QRIS yang
        dibuat aplikasi hanya berformat benar untuk keperluan demonstrasi dan tidak terhubung ke
        rekening mana pun, sehingga pelunasan dilakukan lewat tombol di halaman ini.
      </div>

      {system && (
        <div className="stat-grid mb-16">
          {system.queues.map((q) => (
            <div key={q.queue} className="stat">
              <div className="stat-label">{q.name}</div>
              <div className="stat-value">{q.messages < 0 ? "-" : q.messages}</div>
              <div className="caption mono" style={{ fontSize: 11 }}>{q.queue}</div>
            </div>
          ))}
        </div>
      )}

      {orders === null ? (
        <Loading text="Mengambil daftar pesanan" />
      ) : orders.length === 0 ? (
        <div className="card">
          <Empty
            emoji="✅"
            title="Tidak ada pesanan yang menunggu pembayaran"
            desc="Buat pesanan baru dari katalog wahana untuk melihat kode QRIS di sini."
            action={<Link to="/wahana" className="btn btn-primary">Buka katalog wahana</Link>}
          />
        </div>
      ) : (
        <div className="stack">
          {orders.map(({ order, qris_image, seconds_left }) => (
            <div key={order.code} className="card">
              <div className="row wrap" style={{ alignItems: "flex-start", gap: 18 }}>
                {qris_image && (
                  <img
                    src={qris_image}
                    alt={`QRIS ${order.code}`}
                    style={{
                      width: 140,
                      height: 140,
                      borderRadius: "var(--r-md)",
                      border: "1px solid var(--separator)",
                      padding: 8,
                      background: "#fff",
                      flex: "0 0 auto",
                    }}
                  />
                )}

                <div className="grow" style={{ minWidth: 240 }}>
                  <div className="row-between wrap">
                    <div>
                      <div className="mono title-sm">{order.code}</div>
                      <div className="caption">
                        {order.customer_name} - kunjungan {formatDate(order.visit_date)}
                      </div>
                    </div>
                    <div style={{ textAlign: "right" }}>
                      <div className="price">{rupiah(order.total_amount)}</div>
                      <div className={`caption ${seconds_left < 60 ? "" : ""}`}>
                        Sisa waktu {countdownText(seconds_left)}
                      </div>
                    </div>
                  </div>

                  <div className="divider" />

                  <div className="chips" style={{ gap: 8 }}>
                    {order.items.map((i) => (
                      <span key={i.ride_id} className="badge">
                        {i.ride_emoji} {i.ride_name} x{i.quantity}
                      </span>
                    ))}
                  </div>

                  <div className="caption mt-8">Dibuat {formatTime(order.created_at)}</div>

                  <div className="row mt-16 wrap">
                    <button
                      className="btn btn-success"
                      onClick={() => selesaikan(order.code)}
                      disabled={busy === order.code}
                    >
                      {busy === order.code ? "Memproses..." : "Selesaikan pembayaran"}
                    </button>
                    <Link to={`/pembayaran/${order.code}`} className="btn btn-tinted">
                      Buka halaman pembayaran
                    </Link>
                    <Link to={`/tiket/${order.code}`} className="btn btn-gray">
                      Lihat tiket
                    </Link>
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

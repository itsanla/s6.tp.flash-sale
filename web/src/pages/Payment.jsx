import { useCallback, useEffect, useRef, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { api, countdownText, formatDate, rupiah } from "../api";
import { Loading, StatusBadge } from "../components";
import { useApp } from "../store";

export default function Payment() {
  const { code } = useParams();
  const navigate = useNavigate();
  const { toast } = useApp();

  const [data, setData] = useState(null);
  const [error, setError] = useState("");
  const [seconds, setSeconds] = useState(0);
  const notified = useRef(false);

  const muat = useCallback(async () => {
    try {
      const res = await api.order(code);
      setData(res);
      setSeconds(res.seconds_left);
      return res;
    } catch (e) {
      setError(e.message);
      return null;
    }
  }, [code]);

  useEffect(() => {
    muat();
  }, [muat]);

  // Status pembayaran dipantau berkala. Begitu order lunas, pengunjung diarahkan ke
  // halaman tiket setelah worker selesai menerbitkan tiketnya lewat antrean.
  useEffect(() => {
    const timer = setInterval(async () => {
      const res = await muat();
      if (res && res.order.status === "PAID" && !notified.current) {
        notified.current = true;
        toast("Pembayaran diterima, tiket sedang diterbitkan");
        setTimeout(() => navigate(`/tiket/${code}`), 1200);
      }
    }, 2500);
    return () => clearInterval(timer);
  }, [muat, code, navigate, toast]);

  useEffect(() => {
    const t = setInterval(() => setSeconds((s) => (s > 0 ? s - 1 : 0)), 1000);
    return () => clearInterval(t);
  }, []);

  const batalkan = async () => {
    try {
      await api.cancelOrder(code);
      toast("Pesanan dibatalkan");
      muat();
    } catch (e) {
      toast(e.message, "err");
    }
  };

  if (error) {
    return (
      <div className="container page">
        <div className="card text-center">
          <div className="empty-emoji">🔎</div>
          <div className="title-sm">{error}</div>
          <Link to="/wahana" className="btn btn-tinted mt-16">Kembali ke katalog</Link>
        </div>
      </div>
    );
  }
  if (!data) return <div className="container page"><Loading text="Menyiapkan pembayaran" /></div>;

  const { order } = data;
  const pending = order.status === "PENDING";

  return (
    <div className="container page">
      <div className="text-center mb-16">
        <h1 className="title-lg">Pembayaran</h1>
        <div className="subtitle">
          Kode pesanan <span className="mono">{order.code}</span>
        </div>
      </div>

      {pending ? (
        <div className="qris-box">
          <div className="badge badge-orange mb-16">Menunggu pembayaran</div>
          <div className="title" style={{ marginBottom: 4 }}>{rupiah(order.total_amount)}</div>
          <div className="caption">Pindai kode berikut dengan aplikasi pembayaran Anda</div>

          {data.qris_image && (
            <img className="qris-img mt-16" src={data.qris_image} alt="Kode QRIS pembayaran" />
          )}

          <div className="mt-16">
            <div className="caption">Selesaikan sebelum</div>
            <div className={`countdown ${seconds < 60 ? "warn" : ""}`}>{countdownText(seconds)}</div>
          </div>

          <div className="note mt-16" style={{ textAlign: "left" }}>
            Kode QRIS ini dibuat untuk keperluan demonstrasi sehingga tidak dapat diproses oleh
            aplikasi pembayaran sungguhan. Untuk menyelesaikan pembayaran, buka halaman
            <Link to="/test/qris-list" style={{ color: "var(--blue)", fontWeight: 650 }}> Uji QRIS </Link>
            lalu tekan tombol selesaikan pembayaran pada pesanan ini.
          </div>

          <div className="stack mt-16" style={{ gap: 10 }}>
            <Link to="/test/qris-list" className="btn btn-primary btn-block">
              Buka halaman uji pembayaran
            </Link>
            <button className="btn btn-danger btn-block" onClick={batalkan}>
              Batalkan pesanan
            </button>
          </div>
        </div>
      ) : (
        <div className="qris-box">
          <div className="empty-emoji">{order.status === "PAID" ? "🎉" : "⌛"}</div>
          <div className="title-sm mb-16">
            {order.status === "PAID"
              ? "Pembayaran berhasil"
              : order.status === "EXPIRED"
              ? "Batas waktu pembayaran habis"
              : "Pesanan dibatalkan"}
          </div>
          <StatusBadge status={order.status} />
          {order.status === "PAID" ? (
            <Link to={`/tiket/${order.code}`} className="btn btn-success btn-block mt-24">
              Lihat tiket saya
            </Link>
          ) : (
            <Link to="/wahana" className="btn btn-primary btn-block mt-24">
              Pesan tiket baru
            </Link>
          )}
        </div>
      )}

      <div className="card mt-24" style={{ maxWidth: 560, margin: "24px auto 0" }}>
        <div className="title-sm mb-16">Rincian pesanan</div>
        <div className="row-between">
          <span className="caption">Pemesan</span>
          <span style={{ fontWeight: 600 }}>{order.customer_name}</span>
        </div>
        <div className="row-between mt-8">
          <span className="caption">Tanggal kunjungan</span>
          <span style={{ fontWeight: 600 }}>{formatDate(order.visit_date)}</span>
        </div>
        <div className="divider" />
        {order.items.map((i) => (
          <div key={i.ride_id} className="row-between" style={{ marginBottom: 8 }}>
            <span className="caption">{i.ride_emoji} {i.ride_name} x{i.quantity}</span>
            <span style={{ fontWeight: 600 }}>{rupiah(i.subtotal)}</span>
          </div>
        ))}
        <div className="divider" />
        <div className="row-between">
          <span className="label">Total</span>
          <span className="title-sm" style={{ color: "var(--blue)" }}>{rupiah(order.total_amount)}</span>
        </div>
      </div>
    </div>
  );
}

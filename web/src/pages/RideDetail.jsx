import { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { api, CATEGORY_LABELS, formatDate, rupiah, todayISO } from "../api";
import { Loading, RideThumb, Stepper, ThrillBadge } from "../components";
import { useApp } from "../store";

export default function RideDetail() {
  const { slug } = useParams();
  const navigate = useNavigate();
  const { cart, setVisitDate, addItem, toast } = useApp();

  const [ride, setRide] = useState(null);
  const [error, setError] = useState("");
  const [qty, setQty] = useState(1);

  const date = cart.visitDate || todayISO();

  useEffect(() => {
    setRide(null);
    api
      .ride(slug, date)
      .then(setRide)
      .catch((e) => setError(e.message));
  }, [slug, date]);

  if (error) {
    return (
      <div className="container page">
        <div className="card text-center">
          <div className="empty-emoji">🎪</div>
          <div className="title-sm">{error}</div>
          <Link to="/wahana" className="btn btn-tinted mt-16">Kembali ke katalog</Link>
        </div>
      </div>
    );
  }
  if (!ride) return <div className="container page"><Loading text="Memuat detail wahana" /></div>;

  const habis = ride.available <= 0;
  const maxQty = Math.min(20, Math.max(1, ride.available));

  const tambahKeKeranjang = () => {
    addItem(ride, qty);
    toast(`${qty} tiket ${ride.name} masuk keranjang`);
  };

  const beliLangsung = () => {
    addItem(ride, qty);
    navigate("/keranjang");
  };

  return (
    <div className="container page">
      <Link to="/wahana" className="caption">← Kembali ke katalog</Link>

      <div className="mt-16 detail-layout" style={{ display: "grid", gridTemplateColumns: "1.2fr 1fr", gap: 22, alignItems: "start" }}>
        <div>
          <RideThumb ride={ride} className="ride-hero" />

          <div className="mt-24">
            <div className="row wrap" style={{ gap: 8 }}>
              <span className="badge badge-blue">
                Zona {CATEGORY_LABELS[ride.category] || ride.category}
              </span>
              <ThrillBadge level={ride.thrill_level} />
              {habis ? (
                <span className="badge badge-red">Kuota tanggal ini habis</span>
              ) : (
                <span className="badge badge-green">Sisa kuota {ride.available}</span>
              )}
            </div>
            <h1 className="title-lg mt-8">{ride.name}</h1>
            <p className="subtitle">{ride.tagline}</p>
          </div>

          <div className="chips mt-24">
            <div className="chip">⏱ Durasi <b>{ride.duration_min} menit</b></div>
            <div className="chip">
              📏 Tinggi minimum <b>{ride.min_height_cm > 0 ? `${ride.min_height_cm} cm` : "bebas"}</b>
            </div>
            <div className="chip">🎟 Kuota harian <b>{ride.daily_quota}</b></div>
          </div>

          <div className="card mt-24">
            <div className="title-sm mb-16">Tentang wahana ini</div>
            <p style={{ color: "var(--text-2)", lineHeight: 1.7 }}>{ride.description}</p>
          </div>
        </div>

        <div className="card" style={{ position: "sticky", top: 78 }}>
          <div className="caption">Harga per tiket</div>
          <div className="title-lg" style={{ color: "var(--blue)" }}>{rupiah(ride.price)}</div>

          <div className="divider" />

          <div className="field">
            <label className="label" htmlFor="tgl">Tanggal kunjungan</label>
            <input
              id="tgl"
              type="date"
              className="input"
              min={todayISO()}
              value={date}
              onChange={(e) => setVisitDate(e.target.value)}
            />
            <div className="caption">{formatDate(date)}</div>
          </div>

          <div className="row-between mt-16">
            <span className="label">Jumlah tiket</span>
            <Stepper value={qty} onChange={setQty} min={1} max={maxQty} />
          </div>

          <div className="row-between mt-16">
            <span className="label">Total</span>
            <span className="price">{rupiah(ride.price * qty)}</span>
          </div>

          <div className="stack mt-24" style={{ gap: 10 }}>
            <button className="btn btn-primary btn-block" onClick={beliLangsung} disabled={habis}>
              Beli sekarang
            </button>
            <button className="btn btn-tinted btn-block" onClick={tambahKeKeranjang} disabled={habis}>
              Masukkan keranjang
            </button>
          </div>

          {habis && (
            <div className="note mt-16">
              Kuota wahana ini pada tanggal tersebut sudah habis. Silakan pilih tanggal kunjungan lain.
            </div>
          )}
        </div>
      </div>

      <style>{`
        @media (max-width: 900px) {
          .detail-layout { grid-template-columns: 1fr !important; }
        }
      `}</style>
    </div>
  );
}

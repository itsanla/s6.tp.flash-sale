import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { api } from "../api";
import { RideCard, SkeletonGrid } from "../components";

const CATEGORY_ART = {
  ekstrem: { emoji: "🎢", desc: "Adrenalin tanpa kompromi" },
  keluarga: { emoji: "🎡", desc: "Seru untuk semua umur" },
  anak: { emoji: "🎈", desc: "Aman dan ramah balita" },
  air: { emoji: "🌊", desc: "Basah basahan sepuasnya" },
  petualangan: { emoji: "🪂", desc: "Uji nyali di alam terbuka" },
  indoor: { emoji: "🎬", desc: "Sejuk dan bebas cuaca" },
};

export default function Home() {
  const [rides, setRides] = useState(null);
  const [categories, setCategories] = useState([]);

  useEffect(() => {
    api.rides().then(setRides).catch(() => setRides([]));
    api.categories().then(setCategories).catch(() => setCategories([]));
  }, []);

  const populer = (rides || []).slice(0, 8);

  return (
    <div className="container page">
      <section className="hero">
        <div className="hero-content">
          <div className="hero-badges">
            <span className="hero-badge">32 wahana</span>
            <span className="hero-badge">Bayar pakai QRIS</span>
            <span className="hero-badge">Tiket langsung terbit</span>
          </div>
          <h1>Satu hari penuh keseruan menanti Anda</h1>
          <p>
            Pilih wahana favorit, tentukan tanggal kunjungan, lalu bayar dengan QRIS.
            Tiket elektronik Anda terbit otomatis dan siap dipindai di gerbang wahana.
          </p>
          <div className="row wrap">
            <Link to="/wahana" className="btn btn-lg" style={{ background: "#fff", color: "var(--blue)" }}>
              Jelajahi Wahana
            </Link>
            <Link
              to="/tiket"
              className="btn btn-lg"
              style={{ background: "rgba(255,255,255,0.18)", color: "#fff", border: "1px solid rgba(255,255,255,0.3)" }}
            >
              Cek Tiket Saya
            </Link>
          </div>
        </div>
      </section>

      <section className="mt-32">
        <div className="section-head">
          <div>
            <h2 className="title">Kategori wahana</h2>
            <div className="subtitle">Enam zona berbeda di dalam satu taman</div>
          </div>
        </div>
        <div className="ride-grid">
          {categories.map((c) => {
            const art = CATEGORY_ART[c.slug] || { emoji: "🎪", desc: "" };
            return (
              <Link key={c.slug} to={`/wahana?kategori=${c.slug}`} className="ride-card">
                <div className={`ride-thumb cat-${c.slug}`} style={{ height: 104 }}>
                  <span style={{ position: "relative", zIndex: 1 }}>{art.emoji}</span>
                </div>
                <div className="ride-body">
                  <div className="ride-name">{c.label}</div>
                  <div className="caption">{art.desc}</div>
                  <div className="ride-foot">
                    <span className="badge badge-blue">{c.count} wahana</span>
                    <span className="caption">Lihat semua →</span>
                  </div>
                </div>
              </Link>
            );
          })}
        </div>
      </section>

      <section className="mt-32">
        <div className="section-head">
          <div>
            <h2 className="title">Wahana paling memacu adrenalin</h2>
            <div className="subtitle">Pilihan favorit pengunjung taman</div>
          </div>
          <Link to="/wahana" className="btn btn-tinted btn-sm">
            Semua wahana
          </Link>
        </div>
        {rides === null ? <SkeletonGrid count={8} /> : (
          <div className="ride-grid">
            {populer.map((r) => (
              <RideCard key={r.id} ride={r} />
            ))}
          </div>
        )}
      </section>

      <section className="mt-32">
        <div className="section-head">
          <h2 className="title">Cara memesan tiket</h2>
        </div>
        <div className="ride-grid">
          {[
            { n: "1", t: "Pilih wahana", d: "Tentukan wahana dan tanggal kunjungan, lalu masukkan ke keranjang.", e: "🎠" },
            { n: "2", t: "Bayar dengan QRIS", d: "Kode QRIS terbit otomatis. Selesaikan pembayaran sebelum batas waktu.", e: "📱" },
            { n: "3", t: "Tiket terbit", d: "Tiket elektronik diterbitkan lewat antrean dan langsung muncul di halaman Tiket Saya.", e: "🎟️" },
          ].map((s) => (
            <div key={s.n} className="card">
              <div className="row">
                <div className="list-icon" style={{ background: "var(--gray6)" }}>{s.e}</div>
                <div>
                  <div className="caption">Langkah {s.n}</div>
                  <div className="title-sm">{s.t}</div>
                </div>
              </div>
              <div className="subtitle mt-8">{s.d}</div>
            </div>
          ))}
        </div>
      </section>
    </div>
  );
}

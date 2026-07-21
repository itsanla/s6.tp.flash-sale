import { Link, NavLink } from "react-router-dom";
import { rupiah, STATUS_META } from "./api";
import { useApp } from "./store";

// Kumpulan komponen antarmuka yang dipakai berulang di banyak halaman.

export function Navbar() {
  const { totals } = useApp();
  const link = ({ isActive }) => "nav-link" + (isActive ? " active" : "");

  return (
    <nav className="nav">
      <div className="container nav-inner">
        <Link to="/" className="brand">
          <span className="brand-mark">🎡</span>
          <span>Taman Wahana Nusantara</span>
        </Link>

        <div className="nav-links">
          <NavLink to="/" className={link} end>
            Beranda
          </NavLink>
          <NavLink to="/wahana" className={link}>
            Wahana
          </NavLink>
          <NavLink to="/tiket" className={link}>
            Tiket Saya
          </NavLink>
          <NavLink to="/test/qris-list" className={link}>
            Uji QRIS
          </NavLink>
          <Link to="/keranjang" className="cart-btn" aria-label="Keranjang">
            🛒
            {totals.count > 0 && <span className="cart-count">{totals.count}</span>}
          </Link>
        </div>
      </div>
    </nav>
  );
}

export function Footer() {
  return (
    <footer className="footer">
      <div className="container">
        <div>Taman Wahana Nusantara - proyek demonstrasi Redis, RabbitMQ, dan SQLite</div>
        <div className="mt-8">
          Pembayaran QRIS pada aplikasi ini bersifat simulasi dan tidak memproses uang sungguhan.
        </div>
      </div>
    </footer>
  );
}

export function Toasts() {
  const { toasts } = useApp();
  return (
    <div className="toast-wrap">
      {toasts.map((t) => (
        <div key={t.id} className={`toast ${t.kind}`}>
          <span>{t.kind === "err" ? "⚠️" : "✅"}</span>
          <span>{t.message}</span>
        </div>
      ))}
    </div>
  );
}

export function RideCard({ ride }) {
  const habis = ride.available !== undefined && ride.available <= 0;
  return (
    <Link to={`/wahana/${ride.slug}`} className="ride-card">
      <div className={`ride-thumb cat-${ride.category}`}>
        <span style={{ position: "relative", zIndex: 1 }}>{ride.emoji}</span>
      </div>
      <div className="ride-body">
        <div className="row" style={{ gap: 6, flexWrap: "wrap" }}>
          <ThrillBadge level={ride.thrill_level} />
          {habis && <span className="badge badge-red">Kuota habis</span>}
        </div>
        <div className="ride-name">{ride.name}</div>
        <div className="ride-tagline">{ride.tagline}</div>
        <div className="ride-foot">
          <div className="price">
            {rupiah(ride.price)} <small>/ tiket</small>
          </div>
          <span className="badge">⏱ {ride.duration_min} mnt</span>
        </div>
      </div>
    </Link>
  );
}

export function ThrillBadge({ level }) {
  const meta = {
    1: { label: "Santai", cls: "badge-green" },
    2: { label: "Ringan", cls: "badge-blue" },
    3: { label: "Menantang", cls: "badge-purple" },
    4: { label: "Memacu Adrenalin", cls: "badge-orange" },
    5: { label: "Ekstrem", cls: "badge-red" },
  }[level] || { label: "Santai", cls: "badge-green" };
  return <span className={`badge ${meta.cls}`}>{meta.label}</span>;
}

export function StatusBadge({ status }) {
  const meta = STATUS_META[status] || { label: status, cls: "badge-gray" };
  return <span className={`badge ${meta.cls}`}>{meta.label}</span>;
}

export function Stepper({ value, onChange, min = 1, max = 20 }) {
  return (
    <div className="stepper">
      <button onClick={() => onChange(value - 1)} disabled={value <= min} aria-label="Kurangi">
        −
      </button>
      <span className="value">{value}</span>
      <button onClick={() => onChange(value + 1)} disabled={value >= max} aria-label="Tambah">
        +
      </button>
    </div>
  );
}

export function Empty({ emoji = "🎪", title, desc, action }) {
  return (
    <div className="empty">
      <div className="empty-emoji">{emoji}</div>
      <div className="title-sm" style={{ color: "var(--text)" }}>
        {title}
      </div>
      {desc && <div className="subtitle mt-8">{desc}</div>}
      {action && <div className="mt-16">{action}</div>}
    </div>
  );
}

export function Loading({ text = "Memuat data" }) {
  return (
    <div className="empty">
      <div className="row" style={{ justifyContent: "center" }}>
        <div className="spinner" />
        <span className="subtitle">{text}</span>
      </div>
    </div>
  );
}

export function SkeletonGrid({ count = 8 }) {
  return (
    <div className="ride-grid">
      {Array.from({ length: count }).map((_, i) => (
        <div key={i} className="ride-card">
          <div className="skeleton" style={{ height: 132, borderRadius: 0 }} />
          <div className="ride-body">
            <div className="skeleton" style={{ height: 14, width: "45%" }} />
            <div className="skeleton" style={{ height: 16, width: "80%" }} />
            <div className="skeleton" style={{ height: 30, width: "100%" }} />
          </div>
        </div>
      ))}
    </div>
  );
}

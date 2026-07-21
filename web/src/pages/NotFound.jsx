import { Link } from "react-router-dom";

export default function NotFound() {
  return (
    <div className="container page">
      <div className="card text-center" style={{ maxWidth: 460, margin: "48px auto" }}>
        <div className="empty-emoji">🎪</div>
        <h1 className="title">Halaman tidak ditemukan</h1>
        <p className="subtitle mt-8">
          Halaman yang Anda tuju mungkin sudah dipindahkan atau tautannya keliru.
        </p>
        <Link to="/" className="btn btn-primary mt-24">Kembali ke beranda</Link>
      </div>
    </div>
  );
}

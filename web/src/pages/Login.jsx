import { useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { api } from "../api";
import { useApp } from "../store";

export default function Login() {
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const { signIn, toast } = useApp();

  const [form, setForm] = useState({ email: "", password: "" });
  const [busy, setBusy] = useState(false);

  // Setelah masuk, pengunjung dikembalikan ke halaman yang tadi ingin dibuka.
  const lanjut = params.get("lanjut") || "/profil";
  const set = (k) => (e) => setForm((f) => ({ ...f, [k]: e.target.value }));

  const masuk = async (e) => {
    e.preventDefault();
    setBusy(true);
    try {
      const data = await api.loginUser(form.email, form.password);
      signIn(data.token, data.user);
      toast(`Selamat datang kembali, ${data.user.name}`);
      navigate(lanjut);
    } catch (err) {
      toast(err.message, "err");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="container page">
      <div className="auth-card">
        <div className="text-center mb-16">
          <div className="auth-mark">🎡</div>
          <h1 className="title">Masuk ke akun</h1>
          <p className="subtitle mt-8">
            Simpan riwayat pemesanan dan buka tiket Anda kapan saja
          </p>
        </div>

        <form onSubmit={masuk} className="stack">
          <div className="field">
            <label className="label" htmlFor="email">Email</label>
            <input
              id="email"
              type="email"
              className="input"
              placeholder="nama@email.com"
              autoComplete="email"
              required
              value={form.email}
              onChange={set("email")}
            />
          </div>
          <div className="field">
            <label className="label" htmlFor="password">Password</label>
            <input
              id="password"
              type="password"
              className="input"
              placeholder="Masukkan password"
              autoComplete="current-password"
              required
              value={form.password}
              onChange={set("password")}
            />
          </div>
          <button className="btn btn-primary btn-block btn-lg" type="submit" disabled={busy}>
            {busy ? "Memproses..." : "Masuk"}
          </button>
        </form>

        <div className="divider" />
        <div className="text-center caption">
          Belum punya akun?{" "}
          <Link to="/daftar" style={{ color: "var(--blue)", fontWeight: 650 }}>
            Daftar sekarang
          </Link>
        </div>
      </div>
    </div>
  );
}

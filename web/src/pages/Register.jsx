import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { api } from "../api";
import { useApp } from "../store";

export default function Register() {
  const navigate = useNavigate();
  const { signIn, toast } = useApp();

  const [form, setForm] = useState({ name: "", email: "", phone: "", password: "" });
  const [busy, setBusy] = useState(false);

  const set = (k) => (e) => setForm((f) => ({ ...f, [k]: e.target.value }));

  const daftar = async (e) => {
    e.preventDefault();
    if (form.password.length < 6) {
      toast("Password minimal 6 karakter", "err");
      return;
    }
    setBusy(true);
    try {
      const data = await api.register(form);
      signIn(data.token, data.user);
      toast(`Akun dibuat. Selamat datang, ${data.user.name}`);
      navigate("/profil");
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
          <div className="auth-mark">🎟️</div>
          <h1 className="title">Buat akun</h1>
          <p className="subtitle mt-8">
            Satu akun untuk memesan tiket dan menyimpan seluruh riwayat kunjungan
          </p>
        </div>

        <form onSubmit={daftar} className="stack">
          <div className="field">
            <label className="label" htmlFor="nama">Nama lengkap</label>
            <input
              id="nama"
              className="input"
              placeholder="Nama sesuai identitas"
              autoComplete="name"
              required
              value={form.name}
              onChange={set("name")}
            />
          </div>
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
            <label className="label" htmlFor="telepon">Nomor telepon</label>
            <input
              id="telepon"
              className="input"
              placeholder="08xxxxxxxxxx"
              autoComplete="tel"
              value={form.phone}
              onChange={set("phone")}
            />
          </div>
          <div className="field">
            <label className="label" htmlFor="password">Password</label>
            <input
              id="password"
              type="password"
              className="input"
              placeholder="Minimal 6 karakter"
              autoComplete="new-password"
              required
              value={form.password}
              onChange={set("password")}
            />
          </div>
          <button className="btn btn-primary btn-block btn-lg" type="submit" disabled={busy}>
            {busy ? "Memproses..." : "Daftar"}
          </button>
        </form>

        <div className="divider" />
        <div className="text-center caption">
          Sudah punya akun?{" "}
          <Link to="/masuk" style={{ color: "var(--blue)", fontWeight: 650 }}>
            Masuk di sini
          </Link>
        </div>
      </div>
    </div>
  );
}

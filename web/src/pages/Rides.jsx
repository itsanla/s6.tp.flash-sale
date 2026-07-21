import { useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { api, todayISO } from "../api";
import { Empty, RideCard, SkeletonGrid } from "../components";
import { useApp } from "../store";

const SORTS = [
  { key: "populer", label: "Paling memacu" },
  { key: "murah", label: "Harga terendah" },
  { key: "mahal", label: "Harga tertinggi" },
  { key: "nama", label: "Nama A sampai Z" },
];

export default function Rides() {
  const { cart, setVisitDate } = useApp();
  const [params, setParams] = useSearchParams();
  const category = params.get("kategori") || "";

  const [rides, setRides] = useState(null);
  const [categories, setCategories] = useState([]);
  const [keyword, setKeyword] = useState("");
  const [sort, setSort] = useState("populer");

  const date = cart.visitDate || todayISO();

  useEffect(() => {
    api.categories().then(setCategories).catch(() => setCategories([]));
  }, []);

  useEffect(() => {
    setRides(null);
    api
      .rides({ category, date })
      .then(setRides)
      .catch(() => setRides([]));
  }, [category, date]);

  const shown = useMemo(() => {
    let list = [...(rides || [])];
    const q = keyword.trim().toLowerCase();
    if (q) {
      list = list.filter(
        (r) =>
          r.name.toLowerCase().includes(q) ||
          r.tagline.toLowerCase().includes(q) ||
          r.description.toLowerCase().includes(q)
      );
    }
    if (sort === "murah") list.sort((a, b) => a.price - b.price);
    if (sort === "mahal") list.sort((a, b) => b.price - a.price);
    if (sort === "nama") list.sort((a, b) => a.name.localeCompare(b.name));
    return list;
  }, [rides, keyword, sort]);

  const setCategory = (slug) => {
    if (slug) params.set("kategori", slug);
    else params.delete("kategori");
    setParams(params, { replace: true });
  };

  return (
    <div className="container page">
      <div className="section-head">
        <div>
          <h1 className="title-lg">Katalog Wahana</h1>
          <div className="subtitle">
            Sisa kuota ditampilkan sesuai tanggal kunjungan yang Anda pilih
          </div>
        </div>
      </div>

      <div className="card mb-16">
        <div className="form-grid">
          <div className="field">
            <label className="label" htmlFor="cari">Cari wahana</label>
            <input
              id="cari"
              className="input"
              placeholder="Ketik nama wahana, misalnya roller coaster"
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
            />
          </div>
          <div className="form-grid" style={{ gridTemplateColumns: "1fr 1fr" }}>
            <div className="field">
              <label className="label" htmlFor="tanggal">Tanggal kunjungan</label>
              <input
                id="tanggal"
                type="date"
                className="input"
                min={todayISO()}
                value={date}
                onChange={(e) => setVisitDate(e.target.value)}
              />
            </div>
            <div className="field">
              <label className="label" htmlFor="urut">Urutkan</label>
              <select id="urut" className="select" value={sort} onChange={(e) => setSort(e.target.value)}>
                {SORTS.map((s) => (
                  <option key={s.key} value={s.key}>{s.label}</option>
                ))}
              </select>
            </div>
          </div>
        </div>

        <div className="mt-16">
          <div className="segmented">
            <button className={`segment ${!category ? "active" : ""}`} onClick={() => setCategory("")}>
              Semua
            </button>
            {categories.map((c) => (
              <button
                key={c.slug}
                className={`segment ${category === c.slug ? "active" : ""}`}
                onClick={() => setCategory(c.slug)}
              >
                {c.label} ({c.count})
              </button>
            ))}
          </div>
        </div>
      </div>

      {rides === null ? (
        <SkeletonGrid count={9} />
      ) : shown.length === 0 ? (
        <Empty
          emoji="🔍"
          title="Wahana tidak ditemukan"
          desc="Coba ubah kata kunci pencarian atau pilih kategori lain."
        />
      ) : (
        <>
          <div className="caption mb-16">{shown.length} wahana ditemukan</div>
          <div className="ride-grid">
            {shown.map((r) => (
              <RideCard key={r.id} ride={r} />
            ))}
          </div>
        </>
      )}
    </div>
  );
}

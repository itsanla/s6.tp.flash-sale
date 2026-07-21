import { Route, Routes, useLocation } from "react-router-dom";
import { useEffect } from "react";
import { Footer, Navbar, Toasts } from "./components";
import Home from "./pages/Home";
import Rides from "./pages/Rides";
import RideDetail from "./pages/RideDetail";
import Cart from "./pages/Cart";
import Payment from "./pages/Payment";
import Tickets from "./pages/Tickets";
import QrisTest from "./pages/QrisTest";
import Admin from "./pages/Admin";
import NotFound from "./pages/NotFound";

function ScrollToTop() {
  const { pathname } = useLocation();
  useEffect(() => {
    window.scrollTo({ top: 0, behavior: "instant" });
  }, [pathname]);
  return null;
}

export default function App() {
  return (
    <>
      <ScrollToTop />
      <Navbar />
      <main style={{ minHeight: "calc(100vh - 200px)" }}>
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/wahana" element={<Rides />} />
          <Route path="/wahana/:slug" element={<RideDetail />} />
          <Route path="/keranjang" element={<Cart />} />
          <Route path="/pembayaran/:code" element={<Payment />} />
          <Route path="/tiket" element={<Tickets />} />
          <Route path="/tiket/:code" element={<Tickets />} />
          <Route path="/test/qris-list" element={<QrisTest />} />
          <Route path="/admin" element={<Admin />} />
          <Route path="*" element={<NotFound />} />
        </Routes>
      </main>
      <Footer />
      <Toasts />
    </>
  );
}

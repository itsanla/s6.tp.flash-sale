import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { api, clearUserToken, getUserToken, setUserToken, todayISO } from "./api";

// Konteks keranjang belanja dan notifikasi ringkas (toast) dipakai lintas halaman.

const CART_KEY = "wahana_cart";
const AppContext = createContext(null);

function loadCart() {
  try {
    const raw = localStorage.getItem(CART_KEY);
    if (!raw) return { items: [], visitDate: todayISO() };
    const parsed = JSON.parse(raw);
    return {
      items: Array.isArray(parsed.items) ? parsed.items : [],
      visitDate: parsed.visitDate || todayISO(),
    };
  } catch {
    return { items: [], visitDate: todayISO() };
  }
}

export function AppProvider({ children }) {
  const [cart, setCart] = useState(loadCart);
  const [toasts, setToasts] = useState([]);
  const [user, setUser] = useState(null);
  const [authReady, setAuthReady] = useState(false);

  useEffect(() => {
    localStorage.setItem(CART_KEY, JSON.stringify(cart));
  }, [cart]);

  // Sesi dipulihkan saat aplikasi dibuka: bila token tersimpan masih berlaku, profil
  // pengunjung diambil ulang; bila sudah kedaluwarsa, token dibuang diam diam.
  useEffect(() => {
    if (!getUserToken()) {
      setAuthReady(true);
      return;
    }
    api
      .me()
      .then(setUser)
      .catch(() => clearUserToken())
      .finally(() => setAuthReady(true));
  }, []);

  const signIn = useCallback((token, profile) => {
    setUserToken(token);
    setUser(profile);
  }, []);

  const signOut = useCallback(() => {
    clearUserToken();
    setUser(null);
  }, []);

  const toast = useCallback((message, kind = "ok") => {
    const id = Date.now() + Math.random();
    setToasts((prev) => [...prev, { id, message, kind }]);
    setTimeout(() => setToasts((prev) => prev.filter((t) => t.id !== id)), 2800);
  }, []);

  const addItem = useCallback((ride, quantity) => {
    setCart((prev) => {
      const existing = prev.items.find((i) => i.rideId === ride.id);
      const items = existing
        ? prev.items.map((i) =>
            i.rideId === ride.id ? { ...i, quantity: i.quantity + quantity } : i
          )
        : [
            ...prev.items,
            {
              rideId: ride.id,
              slug: ride.slug,
              name: ride.name,
              emoji: ride.emoji,
              category: ride.category,
              price: ride.price,
              quantity,
            },
          ];
      return { ...prev, items };
    });
  }, []);

  const updateQuantity = useCallback((rideId, quantity) => {
    setCart((prev) => ({
      ...prev,
      items:
        quantity <= 0
          ? prev.items.filter((i) => i.rideId !== rideId)
          : prev.items.map((i) => (i.rideId === rideId ? { ...i, quantity } : i)),
    }));
  }, []);

  const removeItem = useCallback((rideId) => {
    setCart((prev) => ({ ...prev, items: prev.items.filter((i) => i.rideId !== rideId) }));
  }, []);

  const clearCart = useCallback(() => {
    setCart((prev) => ({ ...prev, items: [] }));
  }, []);

  const setVisitDate = useCallback((visitDate) => {
    setCart((prev) => ({ ...prev, visitDate }));
  }, []);

  const totals = useMemo(() => {
    const count = cart.items.reduce((a, i) => a + i.quantity, 0);
    const amount = cart.items.reduce((a, i) => a + i.quantity * i.price, 0);
    return { count, amount };
  }, [cart.items]);

  const value = {
    cart,
    totals,
    addItem,
    updateQuantity,
    removeItem,
    clearCart,
    setVisitDate,
    toast,
    toasts,
    user,
    setUser,
    authReady,
    signIn,
    signOut,
  };

  return <AppContext.Provider value={value}>{children}</AppContext.Provider>;
}

export function useApp() {
  const ctx = useContext(AppContext);
  if (!ctx) throw new Error("useApp harus dipakai di dalam AppProvider");
  return ctx;
}

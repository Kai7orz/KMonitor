import { create } from "zustand";

type AuthState = {
  idToken: string | null;
  uid: string | null;
  email: string | null;
  loading: boolean;
  isAuthenticated: boolean;
  setAuth: (p: { idToken?: string | null; uid?: string | null; email?: string | null }) => void;
  clearAuth: () => void;
  setLoading: (v: boolean) => void;
};

export const useAuthStore = create<AuthState>((set) => ({
  idToken: null,
  uid: null,
  email: null,
  loading: true,
  isAuthenticated: false,

  setAuth: (p) =>
    set((state) => {
      const next = {
        idToken: p.idToken !== undefined ? p.idToken : state.idToken,
        uid: p.uid !== undefined ? p.uid : state.uid,
        email: p.email !== undefined ? p.email : state.email,
      };
      return { ...next, isAuthenticated: !!next.idToken && !!next.uid };
    }),

  clearAuth: () =>
    set({ idToken: null, uid: null, email: null, isAuthenticated: false }),

  setLoading: (v) => set({ loading: v }),
}));

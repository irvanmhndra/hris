import { create } from "zustand";
import type { User } from "../types";
interface Session {
  token: string | null;
  user: User | null;
  setSession: (token: string, user: User) => void;
  clear: () => void;
}
export const useSession = create<Session>((set) => ({
  token: sessionStorage.getItem("hris.token"),
  user: null,
  setSession: (token, user) => {
    sessionStorage.setItem("hris.token", token);
    set({ token, user });
  },
  clear: () => {
    sessionStorage.removeItem("hris.token");
    set({ token: null, user: null });
  },
}));

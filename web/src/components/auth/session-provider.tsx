"use client";
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from "react";
import { api, ApiError, message, type User } from "@/lib/api";

type Session = {
  user: User | null;
  loading: boolean;
  error: string;
  refresh: () => Promise<User | null>;
  setUser: (user: User | null) => void;
  logout: () => Promise<void>;
};
const Context = createContext<Session | null>(null);
export function SessionProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const refresh = useCallback(async () => {
    try {
      const { data } = await api<{ data: User }>("/account");
      setUser(data);
      setError("");
      return data;
    } catch (error) {
      setUser(null);
      if (!(error instanceof ApiError && error.status === 401))
        setError(message(error));
      return null;
    } finally {
      setLoading(false);
    }
  }, []);
  useEffect(() => {
    void refresh();
  }, [refresh]);
  async function logout() {
    await api("/logout", { method: "POST" });
    setUser(null);
  }
  return (
    <Context.Provider
      value={{ user, loading, error, refresh, setUser, logout }}
    >
      {children}
    </Context.Provider>
  );
}
export function useSession() {
  const value = useContext(Context);
  if (!value) throw new Error("اطلاعات نشست کاربری در دسترس نیست.");
  return value;
}

import { useMutation } from "@tanstack/react-query";
import { ArrowRight, Leaf } from "lucide-react";
import { client, ErrorBox } from "../components/common";
import { api } from "../services/api";
import { useSession } from "../stores/session";
import type { User } from "../types";
export function Login({ mode }: { mode: "admin" | "employee" }) {
  const setSession = useSession((s) => s.setSession);
  const mutation = useMutation({
    mutationFn: async (v: Record<string, FormDataEntryValue>) => {
      const data = await api<{ token: string; user: User }>(
        "/auth/login",
        "POST",
        v,
      );
      if (data.user.role !== mode) {
        await fetch("/api/v1/auth/logout", {
          method: "POST",
          headers: { Authorization: `Bearer ${data.token}` },
        });
        throw new Error(
          `Gunakan akun ${mode === "admin" ? "admin" : "karyawan"} untuk portal ini`,
        );
      }
      return data;
    },
    onSuccess: (d) => {
      client.clear();
      setSession(d.token, d.user);
    },
  });
  return (
    <div className="login">
      <div className="login-story">
        <div className="brand">
          <span className="brand-icon">
            <Leaf />
          </span>
          people<span className="brand-dot">.</span>
        </div>
        <div>
          <div className="eyebrow">PEOPLE FIRST. ALWAYS.</div>
          <h1>
            Tim yang hebat.
            <br />
            Dimulai dari
            <br />
            <em>pengelolaan yang baik.</em>
          </h1>
          <p>
            Satu ruang untuk orang-orang yang membangun
            <br />
            masa depan perusahaan Anda.
          </p>
        </div>
        <small>People HRIS · Ruang untuk bertumbuh bersama</small>
      </div>
      <div className="login-form">
        <span className="pill">
          {mode === "admin" ? "WORKSPACE ADMIN" : "PORTAL KARYAWAN"}
        </span>
        <h2>Selamat datang kembali.</h2>
        <p>Masuk untuk memulai hari yang produktif.</p>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            mutation.mutate(Object.fromEntries(new FormData(e.currentTarget)));
          }}
        >
          <label>
            Kode perusahaan
            <input
              name="company"
              defaultValue="demo"
              required
              autoComplete="organization"
            />
          </label>
          <label>
            Email
            <input
              name="email"
              type="email"
              placeholder={
                mode === "admin" ? "admin@demo.hris" : "employee@demo.hris"
              }
              required
              autoComplete="username"
            />
          </label>
          <label>
            Password
            <input
              name="password"
              type="password"
              required
              autoComplete="current-password"
            />
          </label>
          <ErrorBox error={mutation.error} />
          <button className="primary wide" disabled={mutation.isPending}>
            {mutation.isPending ? "Memproses…" : "Masuk ke workspace"}
            <ArrowRight size={18} />
          </button>
        </form>
        <small>Hubungi admin HR jika Anda belum memiliki akun.</small>
      </div>
    </div>
  );
}

import { useMutation, useQuery } from "@tanstack/react-query";
import { ArrowRight, Leaf } from "lucide-react";
import { useState, type ReactElement } from "react";
import { useLocation, useNavigate } from "react-router";
import { client, ErrorBox } from "../components/common";
import { api } from "../services/api";
import { useSession } from "../stores/session";
import type { User } from "../types";
type View = "login" | "forgot" | "register" | "reset";

export function Login({ mode }: { mode: "admin" | "employee" }) {
  const location = useLocation();
  const navigate = useNavigate();
  const resetToken =
    location.pathname === "/reset-password"
      ? new URLSearchParams(location.search).get("token") || ""
      : "";
  const [view, setView] = useState<View>(resetToken ? "reset" : "login");
  const [notice, setNotice] = useState("");
  const config = useQuery({
    queryKey: ["/auth/config"],
    queryFn: () => api<{ signup_enabled: boolean }>("/auth/config"),
    enabled: mode === "admin",
  });
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
  const forgot = useMutation({
    mutationFn: (v: Record<string, FormDataEntryValue>) =>
      api("/auth/forgot-password", "POST", v),
    onSuccess: () => {
      setNotice(
        "Jika akun ditemukan, tautan reset password telah dikirim ke email tersebut. Tautan berlaku 1 jam.",
      );
      setView("login");
    },
  });
  const reset = useMutation({
    mutationFn: (password: string) =>
      api("/auth/reset-password", "POST", { token: resetToken, password }),
    onSuccess: () => {
      setNotice(
        "Password berhasil diubah. Silakan masuk dengan password baru.",
      );
      setView("login");
      navigate("/", { replace: true });
    },
  });
  const register = useMutation({
    mutationFn: (v: Record<string, FormDataEntryValue>) =>
      api<{ token: string; user: User }>("/auth/register", "POST", v),
    onSuccess: (d) => {
      client.clear();
      navigate("/", { replace: true });
      setSession(d.token, d.user);
    },
  });
  const forms: Record<View, ReactElement> = {
    login: (
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
        <button
          type="button"
          className="text-button"
          onClick={() => {
            forgot.reset();
            setNotice("");
            setView("forgot");
          }}
        >
          Lupa password?
        </button>
      </form>
    ),
    forgot: (
      <form
        onSubmit={(e) => {
          e.preventDefault();
          forgot.mutate(Object.fromEntries(new FormData(e.currentTarget)));
        }}
      >
        <label>
          Kode perusahaan
          <input name="company" required autoComplete="organization" />
        </label>
        <label>
          Email akun
          <input name="email" type="email" required autoComplete="username" />
        </label>
        <ErrorBox error={forgot.error} />
        <button className="primary wide" disabled={forgot.isPending}>
          Kirim tautan reset
        </button>
        <button
          type="button"
          className="text-button"
          onClick={() => setView("login")}
        >
          Kembali ke login
        </button>
      </form>
    ),
    reset: (
      <form
        onSubmit={(e) => {
          e.preventDefault();
          const d = new FormData(e.currentTarget);
          if (d.get("password") !== d.get("confirm")) return;
          reset.mutate(String(d.get("password")));
        }}
      >
        <label>
          Password baru
          <input
            name="password"
            type="password"
            minLength={12}
            maxLength={72}
            required
            autoComplete="new-password"
          />
        </label>
        <label>
          Ulangi password baru
          <input
            name="confirm"
            type="password"
            minLength={12}
            maxLength={72}
            required
            autoComplete="new-password"
            onInput={(e) => {
              const form = e.currentTarget.form!;
              e.currentTarget.setCustomValidity(
                e.currentTarget.value ===
                  (form.elements.namedItem("password") as HTMLInputElement)
                    .value
                  ? ""
                  : "Password tidak sama",
              );
            }}
          />
        </label>
        <ErrorBox error={reset.error} />
        <button className="primary wide" disabled={reset.isPending}>
          Simpan password baru
        </button>
      </form>
    ),
    register: (
      <form
        onSubmit={(e) => {
          e.preventDefault();
          register.mutate(Object.fromEntries(new FormData(e.currentTarget)));
        }}
      >
        <label>
          Nama perusahaan
          <input name="company_name" required minLength={2} maxLength={120} />
        </label>
        <label>
          Kode perusahaan (untuk login)
          <input
            name="company_slug"
            required
            pattern="[a-z0-9][a-z0-9\-]{1,38}[a-z0-9]"
            title="3–40 karakter: huruf kecil, angka, atau tanda hubung"
            placeholder="contoh: nusa-karya"
          />
        </label>
        <label>
          Nama Anda
          <input name="name" required minLength={2} maxLength={120} />
        </label>
        <label>
          Email kerja
          <input name="email" type="email" required autoComplete="username" />
        </label>
        <label>
          Password
          <input
            name="password"
            type="password"
            minLength={12}
            maxLength={72}
            required
            autoComplete="new-password"
          />
        </label>
        <ErrorBox error={register.error} />
        <button className="primary wide" disabled={register.isPending}>
          Buat workspace
        </button>
        <button
          type="button"
          className="text-button"
          onClick={() => setView("login")}
        >
          Sudah punya akun? Masuk
        </button>
      </form>
    ),
  };
  const titles: Record<View, [string, string]> = {
    login: [
      "Selamat datang kembali.",
      "Masuk untuk memulai hari yang produktif.",
    ],
    forgot: ["Lupa password?", "Kami kirim tautan reset ke email akun Anda."],
    reset: [
      "Buat password baru.",
      "Minimal 12 karakter. Semua sesi lama akan keluar.",
    ],
    register: [
      "Daftarkan perusahaan.",
      "Workspace baru dengan Anda sebagai admin.",
    ],
  };
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
        <h2>{titles[view][0]}</h2>
        <p>{titles[view][1]}</p>
        {notice && view === "login" && <p className="notice">{notice}</p>}
        <div key={view}>{forms[view]}</div>
        {mode === "admin" && config.data?.signup_enabled && view === "login" ? (
          <small>
            Perusahaan baru?{" "}
            <button
              type="button"
              className="text-button"
              onClick={() => {
                register.reset();
                setView("register");
              }}
            >
              Daftarkan workspace
            </button>
          </small>
        ) : (
          <small>Hubungi admin HR jika Anda belum memiliki akun.</small>
        )}
      </div>
    </div>
  );
}

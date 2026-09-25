import { useQuery } from "@tanstack/react-query";
import {
  ArrowUpRight,
  CalendarDays,
  ChevronRight,
  Leaf,
  LogOut,
  Menu,
  type LucideIcon,
} from "lucide-react";
import { Suspense, useEffect, useState, type ReactNode } from "react";
import { Navigate, NavLink, Route, Routes, useLocation } from "react-router";
import {
  client,
  date,
  ErrorBox,
  initials,
  Loading,
  today,
} from "../components/common";
import { Login } from "../pages/Login";
import { api } from "../services/api";
import { useSession } from "../stores/session";
import type { User } from "../types";
export type Mode = "admin" | "employee";
export type NavItem = readonly [path: string, label: string, icon: LucideIcon];
// Shell is the signed-in frame shared by both portals. Each portal passes its
// own navigation and routes, so it bundles only the pages it can reach.
export function Shell({
  mode,
  links,
  children,
}: {
  mode: Mode;
  links: readonly NavItem[];
  children: ReactNode;
}) {
  const { token, user, setSession, clear } = useSession();
  const me = useQuery({
    queryKey: ["/auth/me", token],
    queryFn: () => api<User>("/auth/me"),
    enabled: !!token,
  });
  const [open, setOpen] = useState(false);
  const [logoutError, setLogoutError] = useState<unknown>(null);
  const location = useLocation();
  useEffect(() => {
    if (me.data && token) setSession(token, me.data);
  }, [me.data, token, setSession]);
  useEffect(() => setOpen(false), [location.pathname]);
  if (!token) return <Login mode={mode} />;
  if (me.isLoading || !user)
    return me.error ? (
      <>
        <ErrorBox error={me.error} />
        <button onClick={clear}>Kembali ke login</button>
      </>
    ) : (
      <Loading />
    );
  if (user.role !== mode)
    return (
      <div className="empty">
        Akun ini tidak memiliki akses portal ini.
        <button onClick={clear}>Ganti akun</button>
      </div>
    );
  const admin = mode === "admin";
  const current =
    links.find((x) => x[0] === location.pathname)?.[1] || "Workspace";
  return (
    <div className="app">
      {open && (
        <button
          className="mobile-backdrop"
          aria-label="Tutup menu"
          onClick={() => setOpen(false)}
        />
      )}
      <aside className={open ? "sidebar open" : "sidebar"}>
        <div className="brand">
          <span className="brand-icon">
            <Leaf size={23} />
          </span>
          people<span className="brand-dot">.</span>
        </div>
        <div className="workspace">
          <div className="workspace-icon">{initials(user.company_name)}</div>
          <div>
            <strong>{user.company_name}</strong>
            <small>Company workspace</small>
          </div>
        </div>
        <div className="nav-label">
          {admin ? "WORKSPACE" : "PERSONAL WORKSPACE"}
        </div>
        <nav>
          {links.map(([path, label, Icon]) => (
            <NavLink key={path} end to={path}>
              <Icon size={19} />
              {label}
              {path === location.pathname && <span className="nav-dot" />}
            </NavLink>
          ))}
        </nav>
        <div className="sidebar-note">
          <div className="mini-leaf">
            <Leaf size={21} />
          </div>
          <strong>Great people, great things.</strong>
          <p>Berikan ruang bagi setiap orang untuk berkembang.</p>
          <span>
            Made for your people <ArrowUpRight size={14} />
          </span>
        </div>
        <div className="profile">
          <span className="avatar">{initials(user.name)}</span>
          <div>
            <strong>{user.name}</strong>
            <small>{admin ? "HR Administrator" : "Karyawan"}</small>
          </div>
          <button
            className="icon-button"
            aria-label="Keluar"
            onClick={async () => {
              try {
                await api("/auth/logout", "POST");
                clear();
                client.clear();
              } catch (e) {
                setLogoutError(e);
              }
            }}
          >
            <LogOut size={17} />
          </button>
        </div>
      </aside>
      <div className="main">
        <header className="topbar">
          <button
            className="icon-button mobile-menu"
            aria-label="Buka menu"
            onClick={() => setOpen(!open)}
          >
            <Menu />
          </button>
          <div>
            Workspace <ChevronRight size={14} />
            <strong>{current}</strong>
          </div>
          <span className="top-date">
            <CalendarDays size={16} />
            {date(today())}
            <span className="timezone">WIB</span>
          </span>
        </header>
        <main>
          <ErrorBox error={logoutError} />
          <Suspense fallback={<Loading />}>
            <Routes>
              {children}
              <Route path="*" element={<Navigate to="/" replace />} />
            </Routes>
          </Suspense>
        </main>
        <footer>
          People HRIS <span>Built around your people.</span>
        </footer>
      </div>
    </div>
  );
}

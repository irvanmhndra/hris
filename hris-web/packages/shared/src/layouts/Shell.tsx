import { useQuery } from "@tanstack/react-query";
import {
  ArrowUpRight,
  Building2,
  CalendarDays,
  ChevronRight,
  Clock3,
  LayoutDashboard,
  Leaf,
  LogOut,
  Menu,
  Users,
  Wallet,
  FileText,
  Megaphone,
  ClipboardCheck,
  Package,
  Target,
  BriefcaseBusiness,
  UserRound,
  History,
  Settings2,
  CalendarCheck,
} from "lucide-react";
import { useEffect, useState } from "react";
import { Navigate, NavLink, Route, Routes, useLocation } from "react-router";
import {
  client,
  date,
  ErrorBox,
  initials,
  Loading,
  today,
} from "../components/common";
import { Attendances } from "../pages/Attendances";
import { Departments } from "../pages/Departments";
import { EmployeeHome } from "../pages/EmployeeHome";
import { Employees } from "../pages/Employees";
import { Leaves } from "../pages/Leaves";
import { Login } from "../pages/Login";
import { Overview } from "../pages/Overview";
import { HRModule } from "../pages/HRModule";
import { Calendar, Balances, Profiles, Audit } from "../pages/HRSettings";
import { Payroll, Salaries, Payslips } from "../pages/Payroll";
import { api } from "../services/api";
import { useSession } from "../stores/session";
import type { User } from "../types";
export function Shell({ mode }: { mode: "admin" | "employee" }) {
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
  const links = admin
    ? ([
        ["/", "Ringkasan", LayoutDashboard],
        ["/employees", "Karyawan", Users],
        ["/departments", "Departemen", Building2],
        ["/attendance", "Kehadiran", Clock3],
        ["/leaves", "Pengajuan cuti", CalendarDays],
        ["/balances", "Saldo cuti", CalendarCheck],
        ["/calendar", "Kalender kerja", CalendarDays],
        ["/hr/overtime", "Lembur", Clock3],
        ["/hr/corrections", "Koreksi absensi", ClipboardCheck],
        ["/profiles", "Profil karyawan", UserRound],
        ["/hr/announcements", "Pengumuman", Megaphone],
        ["/hr/documents", "Dokumen kebijakan", FileText],
        ["/hr/onboarding", "Onboarding", ClipboardCheck],
        ["/hr/assets", "Inventaris aset", Package],
        ["/hr/goals", "Target kinerja", Target],
        ["/hr/recruitment", "Rekrutmen", BriefcaseBusiness],
        ["/salaries", "Komponen gaji", Settings2],
        ["/payroll", "Payroll", Wallet],
        ["/audit", "Riwayat aktivitas", History],
      ] as const)
    : ([
        ["/", "Beranda", LayoutDashboard],
        ["/attendance", "Kehadiran saya", Clock3],
        ["/leaves", "Cuti & izin", CalendarDays],
        ["/balances", "Saldo cuti", CalendarCheck],
        ["/calendar", "Kalender kerja", CalendarDays],
        ["/hr/overtime", "Lembur saya", Clock3],
        ["/hr/corrections", "Koreksi absensi", ClipboardCheck],
        ["/profiles", "Profil saya", UserRound],
        ["/hr/announcements", "Pengumuman", Megaphone],
        ["/hr/documents", "Dokumen kebijakan", FileText],
        ["/hr/onboarding", "Onboarding saya", ClipboardCheck],
        ["/hr/assets", "Aset saya", Package],
        ["/hr/goals", "Target saya", Target],
        ["/payslips", "Slip gaji", Wallet],
      ] as const);
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
          <Routes>
            <Route path="/" element={admin ? <Overview /> : <EmployeeHome />} />
            {admin && (
              <>
                <Route path="/employees" element={<Employees />} />
                <Route path="/departments" element={<Departments />} />
                <Route path="/salaries" element={<Salaries />} />
                <Route path="/payroll" element={<Payroll />} />
                <Route path="/audit" element={<Audit />} />
              </>
            )}
            <Route
              path="/attendance"
              element={<Attendances employee={!admin} />}
            />
            <Route path="/leaves" element={<Leaves employee={!admin} />} />
            <Route path="/balances" element={<Balances />} />
            <Route path="/calendar" element={<Calendar />} />
            <Route path="/profiles" element={<Profiles />} />
            <Route path="/hr/:module" element={<HRModule />} />
            {!admin && <Route path="/payslips" element={<Payslips />} />}
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </main>
        <footer>
          People HRIS <span>Built around your people.</span>
        </footer>
      </div>
    </div>
  );
}

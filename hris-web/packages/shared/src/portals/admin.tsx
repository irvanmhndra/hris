import {
  BriefcaseBusiness,
  Building2,
  CalendarCheck,
  CalendarDays,
  CalendarRange,
  ClipboardCheck,
  Clock3,
  FileText,
  History,
  LayoutDashboard,
  Megaphone,
  Package,
  Settings2,
  Target,
  UserRound,
  Users,
  Wallet,
} from "lucide-react";
import { Route } from "react-router";
import { HRISApp, lazyPage } from "../App";
import type { NavItem } from "../layouts/Shell";
const Overview = lazyPage(() => import("../pages/Overview"), "Overview");
const Employees = lazyPage(() => import("../pages/Employees"), "Employees");
const Departments = lazyPage(
  () => import("../pages/Departments"),
  "Departments",
);
const Attendances = lazyPage(
  () => import("../pages/Attendances"),
  "Attendances",
);
const Leaves = lazyPage(() => import("../pages/Leaves"), "Leaves");
const Balances = lazyPage(() => import("../pages/HRSettings"), "Balances");
const Calendar = lazyPage(() => import("../pages/HRSettings"), "Calendar");
const Profiles = lazyPage(() => import("../pages/HRSettings"), "Profiles");
const HRModule = lazyPage(() => import("../pages/HRModule"), "HRModule");
const Salaries = lazyPage(() => import("../pages/Payroll"), "Salaries");
const Payroll = lazyPage(() => import("../pages/Payroll"), "Payroll");
const Audit = lazyPage(() => import("../pages/Audit"), "Audit");
const Shifts = lazyPage(() => import("../pages/Shifts"), "Shifts");
const links: readonly NavItem[] = [
  ["/", "Ringkasan", LayoutDashboard],
  ["/employees", "Karyawan", Users],
  ["/departments", "Departemen", Building2],
  ["/attendance", "Kehadiran", Clock3],
  ["/shifts", "Shift & jadwal", CalendarRange],
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
];
export function AdminApp() {
  return (
    <HRISApp mode="admin" links={links}>
      <Route path="/" element={<Overview />} />
      <Route path="/employees" element={<Employees />} />
      <Route path="/departments" element={<Departments />} />
      <Route path="/attendance" element={<Attendances employee={false} />} />
      <Route path="/leaves" element={<Leaves employee={false} />} />
      <Route path="/balances" element={<Balances />} />
      <Route path="/calendar" element={<Calendar />} />
      <Route path="/profiles" element={<Profiles />} />
      <Route path="/hr/:module" element={<HRModule />} />
      <Route path="/salaries" element={<Salaries />} />
      <Route path="/payroll" element={<Payroll />} />
      <Route path="/audit" element={<Audit />} />
      <Route path="/shifts" element={<Shifts />} />
    </HRISApp>
  );
}

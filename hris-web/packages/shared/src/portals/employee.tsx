import {
  CalendarCheck,
  CalendarDays,
  ClipboardCheck,
  Clock3,
  FileText,
  LayoutDashboard,
  Megaphone,
  Package,
  Target,
  UserRound,
  Wallet,
} from "lucide-react";
import { Route } from "react-router";
import { HRISApp, lazyPage } from "../App";
import type { NavItem } from "../layouts/Shell";
const EmployeeHome = lazyPage(
  () => import("../pages/EmployeeHome"),
  "EmployeeHome",
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
const Payslips = lazyPage(() => import("../pages/Payslips"), "Payslips");
const links: readonly NavItem[] = [
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
];
export function EmployeeApp() {
  return (
    <HRISApp mode="employee" links={links}>
      <Route path="/" element={<EmployeeHome />} />
      <Route path="/attendance" element={<Attendances employee />} />
      <Route path="/leaves" element={<Leaves employee />} />
      <Route path="/balances" element={<Balances />} />
      <Route path="/calendar" element={<Calendar />} />
      <Route path="/profiles" element={<Profiles />} />
      <Route path="/hr/:module" element={<HRModule />} />
      <Route path="/payslips" element={<Payslips />} />
    </HRISApp>
  );
}

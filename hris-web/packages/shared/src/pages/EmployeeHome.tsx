import {
  CalendarDays,
  CircleCheck,
  Plus,
  Wallet,
  Megaphone,
  Target,
  ClipboardCheck,
} from "lucide-react";
import { NavLink } from "react-router";
import {
  Badge,
  date,
  Empty,
  ErrorBox,
  initials,
  kinds,
  useData,
} from "../components/common";
import { Heading } from "../components/Heading";
import { useSession } from "../stores/session";
import type { Leave } from "../types";
import { BalanceSummary } from "./HRSettings";
import { ClockCard } from "./Attendance";
export function EmployeeHome() {
  const user = useSession((s) => s.user)!;
  const q = useData<Leave[]>("/leaves");
  return (
    <>
      <Heading
        eyebrow="YOUR PERSONAL SPACE"
        title={`Halo, ${user.name.split(" ")[0]} 👋`}
        description="Hari yang baik dimulai dari langkah kecil. Semangat berkarya!"
      />
      <ClockCard />
      <div className="quick-links">
        {[
          ["/payslips", "Slip gaji", Wallet],
          ["/hr/announcements", "Pengumuman", Megaphone],
          ["/hr/onboarding", "Onboarding", ClipboardCheck],
          ["/hr/goals", "Target saya", Target],
        ].map(([path, label, Icon]) => {
          const I = Icon as typeof Wallet;
          return (
            <NavLink
              className="quick-link"
              key={String(path)}
              to={String(path)}
            >
              <I size={18} />
              {String(label)}
            </NavLink>
          );
        })}
      </div>
      <BalanceSummary />
      <div className="employee-grid">
        <section className="panel personal">
          <div className="avatar large">{initials(user.name)}</div>
          <h2>{user.name}</h2>
          <p>{user.email}</p>
          <span className="department-tag">{user.company_name}</span>
          <div className="personal-note">
            <CircleCheck size={18} />
            Akun karyawan aktif
          </div>
        </section>
        <section className="panel">
          <div className="panel-head">
            <div>
              <h2>Pengajuan saya</h2>
              <p>Pantau status cuti dan izin Anda.</p>
            </div>
            <NavLink to="/leaves">
              Ajukan cuti <Plus size={16} />
            </NavLink>
          </div>
          <ErrorBox error={q.error} />
          {q.data?.length ? (
            q.data.slice(0, 4).map((l) => (
              <div className="request" key={l.id}>
                <CalendarDays size={20} />
                <div className="request-name">
                  <strong>{kinds[l.kind]}</strong>
                  <small>
                    {date(l.start_date)} · {l.days} hari
                  </small>
                </div>
                <Badge status={l.status} />
              </div>
            ))
          ) : (
            <Empty>Belum ada pengajuan cuti.</Empty>
          )}
        </section>
      </div>
    </>
  );
}

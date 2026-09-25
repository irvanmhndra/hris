import {
  ArrowRight,
  ArrowUpRight,
  BriefcaseBusiness,
  Building2,
  CalendarDays,
  Clock3,
  Leaf,
  Users,
} from "lucide-react";
import { NavLink } from "react-router";
import {
  Badge,
  date,
  Empty,
  ErrorBox,
  initials,
  kinds,
  Loading,
  useData,
} from "../components/common";
import { Heading } from "../components/Heading";
import { useSession } from "../stores/session";
import type { Dashboard, Department, Employee, Leave } from "../types";
import { EmployeeTable } from "./Employees";
export function Overview() {
  const user = useSession((s) => s.user)!;
  const d = useData<Dashboard>("/dashboard");
  const employees = useData<Employee[]>("/employees");
  const leaves = useData<Leave[]>("/leaves");
  const departments = useData<Department[]>("/departments");
  const stats = d.data;
  return (
    <>
      <Heading
        eyebrow="YOUR PEOPLE, AT A GLANCE"
        title={`Halo, ${user.name.split(" ")[0]} 👋`}
        description="Inilah kabar tim Anda hari ini. Mari buat hari yang berarti."
      >
        <NavLink className="secondary" to="/employees">
          Lihat karyawan <ArrowUpRight size={16} />
        </NavLink>
      </Heading>
      <div className="welcome-banner">
        <div>
          <span className="banner-label">
            <span /> PEOPLE & CULTURE
          </span>
          <h2>
            Orang-orang hebat.
            <br />
            Kemungkinan tak terbatas.
          </h2>
          <p>Kelola tim dengan lebih dekat, tumbuh bersama lebih jauh.</p>
          <NavLink to="/employees">
            Kenali tim Anda <ArrowRight size={16} />
          </NavLink>
        </div>
        <div className="banner-art" aria-hidden="true">
          <div className="orbit one" />
          <div className="orbit two" />
          <div className="art-tile tile-a">
            <Users size={35} />
          </div>
          <div className="art-tile tile-b">
            <Leaf size={30} />
          </div>
          <div className="art-tile tile-c">
            <BriefcaseBusiness size={26} />
          </div>
          <span className="art-spark">✳</span>
          <div className="art-caption">
            <span className="small-dot" /> Growing, together.
          </div>
        </div>
      </div>
      <ErrorBox error={d.error} />
      <div className="stats">
        {[
          {
            label: "Total karyawan aktif",
            value: stats?.employees,
            icon: Users,
            note: "Orang hebat dalam tim",
            color: "green",
          },
          {
            label: "Hadir hari ini",
            value: stats?.present,
            icon: Clock3,
            note: "Sudah melakukan check-in",
            color: "blue",
          },
          {
            label: "Menunggu persetujuan",
            value: stats?.pending,
            icon: CalendarDays,
            note: "Pengajuan cuti & izin",
            color: "orange",
          },
          {
            label: "Departemen",
            value: stats?.departments,
            icon: Building2,
            note: "Berkolaborasi setiap hari",
            color: "purple",
          },
        ].map((s) => (
          <section className="stat" key={s.label}>
            <div>
              <span>{s.label}</span>
              <div className={`stat-icon ${s.color}`}>
                <s.icon size={19} />
              </div>
            </div>
            <strong>{s.value ?? "—"}</strong>
            <small>{s.note}</small>
          </section>
        ))}
      </div>
      <div className="overview-grid">
        <section className="panel">
          <div className="panel-head">
            <div>
              <h2>Pengajuan terbaru</h2>
              <p>Sedikit perhatian, dampak yang berarti.</p>
            </div>
            <NavLink to="/leaves">
              Lihat semua <ArrowUpRight size={15} />
            </NavLink>
          </div>
          <ErrorBox error={leaves.error} />
          {leaves.isLoading ? (
            <Loading />
          ) : leaves.data?.length ? (
            <div className="request-list">
              {leaves.data.slice(0, 4).map((l) => (
                <div className="request" key={l.id}>
                  <span className="avatar lilac">{initials(l.name)}</span>
                  <div className="request-name">
                    <strong>{l.name}</strong>
                    <small>
                      {kinds[l.kind]} · {l.days} hari
                    </small>
                  </div>
                  <div className="request-date">{date(l.start_date)}</div>
                  <Badge status={l.status} />
                </div>
              ))}
            </div>
          ) : (
            <Empty>Belum ada pengajuan cuti.</Empty>
          )}
        </section>
        <section className="panel">
          <div className="panel-head">
            <div>
              <h2>Komposisi tim</h2>
              <p>Beragam keahlian, satu tujuan.</p>
            </div>
            <Building2 size={19} />
          </div>
          <ErrorBox error={departments.error} />
          <div className="department-bars">
            {departments.data?.map((dep, i) => (
              <div key={dep.id}>
                <div>
                  <span>
                    <i
                      style={{
                        background: [
                          "#6366f1",
                          "#8b83da",
                          "#a5b4fc",
                          "#a09ac4",
                          "#93a4d5",
                        ][i % 5],
                      }}
                    />
                    {dep.name}
                  </span>
                  <strong>
                    {dep.count}
                    <small> orang</small>
                  </strong>
                </div>
                <div className="bar-track">
                  <span
                    style={{
                      width: `${stats?.employees ? (dep.count / stats.employees) * 100 : 0}%`,
                      background: [
                        "#6366f1",
                        "#8b83da",
                        "#a5b4fc",
                        "#a09ac4",
                        "#93a4d5",
                      ][i % 5],
                    }}
                  />
                </div>
              </div>
            ))}
          </div>
        </section>
      </div>
      <section className="panel">
        <div className="panel-head">
          <div>
            <h2>Bagian dari perjalanan kita</h2>
            <p>Kenali orang-orang di balik setiap pencapaian.</p>
          </div>
          <NavLink to="/employees">
            Direktori karyawan <ArrowUpRight size={15} />
          </NavLink>
        </div>
        <ErrorBox error={employees.error} />
        {employees.data && <EmployeeTable rows={employees.data.slice(0, 5)} />}
      </section>
    </>
  );
}

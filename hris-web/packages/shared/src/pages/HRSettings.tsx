import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { Plus, CalendarDays, UserRound, Save } from "lucide-react";
import { Heading } from "../components/Heading";
import {
  date,
  Empty,
  ErrorBox,
  Loading,
  Modal,
  today,
  useData,
} from "../components/common";
import { api } from "../services/api";
import { useSession } from "../stores/session";
import type {
  WorkCalendar,
  Holiday,
  Balance,
  Profile,
  Employee,
} from "../types";
const days = ["Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"];
export function Calendar() {
  const admin = useSession((s) => s.user?.role === "admin");
  const q = useData<WorkCalendar>("/calendar");
  const holidays = useData<Holiday[]>("/holidays");
  const qc = useQueryClient();
  const [open, setOpen] = useState(false);
  const [remove, setRemove] = useState<Holiday | null>(null);
  const m = useMutation({
    mutationFn: (v: unknown) => api("/holidays", "POST", v),
    onSuccess: () => {
      qc.invalidateQueries();
      setOpen(false);
    },
  });
  const del = useMutation({
    mutationFn: (id: number) => api(`/holidays/${id}`, "DELETE"),
    onSuccess: () => {
      qc.invalidateQueries();
      setRemove(null);
    },
  });
  return (
    <>
      <Heading
        eyebrow="WORK CALENDAR"
        title="Ritme kerja yang jelas."
        description="Hari kerja, jam operasional, dan hari libur perusahaan."
      />
      <ErrorBox error={q.error} />
      {q.isLoading ? (
        <Loading />
      ) : (
        q.data && (
          <CalendarForm
            key={q.dataUpdatedAt}
            calendar={q.data}
            admin={!!admin}
          />
        )
      )}
      <section className="panel">
        <div className="panel-head">
          <div>
            <h2>Hari libur perusahaan</h2>
            <p>Dikecualikan dari perhitungan pengajuan cuti baru.</p>
          </div>
          {admin && (
            <button
              className="primary"
              onClick={() => {
                m.reset();
                setOpen(true);
              }}
            >
              <Plus size={15} />
              Tambah hari libur
            </button>
          )}
        </div>
        <ErrorBox error={holidays.error} />
        {holidays.data?.map((h) => (
          <div className="request" key={h.id}>
            <CalendarDays size={19} />
            <div className="request-name">
              <strong>{h.name}</strong>
              <small>{date(h.date)}</small>
            </div>
            {admin && (
              <button
                className="text-button"
                onClick={() => {
                  del.reset();
                  setRemove(h);
                }}
              >
                Hapus
              </button>
            )}
          </div>
        ))}
        {!holidays.isLoading && !holidays.data?.length && (
          <Empty>Belum ada hari libur yang ditambahkan HR.</Empty>
        )}
      </section>
      {open && (
        <Modal title="Tambah hari libur" close={() => setOpen(false)}>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              m.mutate(Object.fromEntries(new FormData(e.currentTarget)));
            }}
          >
            <label>
              Nama hari libur
              <input name="name" minLength={2} maxLength={120} required />
            </label>
            <label>
              Tanggal
              <input name="date" type="date" required />
            </label>
            <ErrorBox error={m.error} />
            <div className="modal-actions">
              <button className="primary" disabled={m.isPending}>
                Simpan hari libur
              </button>
            </div>
          </form>
        </Modal>
      )}
      {remove && (
        <Modal title="Hapus hari libur?" close={() => setRemove(null)}>
          <p>
            {remove.name} · {date(remove.date)}. Perhitungan cuti yang sudah
            diajukan tetap dipertahankan.
          </p>
          <ErrorBox error={del.error} />
          <div className="modal-actions">
            <button className="secondary" onClick={() => setRemove(null)}>
              Batal
            </button>
            <button
              className="primary"
              disabled={del.isPending}
              onClick={() => del.mutate(remove.id)}
            >
              Hapus hari libur
            </button>
          </div>
        </Modal>
      )}
    </>
  );
}
function CalendarForm({
  calendar,
  admin,
}: {
  calendar: WorkCalendar;
  admin: boolean;
}) {
  const qc = useQueryClient();
  const m = useMutation({
    mutationFn: (v: unknown) => api("/calendar", "PUT", v),
    onSuccess: () => qc.invalidateQueries(),
  });
  return (
    <section className="panel settings-panel">
      <h2>Pengaturan hari kerja</h2>
      <p className="section-description">
        Zona waktu Asia/Jakarta. Perubahan kalender berlaku untuk pengajuan
        baru.
      </p>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          const data = new FormData(e.currentTarget);
          m.mutate({
            workdays: data.getAll("workdays").map(Number),
            annual_allowance: Number(data.get("annual_allowance")),
            start_time: data.get("start_time"),
            end_time: data.get("end_time"),
          });
        }}
      >
        <div className="weekday-picker">
          {days.map((d, i) => (
            <label key={d}>
              <input
                type="checkbox"
                name="workdays"
                value={i}
                defaultChecked={calendar.workdays.includes(i)}
                disabled={!admin}
              />
              <span>{d}</span>
            </label>
          ))}
        </div>
        <div className="form-grid">
          <label>
            Jam mulai (WIB)
            <input
              type="time"
              name="start_time"
              defaultValue={calendar.start_time}
              required
              disabled={!admin}
            />
          </label>
          <label>
            Jam selesai (WIB)
            <input
              type="time"
              name="end_time"
              defaultValue={calendar.end_time}
              required
              disabled={!admin}
            />
          </label>
          <label>
            Kuota cuti tahunan default
            <input
              type="number"
              name="annual_allowance"
              min={0}
              max={366}
              defaultValue={calendar.annual_allowance}
              required
              disabled={!admin}
            />
          </label>
        </div>
        <p className="form-hint">
          Kuota default digunakan sampai ada alokasi tersimpan untuk
          karyawan/tahun tersebut. Alokasi yang telah tersimpan dikelola di
          Saldo cuti. Jam kerja ini informasi jadwal umum; belum menghitung
          keterlambatan atau shift.
        </p>
        <ErrorBox error={m.error} />
        {admin && (
          <div className="modal-actions">
            <button className="primary" disabled={m.isPending}>
              <Save size={15} />
              Simpan kalender
            </button>
          </div>
        )}
      </form>
    </section>
  );
}
export function BalanceSummary() {
  const year = Number(today().slice(0, 4));
  const q = useData<Balance[]>(`/leave-balances?year=${year}`);
  const b = q.data?.[0];
  return (
    <>
      <ErrorBox error={q.error} />
      {b && (
        <div className="balance-cards">
          {[
            ["Tersedia", b.available],
            ["Kuota tahunan", b.allowance],
            ["Terpakai", b.used],
            ["Dicadangkan", b.reserved],
          ].map(([label, value]) => (
            <div key={label} className="balance-card">
              <span>
                {label} · {year}
              </span>
              <strong>
                {value}
                <small> hari</small>
              </strong>
            </div>
          ))}
        </div>
      )}
    </>
  );
}
export function Balances() {
  const admin = useSession((s) => s.user?.role === "admin");
  const [year, setYear] = useState(Number(today().slice(0, 4)));
  const q = useData<Balance[]>(`/leave-balances?year=${year}`);
  const [edit, setEdit] = useState<Balance | null>(null);
  const qc = useQueryClient();
  const m = useMutation({
    mutationFn: (allowance: number) =>
      api(`/leave-balances/${edit!.employee_id}`, "PUT", { year, allowance }),
    onSuccess: () => {
      qc.invalidateQueries();
      setEdit(null);
    },
  });
  return (
    <>
      <Heading
        eyebrow="LEAVE ENTITLEMENT"
        title="Waktu istirahat, terencana."
        description="Cuti tahunan pending mencadangkan saldo. Penolakan atau pembatalan mengembalikannya."
      >
        <select
          aria-label="Tahun saldo"
          value={year}
          onChange={(e) => setYear(Number(e.target.value))}
        >
          {Array.from(
            { length: 7 },
            (_, i) => Number(today().slice(0, 4)) - 2 + i,
          ).map((y) => (
            <option key={y}>{y}</option>
          ))}
        </select>
      </Heading>
      <section className="panel">
        <ErrorBox error={q.error} />
        {q.isLoading ? (
          <Loading />
        ) : (
          <div className="table-scroll">
            <table>
              <thead>
                <tr>
                  <th>KARYAWAN</th>
                  <th>KUOTA</th>
                  <th>TERPAKAI</th>
                  <th>DICADANGKAN</th>
                  <th>TERSEDIA</th>
                  {admin && <th />}
                </tr>
              </thead>
              <tbody>
                {q.data?.map((b) => (
                  <tr key={b.employee_id}>
                    <td>{b.name}</td>
                    <td>{b.allowance} hari</td>
                    <td>{b.used} hari</td>
                    <td>{b.reserved} hari</td>
                    <td>
                      <strong>{b.available} hari</strong>
                    </td>
                    {admin && (
                      <td>
                        <button
                          className="text-button"
                          onClick={() => {
                            m.reset();
                            setEdit(b);
                          }}
                        >
                          Atur kuota
                        </button>
                      </td>
                    )}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
      <p className="section-description">
        Pengajuan baru mengikuti hari kerja dan libur perusahaan. Pengajuan lama
        tetap menggunakan perhitungan saat diajukan; cuti sakit dan izin pribadi
        tidak mengurangi kuota tahunan.
      </p>
      {edit && (
        <Modal title={`Kuota cuti · ${edit.name}`} close={() => setEdit(null)}>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              m.mutate(Number(new FormData(e.currentTarget).get("allowance")));
            }}
          >
            <label>
              Kuota tahun {year}
              <input
                name="allowance"
                type="number"
                min={edit.used + edit.reserved}
                max={366}
                defaultValue={edit.allowance}
                required
              />
            </label>
            <ErrorBox error={m.error} />
            <div className="modal-actions">
              <button className="primary" disabled={m.isPending}>
                Simpan kuota
              </button>
            </div>
          </form>
        </Modal>
      )}
    </>
  );
}
export function Profiles() {
  const admin = useSession((s) => s.user?.role === "admin");
  const [selected, setSelected] = useState("");
  const employees = useQuery({
    queryKey: ["/employees"],
    queryFn: () => api<Employee[]>("/employees"),
    enabled: !!admin,
  });
  return (
    <>
      <Heading
        eyebrow="EMPLOYEE PROFILE"
        title={admin ? "Informasi yang lebih lengkap." : "Profil saya."}
        description="Data pekerjaan, kontak pribadi, dan kontak darurat."
      />
      {admin && (
        <section className="panel settings-panel">
          <label>
            Pilih karyawan
            <select
              value={selected}
              onChange={(e) => setSelected(e.target.value)}
            >
              <option value="">Pilih nama karyawan</option>
              {employees.data?.map((e) => (
                <option value={e.id} key={e.id}>
                  {e.name} · {e.code}
                </option>
              ))}
            </select>
          </label>
          <ErrorBox error={employees.error} />
        </section>
      )}
      {(!admin || selected) && (
        <ProfileEditor
          key={selected}
          path={admin ? `/employees/${selected}/profile` : "/profile"}
        />
      )}
    </>
  );
}
function ProfileEditor({ path }: { path: string }) {
  const q = useData<Profile>(path);
  const qc = useQueryClient();
  const m = useMutation({
    mutationFn: (v: unknown) => api(path, "PUT", v),
    onSuccess: () => qc.invalidateQueries({ queryKey: [path] }),
  });
  if (q.isLoading) return <Loading />;
  if (!q.data) return <ErrorBox error={q.error} />;
  const p = q.data;
  return (
    <section className="panel settings-panel">
      <div className="profile-heading">
        <span className="record-icon">
          <UserRound />
        </span>
        <div>
          <h2>{p.name}</h2>
          <p>
            {p.code} · {p.position} · {p.department}
          </p>
          <small>
            {p.email} · Bergabung {date(p.joined_on)}
          </small>
        </div>
      </div>
      <form
        key={q.dataUpdatedAt}
        onSubmit={(e) => {
          e.preventDefault();
          m.mutate(Object.fromEntries(new FormData(e.currentTarget)));
        }}
      >
        <div className="form-grid">
          {(
            [
              { key: "phone", label: "Nomor telepon", max: 30 },
              { key: "address", label: "Alamat domisili", max: 500 },
              { key: "emergency_name", label: "Nama kontak darurat", max: 120 },
              {
                key: "emergency_phone",
                label: "Telepon kontak darurat",
                max: 30,
              },
              {
                key: "emergency_relation",
                label: "Hubungan dengan kontak darurat",
                max: 80,
              },
            ] as const
          ).map((f) => (
            <label key={f.key}>
              {f.label}
              <input name={f.key} defaultValue={p[f.key]} maxLength={f.max} />
            </label>
          ))}
        </div>
        <ErrorBox error={m.error} />
        {m.isSuccess && (
          <p className="success-message" role="status">
            Profil berhasil disimpan.
          </p>
        )}
        <div className="modal-actions">
          <button className="primary" disabled={m.isPending}>
            Simpan profil
          </button>
        </div>
      </form>
    </section>
  );
}

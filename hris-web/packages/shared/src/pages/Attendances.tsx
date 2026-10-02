import { useState } from "react";
import {
  date,
  Empty,
  ErrorBox,
  initials,
  Loading,
  time,
  today,
  useData,
} from "../components/common";
import { Heading } from "../components/Heading";
import type { Attendance, AttendanceSummary } from "../types";
import { ClockCard } from "./Attendance";
export function Attendances({ employee }: { employee: boolean }) {
  const q = useData<Attendance[]>("/attendances");
  const [day, setDay] = useState("");
  const rows = q.data?.filter((a) => !day || a.date === day) || [];
  return (
    <>
      <Heading
        eyebrow="TIME & ATTENDANCE"
        title={employee ? "Setiap hari, satu langkah maju." : "Kehadiran tim."}
        description="Catatan check-in dan check-out dalam zona waktu Asia/Jakarta."
      />
      {employee ? <ClockCard /> : <Summary />}
      <section className="panel">
        <div className="toolbar">
          <h2>{employee ? "Riwayat kehadiran" : "Log kehadiran"}</h2>
          <label className="inline-label">
            Tanggal
            <input
              type="date"
              value={day}
              onChange={(e) => setDay(e.target.value)}
            />
          </label>
          {day && (
            <button className="text-button" onClick={() => setDay("")}>
              Reset
            </button>
          )}
          <span className="count">100 catatan terbaru</span>
        </div>
        <ErrorBox error={q.error} />
        {q.isLoading ? (
          <Loading />
        ) : (
          <div className="table-scroll">
            <table>
              <thead>
                <tr>
                  <th>KARYAWAN</th>
                  <th>TANGGAL</th>
                  <th>CHECK-IN</th>
                  <th>CHECK-OUT</th>
                  <th>DURASI</th>
                  <th>JADWAL</th>
                  <th>CATATAN</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((a) => (
                  <tr key={a.id}>
                    <td>
                      <div className="person">
                        <span className="avatar">{initials(a.name)}</span>
                        <strong>{a.name}</strong>
                      </div>
                    </td>
                    <td>{date(a.date)}</td>
                    <td>
                      <span className="time-tag">{time(a.check_in)}</span>
                    </td>
                    <td>{time(a.check_out)}</td>
                    <td>
                      {a.check_out
                        ? `${Math.floor((+new Date(a.check_out) - +new Date(a.check_in)) / 3600000)}j ${Math.floor((+new Date(a.check_out) - +new Date(a.check_in)) / 60000) % 60}m`
                        : "—"}
                    </td>
                    <td>
                      {a.shift_name || "—"}
                      {a.scheduled_start && (
                        <small className="block">
                          {time(a.scheduled_start)}–{time(a.scheduled_end)}
                        </small>
                      )}
                    </td>
                    <td>
                      {a.late_minutes > 0 && (
                        <span className="badge rejected">
                          Terlambat {a.late_minutes}m
                        </span>
                      )}
                      {a.early_leave_minutes > 0 && (
                        <span className="badge pending">
                          Pulang cepat {a.early_leave_minutes}m
                        </span>
                      )}
                      {a.check_in_distance_m != null && (
                        <small className="block">
                          {a.check_in_distance_m} m dari lokasi
                        </small>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            {!rows.length && <Empty>Belum ada catatan kehadiran.</Empty>}
          </div>
        )}
      </section>
    </>
  );
}

function Summary() {
  const [month, setMonth] = useState(today().slice(0, 7));
  const q = useData<AttendanceSummary[]>(`/attendance-summary?month=${month}`);
  return (
    <section className="panel">
      <div className="toolbar">
        <h2>Rekap bulanan</h2>
        <label className="inline-label">
          Bulan
          <input
            type="month"
            value={month}
            onChange={(e) => e.target.value && setMonth(e.target.value)}
          />
        </label>
        <span className="count">Sampai hari ini · sesuai jadwal shift</span>
      </div>
      <ErrorBox error={q.error} />
      {q.isLoading ? (
        <Loading />
      ) : (
        <div className="table-scroll">
          <table>
            <thead>
              <tr>
                <th>KARYAWAN</th>
                <th>HARI KERJA</th>
                <th>HADIR</th>
                <th>TERLAMBAT</th>
                <th>PULANG CEPAT</th>
                <th>CUTI</th>
                <th>TANPA KETERANGAN</th>
              </tr>
            </thead>
            <tbody>
              {q.data?.map((r) => (
                <tr key={r.employee_id}>
                  <td>{r.name}</td>
                  <td>{r.scheduled_days}</td>
                  <td>{r.present_days}</td>
                  <td>
                    {r.late_days}×
                    <small className="block">{r.late_minutes} menit</small>
                  </td>
                  <td>
                    {r.early_leave_days}×
                    <small className="block">
                      {r.early_leave_minutes} menit
                    </small>
                  </td>
                  <td>{r.leave_days}</td>
                  <td>
                    {r.absent_days > 0 ? <strong>{r.absent_days}</strong> : 0}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {!q.data?.length && <Empty />}
        </div>
      )}
    </section>
  );
}

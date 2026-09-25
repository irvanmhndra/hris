import { useState } from "react";
import {
  date,
  Empty,
  ErrorBox,
  initials,
  Loading,
  time,
  useData,
} from "../components/common";
import { Heading } from "../components/Heading";
import type { Attendance } from "../types";
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
      {employee && <ClockCard />}
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

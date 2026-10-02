import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { ChevronLeft, ChevronRight, MapPin, Plus, Trash2 } from "lucide-react";
import { Heading } from "../components/Heading";
import {
  Empty,
  ErrorBox,
  Loading,
  Modal,
  today,
  useData,
} from "../components/common";
import { api } from "../services/api";
import type {
  AttendanceLocation,
  EmployeeSchedule,
  Shift,
  WorkCalendar,
} from "../types";

const iso = (d: Date) => d.toISOString().slice(0, 10);
const addDays = (s: string, n: number) => {
  const d = new Date(s + "T00:00:00Z");
  d.setUTCDate(d.getUTCDate() + n);
  return iso(d);
};
const monday = (s: string) => {
  const d = new Date(s + "T00:00:00Z");
  return addDays(s, -((d.getUTCDay() + 6) % 7));
};
const dayLabel = (s: string) =>
  new Date(s + "T00:00:00Z").toLocaleDateString("id-ID", {
    weekday: "short",
    day: "numeric",
    month: "short",
    timeZone: "UTC",
  });

function useSave<T>(
  path: (v: T) => string,
  method: (v: T) => string,
  done: () => void,
) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (v: T) => api(path(v), method(v), v),
    onSuccess: () => {
      qc.invalidateQueries();
      done();
    },
  });
}

function ShiftForm({ shift, close }: { shift?: Shift; close: () => void }) {
  const m = useSave<Partial<Shift>>(
    () => (shift ? `/shifts/${shift.id}` : "/shifts"),
    () => (shift ? "PUT" : "POST"),
    close,
  );
  return (
    <Modal
      title={shift ? `Edit shift · ${shift.name}` : "Tambah shift"}
      close={close}
    >
      <form
        onSubmit={(e) => {
          e.preventDefault();
          const d = new FormData(e.currentTarget);
          m.mutate({
            name: String(d.get("name")),
            start_time: String(d.get("start_time")),
            end_time: String(d.get("end_time")),
            grace_minutes: Number(d.get("grace_minutes")),
            active: d.get("active") !== "false",
          });
        }}
      >
        <label>
          Nama shift
          <input
            name="name"
            defaultValue={shift?.name}
            maxLength={80}
            required
          />
        </label>
        <div className="form-grid">
          <label>
            Jam mulai
            <input
              name="start_time"
              type="time"
              defaultValue={shift?.start_time || "08:00"}
              required
            />
          </label>
          <label>
            Jam selesai
            <input
              name="end_time"
              type="time"
              defaultValue={shift?.end_time || "17:00"}
              required
            />
          </label>
          <label>
            Toleransi terlambat (menit)
            <input
              name="grace_minutes"
              type="number"
              min={0}
              max={240}
              defaultValue={shift?.grace_minutes ?? 0}
              required
            />
          </label>
          {shift && (
            <label>
              Status
              <select name="active" defaultValue={String(shift.active)}>
                <option value="true">Aktif</option>
                <option value="false">Nonaktif</option>
              </select>
            </label>
          )}
        </div>
        <p className="form-hint">
          Jam selesai lebih awal dari jam mulai berarti shift berakhir keesokan
          hari (shift malam). Check-out ditutup pada absensi terbuka dalam 20
          jam terakhir.
        </p>
        <ErrorBox error={m.error} />
        <div className="modal-actions">
          <button type="button" className="secondary" onClick={close}>
            Batal
          </button>
          <button className="primary" disabled={m.isPending}>
            Simpan shift
          </button>
        </div>
      </form>
    </Modal>
  );
}

function LocationForm({
  location,
  close,
}: {
  location?: AttendanceLocation;
  close: () => void;
}) {
  const m = useSave<Partial<AttendanceLocation>>(
    () =>
      location
        ? `/attendance-locations/${location.id}`
        : "/attendance-locations",
    () => (location ? "PUT" : "POST"),
    close,
  );
  const [here, setHere] = useState<{ lat: number; lng: number } | null>(null);
  return (
    <Modal
      title={
        location ? `Edit lokasi · ${location.name}` : "Tambah lokasi absensi"
      }
      close={close}
    >
      <form
        key={here ? `${here.lat},${here.lng}` : "initial"}
        onSubmit={(e) => {
          e.preventDefault();
          const d = new FormData(e.currentTarget);
          m.mutate({
            name: String(d.get("name")),
            latitude: Number(d.get("latitude")),
            longitude: Number(d.get("longitude")),
            radius_m: Number(d.get("radius_m")),
          });
        }}
      >
        <label>
          Nama lokasi
          <input
            name="name"
            defaultValue={location?.name}
            maxLength={80}
            required
          />
        </label>
        <div className="form-grid">
          <label>
            Latitude
            <input
              name="latitude"
              type="number"
              step="any"
              min={-90}
              max={90}
              defaultValue={here?.lat ?? location?.latitude}
              required
            />
          </label>
          <label>
            Longitude
            <input
              name="longitude"
              type="number"
              step="any"
              min={-180}
              max={180}
              defaultValue={here?.lng ?? location?.longitude}
              required
            />
          </label>
          <label>
            Radius (meter)
            <input
              name="radius_m"
              type="number"
              min={10}
              max={5000}
              defaultValue={location?.radius_m ?? 100}
              required
            />
          </label>
        </div>
        <button
          type="button"
          className="text-button"
          onClick={() =>
            navigator.geolocation?.getCurrentPosition((p) =>
              setHere({ lat: p.coords.latitude, lng: p.coords.longitude }),
            )
          }
        >
          <MapPin size={14} /> Gunakan lokasi saya saat ini
        </button>
        <ErrorBox error={m.error} />
        <div className="modal-actions">
          <button type="button" className="secondary" onClick={close}>
            Batal
          </button>
          <button className="primary" disabled={m.isPending}>
            Simpan lokasi
          </button>
        </div>
      </form>
    </Modal>
  );
}

function AssignForm({
  people,
  shifts,
  from,
  close,
}: {
  people: EmployeeSchedule[];
  shifts: Shift[];
  from: string;
  close: () => void;
}) {
  const m = useSave<unknown>(
    () => "/schedule",
    () => "PUT",
    close,
  );
  const [selected, setSelected] = useState<number[]>([]);
  return (
    <Modal title="Atur jadwal shift" close={close}>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          const d = new FormData(e.currentTarget);
          const choice = String(d.get("shift"));
          m.mutate({
            employee_ids: selected,
            from: d.get("from"),
            to: d.get("to"),
            shift_id: /^\d+$/.test(choice) ? Number(choice) : null,
            clear: choice === "default",
          });
        }}
      >
        <div className="field-label">Karyawan</div>
        <div className="weekday-picker">
          {people.map((p) => (
            <label key={p.employee_id}>
              <input
                type="checkbox"
                checked={selected.includes(p.employee_id)}
                onChange={(e) =>
                  setSelected(
                    e.target.checked
                      ? [...selected, p.employee_id]
                      : selected.filter((x) => x !== p.employee_id),
                  )
                }
              />
              <span>{p.name}</span>
            </label>
          ))}
        </div>
        <div className="form-grid">
          <label>
            Dari
            <input name="from" type="date" defaultValue={from} required />
          </label>
          <label>
            Sampai
            <input
              name="to"
              type="date"
              defaultValue={addDays(from, 6)}
              required
            />
          </label>
        </div>
        <label>
          Jadwal
          <select
            name="shift"
            defaultValue={shifts.find((s) => s.active)?.id ?? "off"}
          >
            {shifts
              .filter((s) => s.active)
              .map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name} · {s.start_time}–{s.end_time}
                </option>
              ))}
            <option value="off">Libur</option>
            <option value="default">Kembalikan ke jadwal default</option>
          </select>
        </label>
        <p className="form-hint">
          Jadwal per tanggal mengalahkan shift default dan kalender kerja
          (termasuk akhir pekan dan hari libur). Maksimal 92 hari sekaligus.
        </p>
        <ErrorBox error={m.error} />
        <div className="modal-actions">
          <button type="button" className="secondary" onClick={close}>
            Batal
          </button>
          <button
            className="primary"
            disabled={m.isPending || !selected.length}
          >
            Terapkan
          </button>
        </div>
      </form>
    </Modal>
  );
}

export function Shifts() {
  const shifts = useData<Shift[]>("/shifts");
  const locations = useData<AttendanceLocation[]>("/attendance-locations");
  const calendar = useData<WorkCalendar>("/calendar");
  const [from, setFrom] = useState(monday(today()));
  const to = addDays(from, 13);
  const sched = useData<EmployeeSchedule[]>(`/schedule?from=${from}&to=${to}`);
  const [shiftForm, setShiftForm] = useState<Shift | "new" | null>(null);
  const [locationForm, setLocationForm] = useState<
    AttendanceLocation | "new" | null
  >(null);
  const [assign, setAssign] = useState(false);
  const qc = useQueryClient();
  const removeLocation = useMutation({
    mutationFn: (id: number) => api(`/attendance-locations/${id}`, "DELETE"),
    onSuccess: () => qc.invalidateQueries(),
  });
  const requireLocation = useMutation({
    mutationFn: (v: boolean) =>
      api("/calendar", "PUT", { ...calendar.data, require_location: v }),
    onSuccess: () => qc.invalidateQueries(),
  });
  const days = sched.data?.[0]?.days.map((d) => d.date) ?? [];
  return (
    <>
      <Heading
        eyebrow="SHIFTS & SCHEDULE"
        title="Jadwal yang jelas untuk setiap shift."
        description="Atur shift, jadwal per tanggal, dan lokasi absensi. Keterlambatan dihitung dari jadwal saat check-in."
      >
        <button className="primary" onClick={() => setShiftForm("new")}>
          <Plus size={15} />
          Tambah shift
        </button>
      </Heading>
      <section className="panel">
        <div className="panel-head">
          <div>
            <h2>Shift</h2>
            <p>
              Shift default diatur di data karyawan; tanpa shift memakai jam
              kantor.
            </p>
          </div>
        </div>
        <ErrorBox error={shifts.error} />
        {shifts.isLoading ? (
          <Loading />
        ) : (
          <div className="table-scroll">
            <table>
              <thead>
                <tr>
                  <th>NAMA</th>
                  <th>JAM</th>
                  <th>TOLERANSI</th>
                  <th>STATUS</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {shifts.data?.map((s) => (
                  <tr key={s.id}>
                    <td>
                      <strong>{s.name}</strong>
                    </td>
                    <td>
                      {s.start_time}–{s.end_time}
                      {s.end_time <= s.start_time && (
                        <small className="block">Berakhir keesokan hari</small>
                      )}
                    </td>
                    <td>{s.grace_minutes} menit</td>
                    <td>
                      <span
                        className={`badge ${s.active ? "active" : "inactive"}`}
                      >
                        <i />
                        {s.active ? "Aktif" : "Nonaktif"}
                      </span>
                    </td>
                    <td>
                      <button
                        className="text-button"
                        onClick={() => setShiftForm(s)}
                      >
                        Edit
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            {!shifts.data?.length && (
              <Empty>Belum ada shift. Semua karyawan memakai jam kantor.</Empty>
            )}
          </div>
        )}
      </section>
      <section className="panel">
        <div className="panel-head">
          <div>
            <h2>Jadwal 2 minggu</h2>
            <p>Huruf tebal = jadwal per tanggal; abu-abu = libur.</p>
          </div>
          <div className="row-actions">
            <button
              className="icon-button"
              aria-label="Minggu sebelumnya"
              onClick={() => setFrom(addDays(from, -7))}
            >
              <ChevronLeft size={16} />
            </button>
            <button
              className="icon-button"
              aria-label="Minggu berikutnya"
              onClick={() => setFrom(addDays(from, 7))}
            >
              <ChevronRight size={16} />
            </button>
            <button
              className="secondary"
              disabled={!sched.data?.length}
              onClick={() => setAssign(true)}
            >
              Atur jadwal
            </button>
          </div>
        </div>
        <ErrorBox error={sched.error} />
        {sched.isLoading ? (
          <Loading />
        ) : (
          <div className="table-scroll">
            <table className="schedule-grid">
              <thead>
                <tr>
                  <th>KARYAWAN</th>
                  {days.map((d) => (
                    <th key={d}>{dayLabel(d)}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {sched.data?.map((p) => (
                  <tr key={p.employee_id}>
                    <td>{p.name}</td>
                    {p.days.map((d) => (
                      <td
                        key={d.date}
                        className={d.off ? "off" : d.assigned ? "assigned" : ""}
                      >
                        {d.off ? "Libur" : d.shift_name}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
      <section className="panel">
        <div className="panel-head">
          <div>
            <h2>Lokasi absensi</h2>
            <p>Jarak ke lokasi terdekat dicatat saat check-in/out.</p>
          </div>
          <button className="secondary" onClick={() => setLocationForm("new")}>
            <MapPin size={15} />
            Tambah lokasi
          </button>
        </div>
        {calendar.data && (
          <div className="weekday-picker">
            <label>
              <input
                type="checkbox"
                checked={calendar.data.require_location}
                disabled={requireLocation.isPending || !locations.data?.length}
                onChange={(e) => requireLocation.mutate(e.target.checked)}
              />
              <span>Wajib berada di radius lokasi saat absensi</span>
            </label>
          </div>
        )}
        <ErrorBox
          error={
            locations.error || removeLocation.error || requireLocation.error
          }
        />
        <div className="table-scroll">
          <table>
            <thead>
              <tr>
                <th>NAMA</th>
                <th>KOORDINAT</th>
                <th>RADIUS</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {locations.data?.map((l) => (
                <tr key={l.id}>
                  <td>{l.name}</td>
                  <td>
                    {l.latitude.toFixed(5)}, {l.longitude.toFixed(5)}
                  </td>
                  <td>{l.radius_m} m</td>
                  <td>
                    <div className="row-actions">
                      <button
                        className="text-button"
                        onClick={() => setLocationForm(l)}
                      >
                        Edit
                      </button>
                      <button
                        className="icon-button"
                        aria-label={`Hapus lokasi ${l.name}`}
                        onClick={() => removeLocation.mutate(l.id)}
                      >
                        <Trash2 size={15} />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {!locations.data?.length && (
            <Empty>Belum ada lokasi. Absensi tidak memeriksa lokasi.</Empty>
          )}
        </div>
      </section>
      {shiftForm && (
        <ShiftForm
          shift={shiftForm === "new" ? undefined : shiftForm}
          close={() => setShiftForm(null)}
        />
      )}
      {locationForm && (
        <LocationForm
          location={locationForm === "new" ? undefined : locationForm}
          close={() => setLocationForm(null)}
        />
      )}
      {assign && sched.data && (
        <AssignForm
          people={sched.data}
          shifts={shifts.data ?? []}
          from={from}
          close={() => setAssign(false)}
        />
      )}
    </>
  );
}

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Clock3, MapPin } from "lucide-react";
import { ErrorBox, time, useData } from "../components/common";
import { api } from "../services/api";
import type { Attendance, AttendanceToday } from "../types";

// position asks the browser for the current location; it resolves to null
// when location is unavailable or denied.
function position(): Promise<{ latitude: number; longitude: number } | null> {
  if (!navigator.geolocation) return Promise.resolve(null);
  return new Promise((resolve) =>
    navigator.geolocation.getCurrentPosition(
      (p) =>
        resolve({ latitude: p.coords.latitude, longitude: p.coords.longitude }),
      () => resolve(null),
      { enableHighAccuracy: true, timeout: 10000, maximumAge: 30000 },
    ),
  );
}

export function ClockCard() {
  const q = useData<Attendance[]>("/attendances");
  const t = useData<AttendanceToday>("/attendance/today");
  const qc = useQueryClient();
  const s = t.data?.schedule;
  // An open attendance (possibly from yesterday's night shift) is checked
  // out first; otherwise today's finished record ends the day.
  const open = q.data?.find(
    (a) => !a.check_out && Date.now() - +new Date(a.check_in) < 20 * 3600000,
  );
  const done = !open && q.data?.find((a) => a.date === s?.date);
  const m = useMutation({
    mutationFn: async (action: string) => {
      const body = t.data?.locations ? await position() : null;
      if (!body && t.data?.require_location)
        throw new Error(
          "Izinkan akses lokasi pada browser untuk melakukan absensi.",
        );
      return api(`/attendance/${action}`, "POST", body ?? undefined);
    },
    onSuccess: () => qc.invalidateQueries(),
  });
  const current = open || done;
  return (
    <section className="clock-card">
      <div>
        <span className="eyebrow">
          KEHADIRAN HARI INI · WIB
          {s &&
            (s.off
              ? " · LIBUR"
              : ` · ${s.shift_name.toUpperCase()} ${time(s.start)}–${time(s.end)}`)}
        </span>
        <h2>
          {done
            ? "Terima kasih untuk hari ini."
            : open
              ? "Selamat berkarya!"
              : "Siap memulai hari?"}
        </h2>
        <p>
          {current
            ? `Check-in ${time(current.check_in)}${current.late_minutes ? ` (terlambat ${current.late_minutes} menit)` : ""}${current.check_out ? ` · Check-out ${time(current.check_out)}` : ""}`
            : s?.off
              ? "Hari ini tidak ada jadwal; check-in tetap tercatat."
              : "Catat kehadiran Anda sebelum memulai aktivitas."}
        </p>
        {!!t.data?.locations && (
          <p className="form-hint">
            <MapPin size={12} /> Lokasi Anda dicatat
            {t.data.require_location
              ? " dan harus berada di area kantor."
              : "."}
          </p>
        )}
      </div>
      <button
        className="primary"
        disabled={m.isPending || q.isLoading || !!q.error || !!done}
        onClick={() => m.mutate(open ? "out" : "in")}
      >
        <Clock3 size={18} />
        {m.isPending
          ? "Memproses…"
          : done
            ? "Selesai hari ini"
            : open
              ? "Check-out"
              : "Check-in sekarang"}
      </button>
      <ErrorBox error={m.error || q.error || t.error} />
    </section>
  );
}

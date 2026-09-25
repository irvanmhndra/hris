import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Clock3 } from "lucide-react";
import { ErrorBox, time, today, useData } from "../components/common";
import { api } from "../services/api";
import type { Attendance } from "../types";
export function ClockCard() {
  const q = useData<Attendance[]>("/attendances");
  const qc = useQueryClient();
  const current = q.data?.find((a) => a.date === today());
  const m = useMutation({
    mutationFn: (action: string) => api(`/attendance/${action}`, "POST"),
    onSuccess: () => qc.invalidateQueries(),
  });
  return (
    <section className="clock-card">
      <div>
        <span className="eyebrow">KEHADIRAN HARI INI · WIB</span>
        <h2>
          {current?.check_out
            ? "Terima kasih untuk hari ini."
            : current
              ? "Selamat berkarya!"
              : "Siap memulai hari?"}
        </h2>
        <p>
          {current
            ? `Check-in ${time(current.check_in)}${current.check_out ? ` · Check-out ${time(current.check_out)}` : ""}`
            : "Catat kehadiran Anda sebelum memulai aktivitas."}
        </p>
      </div>
      <button
        className="primary"
        disabled={
          m.isPending || q.isLoading || !!q.error || !!current?.check_out
        }
        onClick={() => m.mutate(current ? "out" : "in")}
      >
        <Clock3 size={18} />
        {m.isPending
          ? "Memproses…"
          : current?.check_out
            ? "Selesai hari ini"
            : current
              ? "Check-out"
              : "Check-in sekarang"}
      </button>
      <ErrorBox error={m.error || q.error} />
    </section>
  );
}

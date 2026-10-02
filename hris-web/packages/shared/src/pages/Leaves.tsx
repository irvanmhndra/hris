import { useMutation, useQueryClient } from "@tanstack/react-query";
import { CalendarDays, Check, Plus, X } from "lucide-react";
import { useState } from "react";
import {
  Badge,
  date,
  Empty,
  ErrorBox,
  initials,
  kinds,
  Loading,
  Modal,
  useData,
  today,
} from "../components/common";
import { Heading } from "../components/Heading";
import { api } from "../services/api";
import { BalanceSummary } from "./HRSettings";
import type { Leave } from "../types";
export function Leaves({ employee }: { employee: boolean }) {
  const q = useData<Leave[]>("/leaves");
  const [filter, setFilter] = useState("");
  const [open, setOpen] = useState(false);
  const [decision, setDecision] = useState<{
    leave: Leave;
    status: string;
  } | null>(null);
  const qc = useQueryClient();
  const create = useMutation({
    mutationFn: (v: unknown) => api("/leaves", "POST", v),
    onSuccess: () => {
      qc.invalidateQueries();
      setOpen(false);
    },
  });
  const cancel = useMutation({
    mutationFn: (id: number) => api(`/leaves/${id}/cancel`, "POST"),
    onSuccess: () => {
      qc.invalidateQueries();
      setDecision(null);
    },
  });
  const review = useMutation({
    mutationFn: ({ id, status }: { id: number; status: string }) =>
      api(`/leaves/${id}`, "PATCH", { status }),
    onSuccess: () => {
      qc.invalidateQueries();
      setDecision(null);
    },
  });
  const rows = q.data?.filter((l) => !filter || l.status === filter) || [];
  return (
    <>
      <Heading
        eyebrow="TIME TO RECHARGE"
        title={employee ? "Ruang untuk kehidupan lainnya." : "Cuti & izin tim."}
        description={
          employee
            ? "Ajukan waktu istirahat dan pantau persetujuannya di sini."
            : "Tinjau pengajuan dan bantu tim menjaga keseimbangan."
        }
      >
        {employee && (
          <button
            className="primary"
            onClick={() => {
              create.reset();
              setOpen(true);
            }}
          >
            <Plus size={17} />
            Ajukan cuti
          </button>
        )}
      </Heading>
      {employee && <BalanceSummary />}
      <section className="panel">
        <div className="tabs">
          {[
            ["", "Semua"],
            ["pending", "Menunggu"],
            ["approved", "Disetujui"],
            ["rejected", "Ditolak"],
            ["cancelled", "Dibatalkan"],
          ].map(([v, l]) => (
            <button
              className={v === filter ? "active" : ""}
              key={v}
              onClick={() => setFilter(v)}
            >
              {l}
              <span>
                {q.data?.filter((x) => !v || x.status === v).length || 0}
              </span>
            </button>
          ))}
        </div>
        <ErrorBox error={q.error} />
        {q.isLoading ? (
          <Loading />
        ) : (
          <div className="table-scroll">
            <table>
              <thead>
                <tr>
                  <th>{employee ? "JENIS CUTI" : "KARYAWAN"}</th>
                  <th>PERIODE</th>
                  <th>DURASI</th>
                  <th>ALASAN</th>
                  <th>STATUS</th>
                  <th>TINDAKAN</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((l) => (
                  <tr key={l.id}>
                    <td>
                      <div className="person">
                        <span className="avatar lilac">
                          {employee ? (
                            <CalendarDays size={18} />
                          ) : (
                            initials(l.name)
                          )}
                        </span>
                        <div>
                          <strong>{employee ? kinds[l.kind] : l.name}</strong>
                          {!employee && <small>{kinds[l.kind]}</small>}
                        </div>
                      </div>
                    </td>
                    <td>
                      {date(l.start_date)}
                      <small className="block">s.d. {date(l.end_date)}</small>
                    </td>
                    <td>
                      {l.days} hari
                      <small className="block">
                        {l.calculation === "legacy_calendar_days"
                          ? "Perhitungan lama"
                          : "Hari kerja"}
                      </small>
                    </td>
                    <td className="reason">{l.reason}</td>
                    <td>
                      <Badge status={l.status} />
                      {l.status === "pending" && (
                        <small className="block">
                          {l.stage === "manager"
                            ? "Menunggu atasan"
                            : "Menunggu HR"}
                        </small>
                      )}
                      {l.status === "rejected" && l.review_note && (
                        <small className="block">{l.review_note}</small>
                      )}
                    </td>
                    {employee && (
                      <td>
                        {(l.status === "pending" ||
                          (l.status === "approved" &&
                            l.start_date > today())) && (
                          <button
                            className="text-button"
                            onClick={() => {
                              cancel.reset();
                              setDecision({ leave: l, status: "cancelled" });
                            }}
                          >
                            Batalkan
                          </button>
                        )}
                      </td>
                    )}
                    {!employee && (
                      <td>
                        {l.status === "pending" ? (
                          <div className="row-actions">
                            <button
                              className="approve"
                              aria-label={`Setujui cuti ${l.name}`}
                              onClick={() => {
                                review.reset();
                                setDecision({ leave: l, status: "approved" });
                              }}
                            >
                              <Check size={17} />
                            </button>
                            <button
                              className="reject"
                              aria-label={`Tolak cuti ${l.name}`}
                              onClick={() => {
                                review.reset();
                                setDecision({ leave: l, status: "rejected" });
                              }}
                            >
                              <X size={17} />
                            </button>
                          </div>
                        ) : (
                          "—"
                        )}
                      </td>
                    )}
                  </tr>
                ))}
              </tbody>
            </table>
            {!rows.length && (
              <Empty>Belum ada pengajuan pada status ini.</Empty>
            )}
          </div>
        )}
      </section>
      {open && (
        <Modal title="Pengajuan cuti & izin" close={() => setOpen(false)}>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              create.mutate(Object.fromEntries(new FormData(e.currentTarget)));
            }}
          >
            <label>
              Jenis pengajuan
              <select name="kind">
                {Object.entries(kinds).map(([k, v]) => (
                  <option key={k} value={k}>
                    {v}
                  </option>
                ))}
              </select>
            </label>
            <div className="form-grid">
              <label>
                Tanggal mulai
                <input type="date" name="start_date" required />
              </label>
              <label>
                Tanggal selesai
                <input type="date" name="end_date" required />
              </label>
            </div>
            <label>
              Alasan
              <textarea
                name="reason"
                minLength={5}
                maxLength={1000}
                required
                rows={4}
                placeholder="Ceritakan keperluan Anda…"
              />
            </label>
            <p className="form-hint">
              Durasi mengikuti hari kerja dan kalender libur perusahaan. Cuti
              tahunan yang diajukan mencadangkan saldo sampai disetujui,
              ditolak, atau dibatalkan. Cuti tidak dibayar tidak memakai saldo
              dan mengurangi gaji sesuai hari kerja. Bila Anda memiliki atasan,
              pengajuan disetujui atasan terlebih dahulu, lalu HR.
            </p>
            <ErrorBox error={create.error} />
            <div className="modal-actions">
              <button className="primary" disabled={create.isPending}>
                Kirim pengajuan
              </button>
            </div>
          </form>
        </Modal>
      )}
      {decision && (
        <Modal
          title={
            decision.status === "cancelled"
              ? "Batalkan pengajuan?"
              : decision.status === "approved"
                ? "Setujui pengajuan?"
                : "Tolak pengajuan?"
          }
          close={() => setDecision(null)}
        >
          <p>
            {decision.leave.name} · {kinds[decision.leave.kind]} ·{" "}
            {decision.leave.days} hari
          </p>
          <p>
            {date(decision.leave.start_date)} – {date(decision.leave.end_date)}
          </p>
          <ErrorBox error={review.error || cancel.error} />
          <div className="modal-actions">
            <button className="secondary" onClick={() => setDecision(null)}>
              Batal
            </button>
            <button
              className="primary"
              disabled={review.isPending || cancel.isPending}
              onClick={() =>
                decision.status === "cancelled"
                  ? cancel.mutate(decision.leave.id)
                  : review.mutate({
                      id: decision.leave.id,
                      status: decision.status,
                    })
              }
            >
              Konfirmasi keputusan
            </button>
          </div>
        </Modal>
      )}
    </>
  );
}

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { Check, X } from "lucide-react";
import { Heading } from "../components/Heading";
import {
  date,
  Empty,
  ErrorBox,
  initials,
  kinds,
  Loading,
  Modal,
  time,
  useData,
} from "../components/common";
import { api } from "../services/api";
import type { TeamRequest } from "../types";

const types: Record<TeamRequest["type"], string> = {
  leave: "Cuti",
  overtime: "Lembur",
  corrections: "Koreksi absensi",
};

function detail(r: TeamRequest) {
  if (r.type === "leave")
    return `${kinds[r.kind] || r.kind} · ${date(r.start_date!)} – ${date(r.end_date!)}`;
  if (r.type === "overtime")
    return `${date(r.data.start_at)} · ${time(r.data.start_at)}–${time(r.data.end_at)}`;
  return `${date(r.data.date)} · ${r.data.check_in}–${r.data.check_out}`;
}

// TeamApprovals is the manager's queue: requests from direct reports that
// need a first decision before HR sees them.
export function TeamApprovals() {
  const q = useData<TeamRequest[]>("/team/approvals");
  const qc = useQueryClient();
  const [decision, setDecision] = useState<{
    request: TeamRequest;
    action: "approve" | "reject";
  } | null>(null);
  const m = useMutation({
    mutationFn: (note: string) =>
      api(
        `/team/approvals/${decision!.request.type}/${decision!.request.id}`,
        "POST",
        { action: decision!.action, note, version: decision!.request.version },
      ),
    onSuccess: () => {
      qc.invalidateQueries();
      setDecision(null);
    },
  });
  return (
    <>
      <Heading
        eyebrow="TEAM APPROVALS"
        title="Keputusan pertama ada di Anda."
        description="Pengajuan tim langsung Anda. Setujui untuk meneruskan ke HR, atau tolak dengan alasan."
      />
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
                  <th>PENGAJUAN</th>
                  <th>ALASAN</th>
                  <th>TINDAKAN</th>
                </tr>
              </thead>
              <tbody>
                {q.data?.map((r) => (
                  <tr key={`${r.type}-${r.id}`}>
                    <td>
                      <div className="person">
                        <span className="avatar lilac">
                          {initials(r.employee_name)}
                        </span>
                        <div>
                          <strong>{r.employee_name}</strong>
                          <small>{types[r.type]}</small>
                        </div>
                      </div>
                    </td>
                    <td>
                      {r.title && <strong className="block">{r.title}</strong>}
                      {detail(r)}
                    </td>
                    <td className="reason">{r.reason}</td>
                    <td>
                      <div className="row-actions">
                        <button
                          className="approve"
                          aria-label={`Teruskan ${types[r.type]} ${r.employee_name} ke HR`}
                          onClick={() => {
                            m.reset();
                            setDecision({ request: r, action: "approve" });
                          }}
                        >
                          <Check size={17} />
                        </button>
                        <button
                          className="reject"
                          aria-label={`Tolak ${types[r.type]} ${r.employee_name}`}
                          onClick={() => {
                            m.reset();
                            setDecision({ request: r, action: "reject" });
                          }}
                        >
                          <X size={17} />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            {!q.data?.length && (
              <Empty>
                Tidak ada pengajuan tim yang menunggu keputusan Anda.
              </Empty>
            )}
          </div>
        )}
      </section>
      {decision && (
        <Modal
          title={
            decision.action === "approve"
              ? "Setujui dan teruskan ke HR?"
              : "Tolak pengajuan?"
          }
          close={() => setDecision(null)}
        >
          <form
            onSubmit={(e) => {
              e.preventDefault();
              m.mutate(String(new FormData(e.currentTarget).get("note") || ""));
            }}
          >
            <p className="modal-context">
              {decision.request.employee_name} · {types[decision.request.type]}{" "}
              · {detail(decision.request)}
            </p>
            <label>
              {decision.action === "reject"
                ? "Alasan penolakan"
                : "Catatan (opsional)"}
              <textarea
                name="note"
                rows={3}
                maxLength={1000}
                minLength={decision.action === "reject" ? 5 : undefined}
                required={decision.action === "reject"}
              />
            </label>
            <ErrorBox error={m.error} />
            <div className="modal-actions">
              <button
                type="button"
                className="secondary"
                onClick={() => setDecision(null)}
              >
                Batal
              </button>
              <button className="primary" disabled={m.isPending}>
                Konfirmasi
              </button>
            </div>
          </form>
        </Modal>
      )}
    </>
  );
}

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { Plus, ArrowLeft, Download } from "lucide-react";
import { Heading } from "../components/Heading";
import {
  Badge,
  Empty,
  ErrorBox,
  Loading,
  Modal,
  rupiah,
  today,
  useData,
} from "../components/common";
import { api } from "../services/api";
import type { Salary, PayrollRun, Payslip } from "../types";
import { SlipModal } from "./Payslips";
type Components = {
  basic_salary: number;
  allowance: number;
  deduction: number;
  note: string;
  version?: number;
};
function SalaryForm({
  title,
  path,
  initial,
  close,
}: {
  title: string;
  path: string;
  initial: Components;
  close: () => void;
}) {
  const qc = useQueryClient();
  const m = useMutation({
    mutationFn: (v: unknown) => api(path, "PUT", v),
    onSuccess: () => {
      qc.invalidateQueries();
      close();
    },
  });
  const [basic, setBasic] = useState(initial.basic_salary);
  const [allowance, setAllowance] = useState(initial.allowance);
  const [deduction, setDeduction] = useState(initial.deduction);
  return (
    <Modal title={title} close={close}>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          m.mutate({
            basic_salary: basic,
            allowance,
            deduction,
            note: new FormData(e.currentTarget).get("note"),
            version: initial.version || 0,
          });
        }}
      >
        <div className="form-grid">
          <label>
            Gaji pokok (IDR)
            <input
              type="number"
              min={0}
              max={1000000000000}
              step={1}
              value={basic}
              onChange={(e) => setBasic(Number(e.target.value))}
              required
            />
          </label>
          <label>
            Total tunjangan (IDR)
            <input
              type="number"
              min={0}
              max={1000000000000}
              step={1}
              value={allowance}
              onChange={(e) => setAllowance(Number(e.target.value))}
              required
            />
          </label>
          <label>
            Total potongan (IDR)
            <input
              type="number"
              min={0}
              max={Math.min(1000000000000, basic + allowance)}
              step={1}
              value={deduction}
              onChange={(e) => setDeduction(Number(e.target.value))}
              required
            />
          </label>
        </div>
        <label>
          Rincian komponen / catatan
          <textarea
            name="note"
            defaultValue={initial.note}
            maxLength={1000}
            rows={4}
            placeholder="Contoh: tunjangan transport, bonus, potongan pajak/BPJS yang telah dihitung HR"
          />
        </label>
        <div className="payroll-total">
          <span>Gaji bersih</span>
          <strong>{rupiah(basic + allowance - deduction)}</strong>
        </div>
        <p className="form-hint">
          Nominal rupiah bulat. Pajak, BPJS, prorata, dan pembayaran lembur
          dimasukkan oleh HR; belum dihitung otomatis.
        </p>
        <ErrorBox error={m.error} />
        <div className="modal-actions">
          <button type="button" className="secondary" onClick={close}>
            Batal
          </button>
          <button
            className="primary"
            disabled={m.isPending || deduction > basic + allowance}
          >
            Simpan komponen
          </button>
        </div>
      </form>
    </Modal>
  );
}
export function Salaries() {
  const q = useData<Salary[]>("/salaries");
  const [edit, setEdit] = useState<Salary | null>(null);
  return (
    <>
      <Heading
        eyebrow="COMPENSATION SETUP"
        title="Komponen gaji yang jelas."
        description="Tetapkan komponen bulanan sebelum membuat draft payroll."
      />
      <div className="notice">
        Komponen ini menjadi nilai awal saat payroll dibuat. Perubahan
        berikutnya tidak mengubah slip yang sudah tersimpan.
      </div>
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
                  <th>GAJI POKOK</th>
                  <th>TUNJANGAN</th>
                  <th>POTONGAN</th>
                  <th>BERSIH</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {q.data?.map((s) => (
                  <tr key={s.employee_id}>
                    <td>
                      <strong>{s.name}</strong>
                      <small className="block">
                        {s.code}
                        {!s.configured ? " · Belum diatur" : ""}
                      </small>
                    </td>
                    <td>{rupiah(s.basic_salary)}</td>
                    <td>{rupiah(s.allowance)}</td>
                    <td>{rupiah(s.deduction)}</td>
                    <td>
                      <strong>
                        {rupiah(s.basic_salary + s.allowance - s.deduction)}
                      </strong>
                    </td>
                    <td>
                      <button
                        className="text-button"
                        onClick={() => setEdit(s)}
                      >
                        Atur gaji
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            {!q.data?.length && <Empty />}
          </div>
        )}
      </section>
      {edit && (
        <SalaryForm
          title={`Komponen gaji · ${edit.name}`}
          path={`/salaries/${edit.employee_id}`}
          initial={edit}
          close={() => setEdit(null)}
        />
      )}
    </>
  );
}
export function Payroll() {
  const q = useData<PayrollRun[]>("/payroll");
  const qc = useQueryClient();
  const [selected, setSelected] = useState<number | null>(null);
  const [open, setOpen] = useState(false);
  const [action, setAction] = useState<{
    run: PayrollRun;
    action: string;
  } | null>(null);
  const create = useMutation({
    mutationFn: (period: string) =>
      api<{ id: number }>("/payroll", "POST", { period }),
    onSuccess: (d) => {
      qc.invalidateQueries();
      setOpen(false);
      setSelected(d.id);
    },
  });
  const change = useMutation({
    mutationFn: (reference: string) =>
      api(`/payroll/${action!.run.id}/action`, "PATCH", {
        action: action!.action,
        reference,
      }),
    onSuccess: () => {
      qc.invalidateQueries();
      setAction(null);
    },
  });
  const selectedRun = q.data?.find((r) => r.id === selected);
  return (
    <>
      <Heading
        eyebrow="MONTHLY PAYROLL"
        title={
          selectedRun
            ? `Payroll ${selectedRun.period}`
            : "Gaji terhitung. Tim terjaga."
        }
        description="Siapkan draft, verifikasi komponen, lalu terbitkan slip gaji."
      >
        {selected ? (
          <button className="secondary" onClick={() => setSelected(null)}>
            <ArrowLeft size={15} />
            Semua periode
          </button>
        ) : (
          <button
            className="primary"
            onClick={() => {
              create.reset();
              setOpen(true);
            }}
          >
            <Plus size={15} />
            Buat payroll
          </button>
        )}
      </Heading>
      <ErrorBox error={q.error} />
      {selected && selectedRun ? (
        <>
          <div className="payroll-run-summary">
            <div>
              <Badge status={selectedRun.status} />
              <h2>{rupiah(selectedRun.total)}</h2>
              <p>
                {selectedRun.employees} karyawan ·{" "}
                {selectedRun.payment_reference ||
                  "Belum ada referensi pembayaran"}
              </p>
            </div>
            <div className="heading-actions">
              {selectedRun.status === "draft" && (
                <>
                  <button
                    className="secondary"
                    onClick={() => {
                      change.reset();
                      setAction({ run: selectedRun, action: "void" });
                    }}
                  >
                    Batalkan draft
                  </button>
                  <button
                    className="primary"
                    onClick={() => {
                      change.reset();
                      setAction({ run: selectedRun, action: "finalize" });
                    }}
                  >
                    Finalisasi payroll
                  </button>
                </>
              )}
              {selectedRun.status === "finalized" && (
                <button
                  className="primary"
                  onClick={() => {
                    change.reset();
                    setAction({ run: selectedRun, action: "paid" });
                  }}
                >
                  Catat sudah dibayar
                </button>
              )}
            </div>
          </div>
          <PayrollEntries run={selectedRun} />
        </>
      ) : q.isLoading ? (
        <Loading />
      ) : (
        <section className="panel">
          <div className="table-scroll">
            <table>
              <thead>
                <tr>
                  <th>PERIODE</th>
                  <th>KARYAWAN</th>
                  <th>TOTAL GAJI BERSIH</th>
                  <th>STATUS</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {q.data?.map((r) => (
                  <tr key={r.id}>
                    <td>
                      <strong>{r.period}</strong>
                    </td>
                    <td>{r.employees}</td>
                    <td>{rupiah(r.total)}</td>
                    <td>
                      <Badge status={r.status} />
                    </td>
                    <td>
                      <button
                        className="text-button"
                        onClick={() => setSelected(r.id)}
                      >
                        Lihat payroll
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            {!q.data?.length && (
              <Empty>
                Belum ada payroll. Atur komponen gaji, lalu buat periode
                pertama.
              </Empty>
            )}
          </div>
        </section>
      )}
      {open && (
        <Modal title="Buat draft payroll" close={() => setOpen(false)}>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              create.mutate(
                String(new FormData(e.currentTarget).get("period")),
              );
            }}
          >
            <label>
              Periode gaji
              <input
                name="period"
                type="month"
                defaultValue={today().slice(0, 7)}
                min="2000-01"
                max="2200-12"
                required
              />
            </label>
            <p className="notice">
              Seluruh karyawan aktif yang sudah bergabung pada periode ini akan
              disertakan. Komponen gaji harus lengkap. Nilai disalin sebagai
              snapshot dan dapat disesuaikan selama draft.
            </p>
            <ErrorBox error={create.error} />
            <div className="modal-actions">
              <button className="primary" disabled={create.isPending}>
                Buat draft
              </button>
            </div>
          </form>
        </Modal>
      )}
      {action && (
        <Modal
          title={
            action.action === "paid"
              ? "Catat pembayaran manual?"
              : action.action === "void"
                ? "Batalkan draft payroll?"
                : "Finalisasi dan terbitkan slip?"
          }
          close={() => setAction(null)}
        >
          <form
            onSubmit={(e) => {
              e.preventDefault();
              change.mutate(
                String(new FormData(e.currentTarget).get("reference") || ""),
              );
            }}
          >
            <p className="modal-context">
              Periode {action.run.period} · {action.run.employees} karyawan ·{" "}
              {rupiah(action.run.total)}
            </p>
            {action.action === "paid" ? (
              <>
                <p className="notice">
                  Tindakan ini hanya mencatat pembayaran yang telah Anda lakukan
                  di luar aplikasi. Tidak ada transfer dana dari HRIS.
                </p>
                <label>
                  Referensi pembayaran
                  <input
                    name="reference"
                    minLength={5}
                    maxLength={150}
                    required
                    placeholder="Referensi batch / bukti transfer"
                  />
                </label>
              </>
            ) : action.action === "finalize" ? (
              <p className="notice">
                Setelah finalisasi, nominal terkunci dan slip dapat dilihat oleh
                masing-masing karyawan. Pastikan pajak, BPJS, prorata, dan
                penyesuaian lain sudah diverifikasi.
              </p>
            ) : (
              <p className="notice">
                Draft dibatalkan dan tidak ditampilkan kepada karyawan. Anda
                dapat membuat payroll baru untuk periode yang sama.
              </p>
            )}
            <ErrorBox error={change.error} />
            <div className="modal-actions">
              <button
                type="button"
                className="secondary"
                onClick={() => setAction(null)}
              >
                Kembali
              </button>
              <button className="primary" disabled={change.isPending}>
                Konfirmasi
              </button>
            </div>
          </form>
        </Modal>
      )}
    </>
  );
}
function PayrollEntries({ run }: { run: PayrollRun }) {
  const q = useData<Payslip[]>(`/payroll/${run.id}/slips`);
  const [edit, setEdit] = useState<Payslip | null>(null);
  const [slip, setSlip] = useState<Payslip | null>(null);
  function csv() {
    const cell = (v: string | number) =>
      '"' +
      String(v)
        .replace(/^[=+@\-\t\r]/, "'$&")
        .replaceAll('"', '""') +
      '"';
    const text = [
      ["NIK", "Nama", "Gaji pokok", "Tunjangan", "Potongan", "Gaji bersih"],
      ...(q.data || []).map((s) => [
        s.employee_code,
        s.employee_name,
        s.basic_salary,
        s.allowance,
        s.deduction,
        s.net,
      ]),
    ]
      .map((row) => row.map(cell).join(","))
      .join("\r\n");
    const url = URL.createObjectURL(
      new Blob(["\uFEFF" + text], { type: "text/csv;charset=utf-8" }),
    );
    const a = document.createElement("a");
    a.href = url;
    a.download = `payroll-${run.period}-${run.status}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  }
  return (
    <section className="panel">
      <div className="panel-head">
        <div>
          <h2>Rincian slip gaji</h2>
          <p>Snapshot komponen pada periode ini.</p>
        </div>
        <button className="secondary" onClick={csv} disabled={!q.data?.length}>
          <Download size={15} />
          Ekspor CSV
        </button>
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
                <th>GAJI POKOK</th>
                <th>TUNJANGAN</th>
                <th>POTONGAN</th>
                <th>BERSIH</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {q.data?.map((s) => (
                <tr key={s.id}>
                  <td>
                    {s.employee_name}
                    <small className="block">{s.employee_code}</small>
                  </td>
                  <td>{rupiah(s.basic_salary)}</td>
                  <td>{rupiah(s.allowance)}</td>
                  <td>{rupiah(s.deduction)}</td>
                  <td>
                    <strong>{rupiah(s.net)}</strong>
                  </td>
                  <td>
                    <div className="row-actions">
                      {run.status === "draft" && (
                        <button
                          className="text-button"
                          onClick={() => setEdit(s)}
                        >
                          Edit
                        </button>
                      )}
                      <button
                        className="text-button"
                        onClick={() => setSlip(s)}
                      >
                        Slip
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {edit && (
        <SalaryForm
          title={`Penyesuaian · ${edit.employee_name}`}
          path={`/payroll/slips/${edit.id}`}
          initial={edit}
          close={() => setEdit(null)}
        />
      )}{" "}
      {slip && <SlipModal slip={slip} close={() => setSlip(null)} />}
    </section>
  );
}

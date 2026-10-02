import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { Plus, ArrowLeft, Download, Settings2, Trash2 } from "lucide-react";
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
import type {
  Salary,
  SalaryComponent,
  PayrollRun,
  PayrollSettings,
  Payslip,
  PayrollLine,
} from "../types";
import { SlipModal } from "./Payslips";

const MAX = 1000000000000;
const PTKP = ["TK/0", "TK/1", "TK/2", "TK/3", "K/0", "K/1", "K/2", "K/3"];
const TAX_METHODS: Record<string, string> = {
  gross: "Gross — dipotong dari gaji",
  gross_up: "Gross-up — ditanggung perusahaan",
  none: "Manual — tidak dihitung otomatis",
};
const JKK_RATES: [number, string][] = [
  [24, "Kelas I · 0,24%"],
  [54, "Kelas II · 0,54%"],
  [89, "Kelas III · 0,89%"],
  [127, "Kelas IV · 1,27%"],
  [174, "Kelas V · 1,74%"],
];
// Line codes HR may edit on a draft slip; BPJS and PPh 21 are recalculated.
const INPUT_CODES = [
  "BASIC",
  "ALLOWANCE",
  "OVERTIME",
  "THR",
  "ADJUSTMENT",
  "DEDUCTION",
  "LATE",
];

type Row = {
  key: number;
  kind: string;
  code: string;
  name: string;
  amount: number;
  fixed: boolean;
  taxable: boolean;
};
let nextKey = 1;
const toRow = (v: Omit<Row, "key" | "code"> & { code?: string }): Row => ({
  key: nextKey++,
  code: "",
  ...v,
});
const isEarning = (kind: string) => kind === "allowance" || kind === "earning";

function Toggle({
  label,
  checked,
  onChange,
  disabled,
}: {
  label: string;
  checked: boolean;
  onChange: (v: boolean) => void;
  disabled?: boolean;
}) {
  return (
    <label>
      <input
        type="checkbox"
        checked={checked}
        disabled={disabled}
        onChange={(e) => onChange(e.target.checked)}
      />
      <span>{label}</span>
    </label>
  );
}

function LineEditor({
  rows,
  setRows,
  addEarning,
  addDeduction,
}: {
  rows: Row[];
  setRows: (rows: Row[]) => void;
  addEarning: { label: string; row: () => Row };
  addDeduction: { label: string; row: () => Row };
}) {
  const update = (key: number, patch: Partial<Row>) =>
    setRows(rows.map((r) => (r.key === key ? { ...r, ...patch } : r)));
  return (
    <div className="line-editor">
      {rows.map((r) => (
        <div className="line-row" key={r.key}>
          <input
            aria-label="Nama komponen"
            value={r.name}
            maxLength={80}
            required
            onChange={(e) => update(r.key, { name: e.target.value })}
          />
          <input
            aria-label={`Nominal ${r.name}`}
            type="number"
            min={0}
            max={MAX}
            step={1}
            required
            value={r.amount}
            onChange={(e) => update(r.key, { amount: Number(e.target.value) })}
          />
          <div className="weekday-picker">
            {isEarning(r.kind) ? (
              <>
                <Toggle
                  label="Tetap"
                  checked={r.fixed}
                  onChange={(fixed) => update(r.key, { fixed })}
                />
                <Toggle
                  label="Kena pajak"
                  checked={r.taxable}
                  onChange={(taxable) => update(r.key, { taxable })}
                />
              </>
            ) : (
              <span className="line-kind">Potongan</span>
            )}
          </div>
          <button
            type="button"
            className="icon-button"
            aria-label={`Hapus ${r.name}`}
            onClick={() => setRows(rows.filter((x) => x.key !== r.key))}
          >
            <Trash2 size={15} />
          </button>
        </div>
      ))}
      <div className="row-actions">
        <button
          type="button"
          className="text-button"
          onClick={() => setRows([...rows, addEarning.row()])}
        >
          + {addEarning.label}
        </button>
        <button
          type="button"
          className="text-button"
          onClick={() => setRows([...rows, addDeduction.row()])}
        >
          + {addDeduction.label}
        </button>
      </div>
    </div>
  );
}

function SalaryForm({ salary, close }: { salary: Salary; close: () => void }) {
  const qc = useQueryClient();
  const m = useMutation({
    mutationFn: (v: unknown) =>
      api(`/salaries/${salary.employee_id}`, "PUT", v),
    onSuccess: () => {
      qc.invalidateQueries();
      close();
    },
  });
  const [basic, setBasic] = useState(salary.basic_salary);
  const [ptkp, setPtkp] = useState(salary.ptkp_status);
  const [method, setMethod] = useState(salary.tax_method);
  const [kes, setKes] = useState(salary.bpjs_kesehatan);
  const [tk, setTk] = useState(salary.bpjs_ketenagakerjaan);
  const [jp, setJp] = useState(salary.bpjs_pensiun);
  const [overtime, setOvertime] = useState(salary.overtime_eligible);
  const [rows, setRows] = useState<Row[]>(() =>
    salary.components.map((c) => toRow(c)),
  );
  const earnings =
    basic +
    rows.filter((r) => isEarning(r.kind)).reduce((s, r) => s + r.amount, 0);
  const deductions = rows
    .filter((r) => !isEarning(r.kind))
    .reduce((s, r) => s + r.amount, 0);
  return (
    <Modal title={`Komponen gaji · ${salary.name}`} close={close}>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          m.mutate({
            basic_salary: basic,
            ptkp_status: ptkp,
            tax_method: method,
            bpjs_kesehatan: kes,
            bpjs_ketenagakerjaan: tk,
            bpjs_pensiun: tk && jp,
            overtime_eligible: overtime,
            ...Object.fromEntries(
              [
                "nik",
                "npwp",
                "bpjs_kesehatan_number",
                "bpjs_ketenagakerjaan_number",
              ].map((k) => [k, new FormData(e.currentTarget).get(k)]),
            ),
            note: new FormData(e.currentTarget).get("note"),
            components: rows.map((r): SalaryComponent => ({
              kind: r.kind as SalaryComponent["kind"],
              name: r.name,
              amount: r.amount,
              fixed: r.fixed,
              taxable: r.taxable,
            })),
          });
        }}
      >
        <div className="form-grid">
          <label className="full">
            Gaji pokok bulanan (IDR)
            <input
              type="number"
              min={0}
              max={MAX}
              step={1}
              value={basic}
              onChange={(e) => setBasic(Number(e.target.value))}
              required
            />
          </label>
          <label>
            Status PTKP
            <select value={ptkp} onChange={(e) => setPtkp(e.target.value)}>
              {PTKP.map((p) => (
                <option key={p}>{p}</option>
              ))}
            </select>
          </label>
          <label>
            Metode PPh 21
            <select
              value={method}
              onChange={(e) =>
                setMethod(e.target.value as Salary["tax_method"])
              }
            >
              {Object.entries(TAX_METHODS).map(([v, l]) => (
                <option key={v} value={v}>
                  {l}
                </option>
              ))}
            </select>
          </label>
        </div>
        <div className="field-label">Kepesertaan</div>
        <div className="weekday-picker">
          <Toggle label="BPJS Kesehatan" checked={kes} onChange={setKes} />
          <Toggle
            label="BPJS Ketenagakerjaan"
            checked={tk}
            onChange={(v) => {
              setTk(v);
              if (!v) setJp(false);
            }}
          />
          <Toggle
            label="Jaminan Pensiun"
            checked={tk && jp}
            disabled={!tk}
            onChange={setJp}
          />
          <Toggle
            label="Dapat lembur"
            checked={overtime}
            onChange={setOvertime}
          />
        </div>
        <div className="form-grid">
          <label>
            NIK (KTP)
            <input
              name="nik"
              defaultValue={salary.nik}
              inputMode="numeric"
              pattern="[0-9]{16}"
              title="16 digit"
            />
          </label>
          <label>
            NPWP
            <input
              name="npwp"
              defaultValue={salary.npwp}
              inputMode="numeric"
              pattern="[0-9.\-]{15,20}"
              title="15/16 digit"
            />
          </label>
          <label>
            No. BPJS Kesehatan
            <input
              name="bpjs_kesehatan_number"
              defaultValue={salary.bpjs_kesehatan_number}
              inputMode="numeric"
            />
          </label>
          <label>
            No. BPJS Ketenagakerjaan
            <input
              name="bpjs_ketenagakerjaan_number"
              defaultValue={salary.bpjs_ketenagakerjaan_number}
              inputMode="numeric"
            />
          </label>
        </div>
        <div className="field-label">Tunjangan dan potongan rutin</div>
        <LineEditor
          rows={rows}
          setRows={setRows}
          addEarning={{
            label: "Tunjangan",
            row: () =>
              toRow({
                kind: "allowance",
                name: "",
                amount: 0,
                fixed: false,
                taxable: true,
              }),
          }}
          addDeduction={{
            label: "Potongan",
            row: () =>
              toRow({
                kind: "deduction",
                name: "",
                amount: 0,
                fixed: false,
                taxable: true,
              }),
          }}
        />
        <label>
          Catatan
          <textarea
            name="note"
            defaultValue={salary.note}
            maxLength={1000}
            rows={2}
          />
        </label>
        <div className="payroll-total">
          <span>
            Bruto {rupiah(earnings)} · potongan rutin {rupiah(deductions)}
          </span>
          <strong>{rupiah(earnings - deductions)}</strong>
        </div>
        <p className="form-hint">
          Tunjangan <b>tetap</b> menjadi dasar BPJS, upah lembur (1/173), dan
          THR. BPJS dan PPh 21 (TER) dihitung otomatis saat payroll dibuat; gaji
          pokok dan tunjangan diprorata untuk karyawan yang masuk atau keluar di
          tengah periode.
        </p>
        <ErrorBox error={m.error} />
        <div className="modal-actions">
          <button type="button" className="secondary" onClick={close}>
            Batal
          </button>
          <button
            className="primary"
            disabled={m.isPending || deductions > earnings}
          >
            Simpan komponen
          </button>
        </div>
      </form>
    </Modal>
  );
}

function SettingsForm({ close }: { close: () => void }) {
  const q = useData<PayrollSettings>("/payroll/settings");
  const qc = useQueryClient();
  const m = useMutation({
    mutationFn: (v: PayrollSettings) => api("/payroll/settings", "PUT", v),
    onSuccess: () => {
      qc.invalidateQueries();
      close();
    },
  });
  return (
    <Modal title="Pengaturan BPJS perusahaan" close={close}>
      <ErrorBox error={q.error} />
      {q.data ? (
        <form
          onSubmit={(e) => {
            e.preventDefault();
            const d = new FormData(e.currentTarget);
            m.mutate({
              jkk_rate: Number(d.get("jkk_rate")),
              jp_wage_cap: Number(d.get("jp_wage_cap")),
              kes_wage_cap: Number(d.get("kes_wage_cap")),
              late_deduction: String(
                d.get("late_deduction"),
              ) as PayrollSettings["late_deduction"],
              late_deduction_amount: Number(
                d.get("late_deduction_amount") || 0,
              ),
              deduct_absence: d.get("deduct_absence") === "on",
            });
          }}
        >
          <label>
            Kelas risiko JKK
            <select name="jkk_rate" defaultValue={q.data.jkk_rate}>
              {JKK_RATES.map(([v, l]) => (
                <option key={v} value={v}>
                  {l}
                </option>
              ))}
            </select>
          </label>
          <div className="form-grid">
            <label>
              Batas upah Jaminan Pensiun
              <input
                name="jp_wage_cap"
                type="number"
                min={0}
                max={MAX}
                defaultValue={q.data.jp_wage_cap}
                required
              />
            </label>
            <label>
              Batas upah BPJS Kesehatan
              <input
                name="kes_wage_cap"
                type="number"
                min={0}
                max={MAX}
                defaultValue={q.data.kes_wage_cap}
                required
              />
            </label>
          </div>
          <div className="form-grid">
            <label>
              Potongan keterlambatan
              <select
                name="late_deduction"
                defaultValue={q.data.late_deduction}
              >
                <option value="none">Tidak dipotong</option>
                <option value="per_minute">
                  Per menit (upah 1/173 per jam)
                </option>
                <option value="per_occurrence">
                  Nominal per keterlambatan
                </option>
              </select>
            </label>
            <label>
              Nominal per keterlambatan
              <input
                name="late_deduction_amount"
                type="number"
                min={0}
                max={1000000000}
                defaultValue={q.data.late_deduction_amount}
              />
            </label>
          </div>
          <div className="weekday-picker">
            <label>
              <input
                type="checkbox"
                name="deduct_absence"
                defaultChecked={q.data.deduct_absence}
              />
              <span>Potong gaji hari tanpa keterangan (prorata)</span>
            </label>
          </div>
          <p className="form-hint">
            Keterlambatan dan ketidakhadiran dihitung dari jadwal shift sampai
            hari payroll dibuat, dan dapat disesuaikan di slip draft. Batas upah
            ditetapkan BPJS dan dapat berubah setiap tahun; perbarui sebelum
            membuat payroll. Payroll menyimpan nilai yang berlaku saat dibuat.
          </p>
          <ErrorBox error={m.error} />
          <div className="modal-actions">
            <button type="button" className="secondary" onClick={close}>
              Batal
            </button>
            <button className="primary" disabled={m.isPending}>
              Simpan pengaturan
            </button>
          </div>
        </form>
      ) : (
        <Loading />
      )}
    </Modal>
  );
}

export function Salaries() {
  const q = useData<Salary[]>("/salaries");
  const [edit, setEdit] = useState<Salary | null>(null);
  const [settings, setSettings] = useState(false);
  return (
    <>
      <Heading
        eyebrow="COMPENSATION SETUP"
        title="Komponen gaji yang jelas."
        description="Tetapkan gaji, tunjangan, PTKP, dan kepesertaan BPJS sebelum membuat payroll."
      >
        <button className="secondary" onClick={() => setSettings(true)}>
          <Settings2 size={15} />
          Pengaturan BPJS
        </button>
      </Heading>
      <div className="notice">
        Komponen ini menjadi dasar perhitungan saat payroll dibuat. Perubahan
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
                  <th>POTONGAN RUTIN</th>
                  <th>PAJAK & BPJS</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {q.data?.map((s) => {
                  const sum = (kind: string) =>
                    s.components
                      .filter((c) => c.kind === kind)
                      .reduce((t, c) => t + c.amount, 0);
                  const bpjs = [
                    s.bpjs_kesehatan && "Kes",
                    s.bpjs_ketenagakerjaan && "TK",
                    s.bpjs_pensiun && "JP",
                  ].filter(Boolean);
                  return (
                    <tr key={s.employee_id}>
                      <td>
                        <strong>{s.name}</strong>
                        <small className="block">
                          {s.code}
                          {!s.configured ? " · Belum diatur" : ""}
                        </small>
                      </td>
                      <td>{rupiah(s.basic_salary)}</td>
                      <td>{rupiah(sum("allowance"))}</td>
                      <td>{rupiah(sum("deduction"))}</td>
                      <td>
                        {s.ptkp_status} ·{" "}
                        {s.tax_method === "none"
                          ? "manual"
                          : s.tax_method.replace("_", "-")}
                        <small className="block">
                          BPJS {bpjs.length ? bpjs.join(", ") : "—"}
                        </small>
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
                  );
                })}
              </tbody>
            </table>
            {!q.data?.length && <Empty />}
          </div>
        )}
      </section>
      {edit && <SalaryForm salary={edit} close={() => setEdit(null)} />}
      {settings && <SettingsForm close={() => setSettings(false)} />}
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
    mutationFn: (v: { period: string; thr_date: string }) =>
      api<{ id: number }>("/payroll", "POST", v),
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
  const correct = useMutation({
    mutationFn: (id: number) =>
      api<{ id: number }>(`/payroll/${id}/correction`, "POST"),
    onSuccess: (d) => {
      qc.invalidateQueries();
      setSelected(d.id);
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
        description="Buat draft, verifikasi perhitungan, lalu terbitkan slip gaji."
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
              {selectedRun.kind === "correction" && (
                <span className="badge pending">
                  <i />
                  Koreksi
                </span>
              )}
              <h2>{rupiah(selectedRun.total)}</h2>
              <p>
                {selectedRun.employees} karyawan · PPh 21{" "}
                {rupiah(selectedRun.tax)} · BPJS perusahaan{" "}
                {rupiah(selectedRun.employer_cost)}
                {selectedRun.thr_date
                  ? ` · THR hari raya ${selectedRun.thr_date}`
                  : ""}
              </p>
              <p>
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
              {selectedRun.kind === "regular" &&
                (selectedRun.status === "finalized" ||
                  selectedRun.status === "paid") && (
                  <button
                    className="secondary"
                    disabled={correct.isPending}
                    onClick={() => correct.mutate(selectedRun.id)}
                  >
                    Buat koreksi
                  </button>
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
          <ErrorBox error={correct.error} />
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
                  <th>PPH 21</th>
                  <th>STATUS</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {q.data?.map((r) => (
                  <tr key={r.id}>
                    <td>
                      <strong>{r.period}</strong>
                      {r.thr_date && <small className="block">+ THR</small>}
                      {r.kind === "correction" && (
                        <small className="block">Koreksi</small>
                      )}
                    </td>
                    <td>{r.employees}</td>
                    <td>{rupiah(r.total)}</td>
                    <td>{rupiah(r.tax)}</td>
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
              const d = new FormData(e.currentTarget);
              create.mutate({
                period: String(d.get("period")),
                thr_date: d.get("with_thr") ? String(d.get("thr_date")) : "",
              });
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
            <ThrFields />
            <p className="notice">
              Karyawan aktif dan karyawan yang keluar pada periode ini
              disertakan, diprorata per hari kerja. Lembur yang disetujui dan
              belum dibayar ikut dihitung. BPJS dan PPh 21 (TER; Desember atau
              bulan keluar memakai perhitungan tahunan) dihitung otomatis.
              Periode sebelumnya harus sudah difinalisasi.
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
                Setelah finalisasi, nominal terkunci, slip dapat dilihat oleh
                masing-masing karyawan, dan PPh 21 periode ini menjadi dasar
                perhitungan tahunan. Pastikan seluruh slip sudah diverifikasi.
              </p>
            ) : (
              <p className="notice">
                Draft dibatalkan dan tidak ditampilkan kepada karyawan. Lembur
                di dalamnya kembali tersedia untuk payroll berikutnya.
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

function ThrFields() {
  const [withThr, setWithThr] = useState(false);
  return (
    <>
      <div className="weekday-picker">
        <Toggle
          label="Sertakan THR keagamaan"
          checked={withThr}
          onChange={setWithThr}
        />
        {withThr && <input type="hidden" name="with_thr" value="1" />}
      </div>
      {withThr && (
        <label>
          Tanggal hari raya
          <input name="thr_date" type="date" required />
          <span className="form-hint">
            Masa kerja dihitung sampai tanggal ini: ≥12 bulan mendapat 1 bulan
            upah tetap, 1–12 bulan proporsional.
          </span>
        </label>
      )}
    </>
  );
}

// downloadCsv saves rows as a UTF-8 CSV that Excel opens correctly; cells
// starting with formula characters are neutralised.
function downloadCsv(filename: string, rows: (string | number)[][]) {
  const cell = (v: string | number) =>
    '"' +
    String(v)
      .replace(/^[=+@\-\t\r]/, "'$&")
      .replaceAll('"', '""') +
    '"';
  const text = rows.map((row) => row.map(cell).join(",")).join("\r\n");
  const url = URL.createObjectURL(
    new Blob(["\uFEFF" + text], { type: "text/csv;charset=utf-8" }),
  );
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

const sumCodes = (s: Payslip, ...codes: string[]) =>
  s.lines
    .filter((l) => codes.includes(l.code))
    .reduce((t, l) => t + l.amount, 0);

function PayrollEntries({ run }: { run: PayrollRun }) {
  const q = useData<Payslip[]>(`/payroll/${run.id}/slips`);
  const [edit, setEdit] = useState<Payslip | null>(null);
  const [slip, setSlip] = useState<Payslip | null>(null);
  const slips = q.data || [];
  const name = `${run.period}${run.kind === "correction" ? "-koreksi" : ""}`;
  const csv = () =>
    downloadCsv(`payroll-${name}-${run.status}.csv`, [
      [
        "Kode",
        "Nama",
        "PTKP",
        "Hari kerja",
        "Gaji pokok",
        "Pendapatan lain",
        "Bruto pajak",
        "PPh 21",
        "BPJS karyawan",
        "Total potongan",
        "Gaji bersih",
        "BPJS perusahaan",
      ],
      ...slips.map((s) => [
        s.employee_code,
        s.employee_name,
        s.ptkp_status,
        `${s.worked_days}/${s.period_days}`,
        s.basic_salary,
        s.allowance,
        s.taxable_gross,
        s.pph21,
        sumCodes(s, "BPJS_KES_EE", "JHT_EE", "JP_EE"),
        s.deduction,
        s.net,
        s.employer_cost,
      ]),
    ]);
  // Recap for filling e-Bupot 21 / Coretax (not an official import file).
  const pph21 = () =>
    downloadCsv(`rekap-pph21-${name}.csv`, [
      [
        "Masa Pajak",
        "Tahun Pajak",
        "NIK",
        "NPWP",
        "Nama",
        "Status PTKP",
        "Kode Objek Pajak",
        "Penghasilan Bruto",
        "Tarif",
        "PPh 21 Dipotong",
        "Metode",
      ],
      ...slips
        .filter((s) => s.tax_method !== "none")
        .map((s) => [
          Number(run.period.slice(5)),
          run.period.slice(0, 4),
          s.nik,
          s.npwp,
          s.employee_name,
          s.ptkp_status,
          "21-100-01",
          s.taxable_gross,
          s.run_kind === "correction"
            ? "Koreksi (selisih)"
            : s.final_period
              ? "Pasal 17 (masa terakhir)"
              : `${(s.ter_rate / 100).toLocaleString("id-ID")}% TER`,
          s.pph21,
          s.tax_method === "gross_up" ? "Gross-up" : "Gross",
        ]),
    ]);
  const bpjsCodes = [
    "BPJS_KES_ER",
    "BPJS_KES_EE",
    "JHT_ER",
    "JHT_EE",
    "JP_ER",
    "JP_EE",
    "JKK",
    "JKM",
  ];
  const bpjs = () =>
    downloadCsv(`rekap-bpjs-${name}.csv`, [
      [
        "Kode",
        "Nama",
        "NIK",
        "No. BPJS Kesehatan",
        "No. BPJS Ketenagakerjaan",
        "Dasar upah",
        "Kes perusahaan",
        "Kes karyawan",
        "JHT perusahaan",
        "JHT karyawan",
        "JP perusahaan",
        "JP karyawan",
        "JKK",
        "JKM",
        "Total iuran",
      ],
      ...slips.map((s) => [
        s.employee_code,
        s.employee_name,
        s.nik,
        s.bpjs_kesehatan_number,
        s.bpjs_ketenagakerjaan_number,
        s.lines
          .filter((l) => l.kind === "earning" && l.fixed)
          .reduce((t, l) => t + l.amount, 0),
        ...bpjsCodes.map((c) => sumCodes(s, c)),
        sumCodes(s, ...bpjsCodes),
      ]),
    ]);
  return (
    <section className="panel">
      <div className="panel-head">
        <div>
          <h2>Rincian slip gaji</h2>
          <p>Snapshot perhitungan pada periode ini.</p>
        </div>
        <div className="row-actions">
          <button className="secondary" onClick={csv} disabled={!slips.length}>
            <Download size={15} />
            Ekspor CSV
          </button>
          <button
            className="secondary"
            onClick={pph21}
            disabled={!slips.length}
          >
            Rekap PPh 21
          </button>
          <button className="secondary" onClick={bpjs} disabled={!slips.length}>
            Rekap BPJS
          </button>
        </div>
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
                <th>PENDAPATAN</th>
                <th>BPJS</th>
                <th>PPH 21</th>
                <th>BERSIH</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {q.data?.map((s) => (
                <tr key={s.id}>
                  <td>
                    {s.employee_name}
                    <small className="block">
                      {s.employee_code} · {s.ptkp_status}
                      {s.worked_days < s.period_days &&
                        ` · prorata ${s.worked_days}/${s.period_days} hari`}
                      {s.final_period && " · PPh tahunan"}
                      {s.late_count > 0 &&
                        ` · terlambat ${s.late_count}× (${s.late_minutes} mnt)`}
                      {s.absent_days > 0 && ` · alpa ${s.absent_days} hari`}
                    </small>
                  </td>
                  <td>{rupiah(s.basic_salary + s.allowance)}</td>
                  <td>
                    {rupiah(sumCodes(s, "BPJS_KES_EE", "JHT_EE", "JP_EE"))}
                  </td>
                  <td>
                    {s.tax_method === "none"
                      ? "Manual"
                      : s.pph21 < 0
                        ? `− ${rupiah(-s.pph21)}`
                        : rupiah(s.pph21)}
                  </td>
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
                          Sesuaikan
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
      {edit && <SlipForm slip={edit} close={() => setEdit(null)} />}
      {slip && <SlipModal slip={slip} close={() => setSlip(null)} />}
    </section>
  );
}

function SlipForm({ slip, close }: { slip: Payslip; close: () => void }) {
  const qc = useQueryClient();
  const m = useMutation({
    mutationFn: (v: unknown) => api(`/payroll/slips/${slip.id}`, "PUT", v),
    onSuccess: () => {
      qc.invalidateQueries();
      close();
    },
  });
  const [rows, setRows] = useState<Row[]>(() =>
    slip.lines.filter((l) => INPUT_CODES.includes(l.code)).map((l) => toRow(l)),
  );
  const statutory = slip.lines.filter((l) => !INPUT_CODES.includes(l.code));
  return (
    <Modal title={`Penyesuaian · ${slip.employee_name}`} close={close}>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          m.mutate({
            version: slip.version,
            note: new FormData(e.currentTarget).get("note"),
            lines: rows.map((r): PayrollLine => ({
              kind: r.kind as PayrollLine["kind"],
              code: r.code,
              name: r.name,
              amount: r.amount,
              fixed: r.fixed,
              taxable: r.taxable,
            })),
          });
        }}
      >
        <div className="field-label">Pendapatan dan potongan</div>
        <LineEditor
          rows={rows}
          setRows={setRows}
          addEarning={{
            label: "Pendapatan (bonus, insentif…)",
            row: () =>
              toRow({
                kind: "earning",
                code: "ADJUSTMENT",
                name: "",
                amount: 0,
                fixed: false,
                taxable: true,
              }),
          }}
          addDeduction={{
            label: "Potongan (kasbon, unpaid leave…)",
            row: () =>
              toRow({
                kind: "deduction",
                code: "ADJUSTMENT",
                name: "",
                amount: 0,
                fixed: false,
                taxable: false,
              }),
          }}
        />
        {statutory.length > 0 && (
          <p className="form-hint">
            Dihitung ulang saat disimpan:{" "}
            {statutory.map((l) => `${l.name} ${rupiah(l.amount)}`).join(" · ")}
          </p>
        )}
        <label>
          Catatan
          <textarea
            name="note"
            defaultValue={slip.note}
            maxLength={1000}
            rows={2}
          />
        </label>
        <ErrorBox error={m.error} />
        <div className="modal-actions">
          <button type="button" className="secondary" onClick={close}>
            Batal
          </button>
          <button className="primary" disabled={m.isPending || !rows.length}>
            Simpan dan hitung ulang
          </button>
        </div>
      </form>
    </Modal>
  );
}

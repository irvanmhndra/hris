import { useState } from "react";
import { FileText, Printer, Wallet } from "lucide-react";
import { Heading } from "../components/Heading";
import {
  Badge,
  Empty,
  ErrorBox,
  Loading,
  Modal,
  rupiah,
  useData,
} from "../components/common";
import { useSession } from "../stores/session";
import type { PayrollLine, Payslip } from "../types";
function SlipSection({
  title,
  lines,
  minus,
  muted,
}: {
  title: string;
  lines: PayrollLine[];
  minus?: boolean;
  muted?: boolean;
}) {
  if (!lines.length) return null;
  return (
    <>
      <h3 className="slip-section">{title}</h3>
      <dl className={muted ? "slip-lines muted" : "slip-lines"}>
        {lines.map((l, i) => (
          <div key={i}>
            <dt>{l.name}</dt>
            <dd>
              {minus ? "− " : ""}
              {rupiah(l.amount)}
            </dd>
          </div>
        ))}
        <div className="slip-subtotal">
          <dt>Total</dt>
          <dd>
            {minus ? "− " : ""}
            {rupiah(lines.reduce((t, l) => t + l.amount, 0))}
          </dd>
        </div>
      </dl>
    </>
  );
}
export function SlipModal({
  slip,
  close,
}: {
  slip: Payslip;
  close: () => void;
}) {
  const company = useSession((s) => s.user?.company_name);
  return (
    <Modal title="Slip gaji" close={close}>
      <article className="payslip-print">
        <div className="slip-heading">
          <div>
            <span className="eyebrow">PEOPLE · PAYSLIP</span>
            <h2>{company}</h2>
            <p>
              Periode {slip.period} · ID slip {slip.id}
            </p>
          </div>
          <Badge status={slip.status} />
        </div>
        <div className="slip-person">
          <h2>{slip.employee_name}</h2>
          <p>
            {slip.employee_code} · {slip.position}
          </p>
        </div>
        <p className="slip-meta">
          PTKP {slip.ptkp_status}
          {slip.tax_method === "gross_up" && " · PPh 21 ditanggung perusahaan"}
          {slip.tax_method === "none" && " · PPh 21 dihitung manual"}
          {slip.worked_days < slip.period_days &&
            ` · ${slip.worked_days} dari ${slip.period_days} hari kerja`}
          {slip.final_period && " · PPh 21 perhitungan tahunan"}
        </p>
        <SlipSection
          title="Pendapatan"
          lines={slip.lines.filter((l) => l.kind === "earning")}
        />
        <SlipSection
          title="Potongan"
          lines={slip.lines.filter((l) => l.kind === "deduction")}
          minus
        />
        <div className="payroll-total">
          <span>Gaji bersih</span>
          <strong>{rupiah(slip.net)}</strong>
        </div>
        <SlipSection
          title="Ditanggung perusahaan (tidak memotong gaji)"
          lines={slip.lines.filter((l) => l.kind === "employer")}
          muted
        />
        {slip.note && <p className="full-description">{slip.note}</p>}
        <small>
          Slip merupakan catatan perhitungan HR. Status final belum menunjukkan
          dana telah ditransfer.
        </small>
      </article>
      <div className="modal-actions">
        <button className="primary" onClick={() => window.print()}>
          <Printer size={16} />
          Cetak / simpan PDF
        </button>
      </div>
    </Modal>
  );
}
export function Payslips() {
  const q = useData<Payslip[]>("/payslips");
  const [slip, setSlip] = useState<Payslip | null>(null);
  return (
    <>
      <Heading
        eyebrow="MY PAYSLIPS"
        title="Gaji Anda, transparan."
        description="Slip yang telah difinalisasi oleh HR tersedia di sini."
      />
      <ErrorBox error={q.error} />
      {q.isLoading ? (
        <Loading />
      ) : (
        <div className="department-cards">
          {q.data?.map((s) => (
            <section className="panel department-card" key={s.id}>
              <div className="record-heading">
                <span className="record-icon">
                  <Wallet size={22} />
                </span>
                <Badge status={s.status} />
              </div>
              <h2>{s.period}</h2>
              <p>Gaji bersih</p>
              <strong className="slip-amount">{rupiah(s.net)}</strong>
              <button className="primary" onClick={() => setSlip(s)}>
                <FileText size={16} />
                Lihat slip gaji
              </button>
            </section>
          ))}
        </div>
      )}
      {!q.isLoading && !q.data?.length && (
        <Empty>Belum ada slip gaji yang diterbitkan.</Empty>
      )}
      {slip && <SlipModal slip={slip} close={() => setSlip(null)} />}
    </>
  );
}

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
import type { Payslip } from "../types";
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
        <dl className="slip-lines">
          <div>
            <dt>Gaji pokok</dt>
            <dd>{rupiah(slip.basic_salary)}</dd>
          </div>
          <div>
            <dt>Tunjangan</dt>
            <dd>{rupiah(slip.allowance)}</dd>
          </div>
          <div>
            <dt>Gaji bruto</dt>
            <dd>{rupiah(slip.basic_salary + slip.allowance)}</dd>
          </div>
          <div>
            <dt>Potongan</dt>
            <dd>− {rupiah(slip.deduction)}</dd>
          </div>
        </dl>
        <div className="payroll-total">
          <span>Gaji bersih</span>
          <strong>{rupiah(slip.net)}</strong>
        </div>
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

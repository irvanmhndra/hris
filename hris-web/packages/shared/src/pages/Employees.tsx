import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Download, Plus, Search } from "lucide-react";
import { useState, type FormEvent } from "react";
import {
  Badge,
  date,
  Empty,
  ErrorBox,
  initials,
  Loading,
  Modal,
  today,
  useData,
} from "../components/common";
import { Heading } from "../components/Heading";
import { api } from "../services/api";
import type { Department, Employee } from "../types";
export function EmployeeTable({
  rows,
  edit,
}: {
  rows: Employee[];
  edit?: (e: Employee) => void;
}) {
  return (
    <div className="table-scroll">
      <table>
        <thead>
          <tr>
            <th>KARYAWAN</th>
            <th>DEPARTEMEN</th>
            <th>JABATAN</th>
            <th>BERGABUNG</th>
            <th>STATUS</th>
            {edit && <th />}
          </tr>
        </thead>
        <tbody>
          {rows.map((e, i) => (
            <tr key={e.id}>
              <td>
                <div className="person">
                  <span className={`avatar tone-${i % 4}`}>
                    {initials(e.name)}
                  </span>
                  <div>
                    <strong>{e.name}</strong>
                    <small>
                      {e.code} · {e.email}
                    </small>
                  </div>
                </div>
              </td>
              <td>
                <span className="department-tag">{e.department}</span>
              </td>
              <td>{e.position}</td>
              <td>{date(e.joined_on)}</td>
              <td>
                <Badge status={e.status} />
              </td>
              {edit && (
                <td>
                  <button className="text-button" onClick={() => edit(e)}>
                    Edit
                  </button>
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
      {!rows.length && <Empty />}
    </div>
  );
}
function EmployeeForm({
  employee,
  close,
}: {
  employee?: Employee;
  close: () => void;
}) {
  const depts = useData<Department[]>("/departments");
  const qc = useQueryClient();
  const m = useMutation({
    mutationFn: (v: Record<string, unknown>) =>
      api(
        employee ? `/employees/${employee.id}` : "/employees",
        employee ? "PUT" : "POST",
        v,
      ),
    onSuccess: () => {
      qc.invalidateQueries();
      close();
    },
  });
  function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const data = Object.fromEntries(new FormData(e.currentTarget));
    m.mutate({ ...data, department_id: Number(data.department_id) });
  }
  return (
    <Modal title={employee ? "Edit karyawan" : "Tambah karyawan"} close={close}>
      <form onSubmit={submit}>
        <div className="form-grid">
          <label>
            Nama lengkap
            <input
              autoFocus
              name="name"
              defaultValue={employee?.name}
              required
              maxLength={120}
            />
          </label>
          <label>
            NIK / kode karyawan
            <input
              name="code"
              defaultValue={employee?.code}
              required
              maxLength={40}
            />
          </label>
          <label className="full">
            Email kantor
            <input
              name="email"
              type="email"
              defaultValue={employee?.email}
              required
            />
          </label>
          <label>
            Departemen
            <select
              name="department_id"
              defaultValue={employee?.department_id || ""}
              required
            >
              <option value="" disabled>
                Pilih departemen
              </option>
              {depts.data?.map((d) => (
                <option value={d.id} key={d.id}>
                  {d.name}
                </option>
              ))}
            </select>
          </label>
          <label>
            Jabatan
            <input
              name="position"
              defaultValue={employee?.position}
              required
              maxLength={120}
            />
          </label>
          <label>
            Tanggal bergabung
            <input
              type="date"
              name="joined_on"
              defaultValue={employee?.joined_on || today()}
              required
            />
          </label>
          <label>
            Status
            <select name="status" defaultValue={employee?.status || "active"}>
              <option value="active">Aktif</option>
              <option value="inactive">Nonaktif</option>
            </select>
          </label>
          <label className="full">
            {employee ? "Password baru (opsional)" : "Password portal karyawan"}
            <input
              name="password"
              type="password"
              minLength={12}
              maxLength={72}
              required={!employee}
              autoComplete="new-password"
              placeholder={
                employee
                  ? "Kosongkan untuk mempertahankan password"
                  : "Minimal 12 karakter"
              }
            />
          </label>
        </div>
        <ErrorBox error={m.error || depts.error} />
        <div className="modal-actions">
          <button type="button" className="secondary" onClick={close}>
            Batal
          </button>
          <button
            className="primary"
            disabled={m.isPending || !depts.data?.length}
          >
            {m.isPending ? "Menyimpan…" : "Simpan karyawan"}
          </button>
        </div>
      </form>
    </Modal>
  );
}
export function Employees() {
  const q = useData<Employee[]>("/employees");
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("");
  const [editing, setEditing] = useState<Employee | null | undefined>(
    undefined,
  );
  const rows =
    q.data?.filter(
      (e) =>
        (!status || e.status === status) &&
        `${e.name} ${e.code} ${e.email} ${e.department}`
          .toLowerCase()
          .includes(search.toLowerCase()),
    ) || [];
  function download() {
    const columns = [
      "NIK",
      "Nama",
      "Email",
      "Departemen",
      "Jabatan",
      "Status",
      "Bergabung",
    ];
    const cell = (s: string) =>
      '"' + (/^[=+@\-\t\r]/.test(s) ? "'" + s : s).replaceAll('"', '""') + '"';
    const csv = [
      columns,
      ...rows.map((e) => [
        e.code,
        e.name,
        e.email,
        e.department,
        e.position,
        e.status,
        e.joined_on,
      ]),
    ]
      .map((r) => r.map(cell).join(","))
      .join("\r\n");
    const url = URL.createObjectURL(
      new Blob(["\uFEFF" + csv], { type: "text/csv;charset=utf-8;" }),
    );
    const a = document.createElement("a");
    a.href = url;
    a.download = "karyawan.csv";
    a.click();
    URL.revokeObjectURL(url);
  }
  return (
    <>
      <Heading
        eyebrow="PEOPLE DIRECTORY"
        title="Orang-orang di balik karya."
        description="Kelola informasi dan perjalanan setiap anggota tim."
      >
        <button
          className="secondary"
          onClick={download}
          disabled={!rows.length}
        >
          <Download size={16} />
          Ekspor CSV
        </button>
        <button className="primary" onClick={() => setEditing(null)}>
          <Plus size={17} />
          Tambah karyawan
        </button>
      </Heading>
      <section className="panel">
        <div className="toolbar">
          <div className="search">
            <Search size={18} />
            <input
              aria-label="Cari karyawan"
              placeholder="Cari nama, NIK, atau departemen…"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
          <select
            aria-label="Filter status"
            value={status}
            onChange={(e) => setStatus(e.target.value)}
          >
            <option value="">Semua status</option>
            <option value="active">Aktif</option>
            <option value="inactive">Nonaktif</option>
          </select>
          <span className="count">{rows.length} karyawan</span>
        </div>
        <ErrorBox error={q.error} />
        {q.isLoading ? (
          <Loading />
        ) : (
          <EmployeeTable rows={rows} edit={setEditing} />
        )}
      </section>
      {editing !== undefined && (
        <EmployeeForm
          employee={editing || undefined}
          close={() => setEditing(undefined)}
        />
      )}
    </>
  );
}

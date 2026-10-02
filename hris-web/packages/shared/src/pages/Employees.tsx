import {
  keepPreviousData,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { Download, Plus, Search } from "lucide-react";
import { useState, type FormEvent } from "react";
import {
  ErrorBox,
  Loading,
  Modal,
  Pager,
  today,
  useData,
  useDebounced,
} from "../components/common";
import { Heading } from "../components/Heading";
import { api, apiPage, query } from "../services/api";
import { EmployeeTable } from "../components/EmployeeTable";
import type { Department, Employee, Shift } from "../types";
function EmployeeForm({
  employee,
  close,
}: {
  employee?: Employee;
  close: () => void;
}) {
  const depts = useData<Department[]>("/departments");
  const people = useData<Employee[]>("/employees");
  const shifts = useData<Shift[]>("/shifts");
  const [status, setStatus] = useState(employee?.status || "active");
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
    m.mutate({
      ...data,
      department_id: Number(data.department_id),
      manager_id: data.manager_id ? Number(data.manager_id) : null,
      shift_id: data.shift_id ? Number(data.shift_id) : null,
    });
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
            Atasan langsung
            <select
              name="manager_id"
              key={people.data ? "ready" : "loading"}
              defaultValue={employee?.manager_id || ""}
            >
              <option value="">Tanpa atasan (langsung HR)</option>
              {people.data
                ?.filter((p) => p.id !== employee?.id && p.status === "active")
                .map((p) => (
                  <option value={p.id} key={p.id}>
                    {p.name} · {p.position}
                  </option>
                ))}
            </select>
          </label>
          <label>
            Shift default
            <select
              name="shift_id"
              key={shifts.data ? "ready" : "loading"}
              defaultValue={employee?.shift_id || ""}
            >
              <option value="">Jam kantor (kalender kerja)</option>
              {shifts.data
                ?.filter((s) => s.active || s.id === employee?.shift_id)
                .map((s) => (
                  <option value={s.id} key={s.id}>
                    {s.name} · {s.start_time}–{s.end_time}
                  </option>
                ))}
            </select>
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
            <select
              name="status"
              value={status}
              onChange={(e) => setStatus(e.target.value as Employee["status"])}
            >
              <option value="active">Aktif</option>
              <option value="inactive">Nonaktif</option>
            </select>
          </label>
          {status === "inactive" && (
            <label>
              Hari kerja terakhir
              <input
                type="date"
                name="left_on"
                defaultValue={employee?.left_on || today()}
                required
              />
            </label>
          )}
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
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("");
  const [page, setPage] = useState(1);
  const [exporting, setExporting] = useState(false);
  const [exportError, setExportError] = useState<unknown>(null);
  const [editing, setEditing] = useState<Employee | null | undefined>(
    undefined,
  );
  const term = useDebounced(search.trim());
  const filters = { search: term, status };
  const q = useQuery({
    queryKey: ["/employees", "page", term, status, page],
    queryFn: () =>
      apiPage<Employee>(
        "/employees" + query({ ...filters, page, per_page: 25 }),
      ),
    placeholderData: keepPreviousData,
  });
  const rows = q.data?.items || [];
  const total = q.data?.pagination.total_records ?? 0;
  // The CSV covers every employee matching the filters, not just this page.
  async function download() {
    setExporting(true);
    setExportError(null);
    try {
      const all = await api<Employee[]>("/employees" + query(filters));
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
        '"' +
        (/^[=+@\-\t\r]/.test(s) ? "'" + s : s).replaceAll('"', '""') +
        '"';
      const csv = [
        columns,
        ...all.map((e) => [
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
    } catch (e) {
      setExportError(e);
    } finally {
      setExporting(false);
    }
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
          disabled={!total || exporting}
        >
          <Download size={16} />
          {exporting ? "Mengekspor…" : "Ekspor CSV"}
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
              placeholder="Cari nama, NIK, email, atau departemen…"
              value={search}
              maxLength={120}
              onChange={(e) => {
                setSearch(e.target.value);
                setPage(1);
              }}
            />
          </div>
          <select
            aria-label="Filter status"
            value={status}
            onChange={(e) => {
              setStatus(e.target.value);
              setPage(1);
            }}
          >
            <option value="">Semua status</option>
            <option value="active">Aktif</option>
            <option value="inactive">Nonaktif</option>
          </select>
          <span className="count">{total} karyawan</span>
        </div>
        <ErrorBox error={q.error || exportError} />
        {q.isLoading ? (
          <Loading />
        ) : (
          <EmployeeTable rows={rows} edit={setEditing} />
        )}
        <Pager pagination={q.data?.pagination} onPage={setPage} />
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

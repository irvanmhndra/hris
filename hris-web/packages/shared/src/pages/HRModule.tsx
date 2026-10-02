import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import {
  Plus,
  Search,
  ArrowUpRight,
  CalendarDays,
  Download,
  Users,
  ClipboardList,
  UserPlus,
} from "lucide-react";
import { useParams } from "react-router";
import { Heading } from "../components/Heading";
import {
  Badge,
  date,
  Empty,
  ErrorBox,
  Loading,
  Modal,
  Pager,
  today,
} from "../components/common";
import { modules, statusLabels } from "../config/modules";
import { api, apiPage, download, query, upload } from "../services/api";
import { useSession } from "../stores/session";
import type { HRItem, Employee, Department } from "../types";
export function HRModule() {
  const { module = "" } = useParams();
  return modules[module] ? (
    <ModulePage key={module} module={module} />
  ) : (
    <Empty>Modul tidak ditemukan.</Empty>
  );
}
function ModulePage({ module }: { module: string }) {
  const cfg = modules[module];
  const admin = useSession((s) => s.user?.role === "admin");
  const [search, setSearch] = useState("");
  const [filter, setFilter] = useState("");
  const [page, setPage] = useState(1);
  const q = useQuery({
    queryKey: [`/hr/${module}`, "page", page, filter],
    queryFn: () =>
      apiPage<HRItem>(
        `/hr/${module}${query({ page, per_page: 20, status: filter })}`,
      ),
  });
  const qc = useQueryClient();
  const [hiring, setHiring] = useState<HRItem | null>(null);
  const [fileError, setFileError] = useState<unknown>(null);
  const [editing, setEditing] = useState<HRItem | null | undefined>();
  const [detail, setDetail] = useState<HRItem | null>(null);
  const [action, setAction] = useState<{ item: HRItem; action: string } | null>(
    null,
  );
  const m = useMutation({
    mutationFn: (v: Record<string, unknown>) =>
      api(`/hr/${module}/${action!.item.id}/action`, "PATCH", {
        ...v,
        action: action!.action,
        version: action!.item.version,
      }),
    onSuccess: () => {
      qc.invalidateQueries();
      setAction(null);
    },
  });
  const rows =
    q.data?.items.filter((x) =>
      `${x.title} ${x.employee_name} ${x.description} ${x.data.code || ""} ${x.data.position || ""}`
        .toLowerCase()
        .includes(search.toLowerCase()),
    ) || [];
  const canCreate = cfg.request ? !admin : admin;
  const choose = (item: HRItem, verb: string) => {
    m.reset();
    setAction({ item, action: verb });
  };
  return (
    <>
      <Heading
        eyebrow={cfg.eyebrow}
        title={cfg.title}
        description={cfg.description}
      >
        {canCreate && (
          <button className="primary" onClick={() => setEditing(null)}>
            <Plus size={16} />
            {cfg.request ? "Ajukan" : "Tambah"} {cfg.singular.toLowerCase()}
          </button>
        )}
      </Heading>
      <div className="module-summary">
        <span>
          <strong>{q.data?.pagination.total_records || 0}</strong>{" "}
          {cfg.singular.toLowerCase()}
          {filter && ` · ${statusLabels[filter]}`}
        </span>
        <span className="muted">
          {admin ? "Workspace perusahaan" : "Data sesuai akses Anda"}
        </span>
      </div>
      <section className="panel">
        <div className="toolbar">
          <div className="search">
            <Search size={17} />
            <input
              aria-label="Cari data"
              placeholder="Cari di halaman ini…"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
          <select
            aria-label="Status"
            value={filter}
            onChange={(e) => {
              setFilter(e.target.value);
              setPage(1);
            }}
          >
            <option value="">Semua status</option>
            {cfg.statuses.map((s) => (
              <option value={s} key={s}>
                {statusLabels[s]}
              </option>
            ))}
          </select>
          <span className="count">{rows.length} catatan</span>
        </div>
        <ErrorBox error={q.error || fileError} />
        {q.isLoading ? (
          <Loading />
        ) : !rows.length ? (
          <Empty>Belum ada {cfg.singular.toLowerCase()} yang sesuai.</Empty>
        ) : (
          <div className="record-list">
            {rows.map((item) => (
              <article className="record" key={item.id}>
                <div className="record-icon">
                  <ClipboardList size={21} />
                </div>
                <div className="record-body">
                  <div className="record-heading">
                    <button
                      className="record-title"
                      onClick={() => setDetail(item)}
                    >
                      {item.title}
                    </button>
                    <Badge status={item.status} />
                  </div>
                  {item.status === "pending" && item.stage === "manager" && (
                    <small className="block">Menunggu persetujuan atasan</small>
                  )}
                  <p className="record-excerpt">
                    {item.description || "Belum ada keterangan tambahan."}
                  </p>
                  <div className="record-meta">
                    {item.employee_name && (
                      <span>
                        <Users size={13} />
                        {item.employee_name}
                      </span>
                    )}
                    {item.due_date && (
                      <span>
                        <CalendarDays size={13} />
                        {date(item.due_date)}
                      </span>
                    )}
                    {module === "assets" && (
                      <span>
                        {item.data.code} · {item.data.category}
                      </span>
                    )}
                    {module === "recruitment" && (
                      <span>
                        {item.data.position} · {item.data.email}
                      </span>
                    )}
                    {module === "corrections" && (
                      <span>
                        {date(String(item.data.date))} · {item.data.check_in}–
                        {item.data.check_out} WIB
                      </span>
                    )}
                    {module === "overtime" && (
                      <span>
                        {new Date(String(item.data.start_at)).toLocaleString(
                          "id-ID",
                          { timeZone: "Asia/Jakarta" },
                        )}{" "}
                        ·{" "}
                        {(
                          (+new Date(String(item.data.end_at)) -
                            +new Date(String(item.data.start_at))) /
                          3600000
                        ).toFixed(2)}{" "}
                        jam
                      </span>
                    )}
                  </div>
                  {module === "goals" && (
                    <div className="goal-progress">
                      <div>
                        <span
                          style={{
                            width: `${Number(item.data.progress) || 0}%`,
                          }}
                        />
                      </div>
                      <strong>{Number(item.data.progress) || 0}%</strong>
                    </div>
                  )}
                  {item.review_note && (
                    <p className="review-note">
                      Catatan HR: {item.review_note}
                    </p>
                  )}
                </div>
                <div className="record-actions">
                  {admin && !cfg.request && (
                    <button
                      className="secondary"
                      onClick={() => setEditing(item)}
                    >
                      Edit
                    </button>
                  )}
                  {cfg.request &&
                    item.status === "pending" &&
                    (admin ? (
                      <>
                        <button
                          className="primary"
                          onClick={() => choose(item, "approve")}
                        >
                          Setujui
                        </button>
                        <button
                          className="secondary"
                          onClick={() => choose(item, "reject")}
                        >
                          Tolak
                        </button>
                      </>
                    ) : (
                      <button
                        className="secondary"
                        onClick={() => choose(item, "cancel")}
                      >
                        Batalkan
                      </button>
                    ))}
                  {module === "onboarding" && item.status !== "done" && (
                    <button
                      className="primary"
                      onClick={() =>
                        choose(
                          item,
                          item.status === "todo" ? "start" : "complete",
                        )
                      }
                    >
                      {item.status === "todo" ? "Mulai tugas" : "Selesaikan"}
                    </button>
                  )}
                  {module === "goals" && (
                    <button
                      className="primary"
                      onClick={() => choose(item, "progress")}
                    >
                      Update progres
                    </button>
                  )}
                  {!!item.data.file_id && (
                    <button
                      className="secondary"
                      onClick={() =>
                        download(
                          Number(item.data.file_id),
                          String(item.data.file_name || "lampiran"),
                        ).catch(setFileError)
                      }
                    >
                      <Download size={15} />
                      {String(item.data.file_name || "Unduh lampiran")}
                    </button>
                  )}
                  {admin &&
                    module === "recruitment" &&
                    !item.employee_id &&
                    (item.status === "offer" || item.status === "hired") && (
                      <button
                        className="primary"
                        onClick={() => setHiring(item)}
                      >
                        <UserPlus size={15} />
                        Jadikan karyawan
                      </button>
                    )}
                  {module === "documents" &&
                    String(item.data.url).startsWith("https://") && (
                      <a
                        className="primary"
                        href={String(item.data.url)}
                        target="_blank"
                        rel="noopener noreferrer"
                      >
                        Buka dokumen <ArrowUpRight size={15} />
                      </a>
                    )}
                </div>
              </article>
            ))}
          </div>
        )}
        <Pager pagination={q.data?.pagination} onPage={setPage} />
      </section>
      {hiring && <HireForm candidate={hiring} close={() => setHiring(null)} />}
      {editing !== undefined && (
        <ItemForm
          module={module}
          item={editing || undefined}
          close={() => setEditing(undefined)}
        />
      )}
      {detail && (
        <Modal title={detail.title} close={() => setDetail(null)}>
          <Badge status={detail.status} />
          <p className="full-description">
            {detail.description || "Tidak ada keterangan tambahan."}
          </p>
          {cfg.fields
            .filter((f) => f.key !== "progress")
            .map((f) => (
              <p key={f.key}>
                <strong>{f.label}:</strong> {String(detail.data[f.key] || "—")}
              </p>
            ))}
        </Modal>
      )}
      {action && (
        <Modal
          title={
            {
              approve: "Setujui pengajuan?",
              reject: "Tolak pengajuan?",
              cancel: "Batalkan pengajuan?",
              start: "Mulai tugas?",
              complete: "Selesaikan tugas?",
              progress: "Update progres",
            }[action.action] || "Konfirmasi"
          }
          close={() => setAction(null)}
        >
          <p className="modal-context">
            {action.item.title}
            {action.item.employee_name && ` · ${action.item.employee_name}`}
          </p>
          {module === "corrections" && action.action === "approve" && (
            <p className="notice">
              Persetujuan akan memperbarui jam masuk dan pulang pada catatan
              absensi.
            </p>
          )}
          <form
            onSubmit={(e) => {
              e.preventDefault();
              const v = Object.fromEntries(new FormData(e.currentTarget));
              m.mutate({ ...v, progress: Number(v.progress || 0) });
            }}
          >
            {action.action === "progress" ? (
              <label>
                Progres (%)
                <input
                  autoFocus
                  name="progress"
                  type="number"
                  min={0}
                  max={100}
                  defaultValue={Number(action.item.data.progress) || 0}
                  required
                />
              </label>
            ) : action.action === "reject" || action.action === "approve" ? (
              <label>
                Catatan HR{" "}
                {action.action === "reject" ? "(wajib)" : "(opsional)"}
                <textarea
                  name="note"
                  required={action.action === "reject"}
                  minLength={action.action === "reject" ? 5 : undefined}
                  maxLength={1000}
                  rows={3}
                />
              </label>
            ) : null}
            <ErrorBox error={m.error} />
            <div className="modal-actions">
              <button
                type="button"
                className="secondary"
                onClick={() => setAction(null)}
              >
                Kembali
              </button>
              <button className="primary" disabled={m.isPending}>
                {m.isPending ? "Menyimpan…" : "Konfirmasi"}
              </button>
            </div>
          </form>
        </Modal>
      )}
    </>
  );
}
function ItemForm({
  module,
  item,
  close,
}: {
  module: string;
  item?: HRItem;
  close: () => void;
}) {
  const cfg = modules[module];
  const admin = useSession((s) => s.user?.role === "admin");
  const qc = useQueryClient();
  const employees = useQuery({
    queryKey: ["/employees"],
    queryFn: () => api<Employee[]>("/employees"),
    enabled: !!admin && !!cfg.assignment,
  });
  const attachable =
    admin && (module === "documents" || module === "announcements");
  const [keepFile, setKeepFile] = useState(!!item?.data.file_id);
  const m = useMutation({
    mutationFn: async ({
      v,
      file,
    }: {
      v: { data: Record<string, unknown> } & Record<string, unknown>;
      file: File | null;
    }) => {
      if (file && file.size) {
        const f = await upload(file);
        v.data = { ...v.data, file_id: f.id, url: "" };
      } else if (keepFile && item?.data.file_id) {
        v.data = { ...v.data, file_id: Number(item.data.file_id), url: "" };
      }
      return api(
        item ? `/hr/${module}/${item.id}` : `/hr/${module}`,
        item ? "PUT" : "POST",
        v,
      );
    },
    onSuccess: () => {
      qc.invalidateQueries();
      close();
    },
  });
  return (
    <Modal
      title={`${item ? "Edit" : cfg.request ? "Ajukan" : "Tambah"} ${cfg.singular.toLowerCase()}`}
      close={close}
    >
      <form
        onSubmit={(e) => {
          e.preventDefault();
          const fields = Object.fromEntries(new FormData(e.currentTarget));
          const data: Record<string, unknown> = {};
          for (const f of cfg.fields) {
            let value: unknown = fields[f.key];
            if (f.type === "number") value = Number(value);
            if (f.type === "datetime-local")
              value = String(value) + ":00+07:00";
            data[f.key] = value;
          }
          const file = fields.file instanceof File ? fields.file : null;
          m.mutate({
            v: {
              title: fields.title,
              description: fields.description,
              status: fields.status || cfg.statuses[0],
              due_date: fields.due_date || "",
              employee_id: fields.employee_id
                ? Number(fields.employee_id)
                : null,
              data,
              version: item?.version || 0,
            },
            file,
          });
        }}
      >
        <label>
          {cfg.titleLabel || "Judul"}
          <input
            name="title"
            defaultValue={item?.title}
            minLength={2}
            maxLength={160}
            required
            autoFocus
          />
        </label>
        <div className="form-grid">
          {cfg.assignment && (
            <label>
              Karyawan {cfg.assignment === "optional" ? "(jika dipinjam)" : ""}
              <select
                name="employee_id"
                defaultValue={item?.employee_id || ""}
                required={cfg.assignment === "required"}
              >
                <option value="">Pilih karyawan</option>
                {employees.data?.map((e) => (
                  <option value={e.id} key={e.id}>
                    {e.name} · {e.code}
                  </option>
                ))}
              </select>
            </label>
          )}
          {!cfg.request && module !== "goals" && (
            <label>
              Status
              <select
                name="status"
                defaultValue={item?.status || cfg.statuses[0]}
              >
                {cfg.statuses.map((s) => (
                  <option value={s} key={s}>
                    {statusLabels[s]}
                  </option>
                ))}
              </select>
            </label>
          )}
          {cfg.due && (
            <label>
              Tenggat / tanggal tindak lanjut
              <input
                name="due_date"
                type="date"
                defaultValue={item?.due_date}
              />
            </label>
          )}
          {cfg.fields.map((f) => (
            <label
              key={f.key}
              className={f.key === "url" || f.key === "target" ? "full" : ""}
            >
              {f.label}
              <input
                name={f.key}
                type={f.type || "text"}
                required={f.required}
                min={f.min}
                max={f.max}
                maxLength={f.key === "target" ? 500 : 1000}
                defaultValue={
                  item?.data[f.key] ?? (f.type === "number" ? 0 : "")
                }
              />
            </label>
          ))}
        </div>
        {attachable && (
          <label>
            Lampiran file (PDF, gambar, DOCX, XLSX · maks. 10 MB)
            {keepFile && item?.data.file_id ? (
              <span className="row-actions">
                {String(item.data.file_name)}
                <button
                  type="button"
                  className="text-button"
                  onClick={() => setKeepFile(false)}
                >
                  Ganti / hapus
                </button>
              </span>
            ) : (
              <input
                name="file"
                type="file"
                accept=".pdf,.png,.jpg,.jpeg,.webp,.docx,.xlsx"
              />
            )}
          </label>
        )}
        <label>
          {cfg.request ? "Alasan" : "Keterangan"}
          <textarea
            name="description"
            defaultValue={item?.description}
            rows={4}
            required={cfg.request || module === "announcements"}
            minLength={cfg.request ? 5 : undefined}
            maxLength={5000}
          />
        </label>
        <ErrorBox error={m.error || employees.error} />
        <div className="modal-actions">
          <button className="secondary" type="button" onClick={close}>
            Batal
          </button>
          <button className="primary" disabled={m.isPending}>
            {m.isPending ? "Menyimpan…" : "Simpan"}
          </button>
        </div>
      </form>
    </Modal>
  );
}

// HireForm turns an offer/hired candidate into an employee with a portal
// account; the candidate is linked to the new employee.
function HireForm({
  candidate,
  close,
}: {
  candidate: HRItem;
  close: () => void;
}) {
  const qc = useQueryClient();
  const depts = useQuery({
    queryKey: ["/departments"],
    queryFn: () => api<Department[]>("/departments"),
  });
  const m = useMutation({
    mutationFn: (v: Record<string, unknown>) =>
      api(`/hr/recruitment/${candidate.id}/convert`, "POST", v),
    onSuccess: () => {
      qc.invalidateQueries();
      close();
    },
  });
  return (
    <Modal title={`Jadikan karyawan · ${candidate.title}`} close={close}>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          const d = Object.fromEntries(new FormData(e.currentTarget));
          m.mutate({ ...d, department_id: Number(d.department_id) });
        }}
      >
        <div className="form-grid">
          <label>
            Nama lengkap
            <input
              name="name"
              defaultValue={candidate.title}
              required
              maxLength={120}
            />
          </label>
          <label>
            NIK / kode karyawan
            <input name="code" required maxLength={40} autoFocus />
          </label>
          <label className="full">
            Email kantor
            <input
              name="email"
              type="email"
              defaultValue={String(candidate.data.email || "")}
              required
            />
          </label>
          <label>
            Departemen
            <select name="department_id" defaultValue="" required>
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
              defaultValue={String(candidate.data.position || "")}
              required
              maxLength={120}
            />
          </label>
          <label>
            Tanggal bergabung
            <input
              name="joined_on"
              type="date"
              defaultValue={today()}
              required
            />
          </label>
          <label>
            Password portal karyawan
            <input
              name="password"
              type="password"
              minLength={12}
              maxLength={72}
              required
              autoComplete="new-password"
            />
          </label>
        </div>
        <p className="form-hint">
          Data karyawan dan akun portal dibuat sekaligus; status kandidat
          menjadi Diterima dan tertaut ke karyawan baru. Atur gaji, atasan, dan
          shift setelahnya.
        </p>
        <ErrorBox error={m.error || depts.error} />
        <div className="modal-actions">
          <button type="button" className="secondary" onClick={close}>
            Batal
          </button>
          <button className="primary" disabled={m.isPending}>
            Buat karyawan
          </button>
        </div>
      </form>
    </Modal>
  );
}

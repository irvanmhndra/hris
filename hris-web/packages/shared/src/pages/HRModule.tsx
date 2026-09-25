import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import {
  Plus,
  Search,
  ArrowUpRight,
  CalendarDays,
  Users,
  ClipboardList,
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
  useData,
} from "../components/common";
import { modules, statusLabels } from "../config/modules";
import { api } from "../services/api";
import { useSession } from "../stores/session";
import type { HRItem, Employee } from "../types";
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
  const q = useData<HRItem[]>(`/hr/${module}`);
  const qc = useQueryClient();
  const [search, setSearch] = useState("");
  const [filter, setFilter] = useState("");
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
    q.data?.filter(
      (x) =>
        (!filter || x.status === filter) &&
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
          <strong>{q.data?.length || 0}</strong> {cfg.singular.toLowerCase()}
        </span>
        <span>
          <strong>
            {q.data?.filter((x) =>
              ["pending", "todo", "active", "applied", "draft"].includes(
                x.status,
              ),
            ).length || 0}
          </strong>{" "}
          perlu ditindaklanjuti
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
              placeholder="Cari judul, nama, atau detail…"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
          <select
            aria-label="Status"
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
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
        <ErrorBox error={q.error} />
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
      </section>
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
  const m = useMutation({
    mutationFn: (v: unknown) =>
      api(
        item ? `/hr/${module}/${item.id}` : `/hr/${module}`,
        item ? "PUT" : "POST",
        v,
      ),
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
          m.mutate({
            title: fields.title,
            description: fields.description,
            status: fields.status || cfg.statuses[0],
            due_date: fields.due_date || "",
            employee_id: fields.employee_id ? Number(fields.employee_id) : null,
            data,
            version: item?.version || 0,
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

import { QueryClient, useQuery } from "@tanstack/react-query";
import { ChevronLeft, ChevronRight, Leaf, X } from "lucide-react";
import { useEffect, useRef, useState, type ReactNode } from "react";
import { statusLabels } from "../config/modules";
import { api } from "../services/api";
import type { Pagination } from "../types";
export const client = new QueryClient({
  defaultOptions: {
    queries: { retry: 1, staleTime: 15000, refetchOnWindowFocus: true },
  },
});
export const today = () =>
  new Intl.DateTimeFormat("en-CA", {
    timeZone: "Asia/Jakarta",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(new Date());
export const date = (s: string) =>
  new Date(s.length === 10 ? s + "T12:00:00+07:00" : s).toLocaleDateString(
    "id-ID",
    {
      day: "numeric",
      month: "short",
      year: "numeric",
      timeZone: "Asia/Jakarta",
    },
  );
export const time = (s: string | null) =>
  s
    ? new Date(s).toLocaleTimeString("id-ID", {
        hour: "2-digit",
        minute: "2-digit",
        timeZone: "Asia/Jakarta",
      })
    : "—";
export const rupiah = (value: number) =>
  new Intl.NumberFormat("id-ID", {
    style: "currency",
    currency: "IDR",
    maximumFractionDigits: 0,
  }).format(value);
export const initials = (s: string) =>
  s
    .split(" ")
    .slice(0, 2)
    .map((x) => x[0])
    .join("");
export const kinds: Record<string, string> = {
  annual: "Cuti tahunan",
  sick: "Sakit",
  personal: "Izin pribadi",
  unpaid: "Cuti tidak dibayar",
};
export function Badge({ status }: { status: string }) {
  const labels: Record<string, string> = {
    ...statusLabels,
    active: "Aktif",
    inactive: "Nonaktif",
    pending: "Menunggu",
    approved: "Disetujui",
    rejected: "Ditolak",
  };
  return (
    <span className={`badge ${status}`}>
      <i />
      {labels[status] || status}
    </span>
  );
}
export function ErrorBox({ error }: { error: unknown }) {
  return error ? (
    <div role="alert" className="error">
      {error instanceof Error ? error.message : "Terjadi kesalahan"}
    </div>
  ) : null;
}
export function Empty({
  children = "Belum ada data untuk ditampilkan.",
}: {
  children?: ReactNode;
}) {
  return (
    <div className="empty">
      <Leaf size={30} />
      <p>{children}</p>
    </div>
  );
}
export function Loading() {
  return <div className="empty">Memuat data…</div>;
}
export function useData<T>(path: string) {
  return useQuery({ queryKey: [path], queryFn: () => api<T>(path) });
}
// useDebounced delays a fast-changing value (e.g. a search box) so each
// keystroke does not trigger its own request.
export function useDebounced<T>(value: T, ms = 300) {
  const [v, setV] = useState(value);
  useEffect(() => {
    const t = setTimeout(() => setV(value), ms);
    return () => clearTimeout(t);
  }, [value, ms]);
  return v;
}
export function Pager({
  pagination,
  onPage,
}: {
  pagination?: Pagination;
  onPage: (page: number) => void;
}) {
  if (!pagination || pagination.total_pages <= 1) return null;
  const { current_page: page, total_pages: pages } = pagination;
  return (
    <nav className="pager" aria-label="Halaman">
      <button
        className="secondary"
        disabled={page <= 1}
        onClick={() => onPage(page - 1)}
        aria-label="Halaman sebelumnya"
      >
        <ChevronLeft size={15} />
      </button>
      <span>
        Halaman {page} dari {pages}
      </span>
      <button
        className="secondary"
        disabled={page >= pages}
        onClick={() => onPage(page + 1)}
        aria-label="Halaman berikutnya"
      >
        <ChevronRight size={15} />
      </button>
    </nav>
  );
}
export function Modal({
  title,
  children,
  close,
}: {
  title: string;
  children: ReactNode;
  close: () => void;
}) {
  const dialog = useRef<HTMLElement>(null);
  const closeRef = useRef(close);
  closeRef.current = close;
  useEffect(() => {
    const previous = document.activeElement as HTMLElement | null;
    const panel = dialog.current;
    const focusable = () =>
      Array.from(
        panel?.querySelectorAll<HTMLElement>(
          'button:not(:disabled), input:not(:disabled), select:not(:disabled), textarea:not(:disabled), a[href], [tabindex="0"]',
        ) || [],
      );
    (
      panel?.querySelector<HTMLElement>("input, select, textarea") ||
      focusable()[0]
    )?.focus();
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") closeRef.current();
      if (e.key === "Tab") {
        const nodes = focusable();
        const first = nodes[0];
        const last = nodes[nodes.length - 1];
        if (e.shiftKey && document.activeElement === first) {
          e.preventDefault();
          last?.focus();
        } else if (!e.shiftKey && document.activeElement === last) {
          e.preventDefault();
          first?.focus();
        }
      }
    };
    const overflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    window.addEventListener("keydown", handler);
    return () => {
      window.removeEventListener("keydown", handler);
      document.body.style.overflow = overflow;
      previous?.focus();
    };
  }, []);
  return (
    <div
      className="overlay"
      onClick={(e) => {
        if (e.target === e.currentTarget) close();
      }}
    >
      <section
        ref={dialog}
        role="dialog"
        aria-modal="true"
        aria-label={title}
        className="modal"
      >
        <div className="modal-head">
          <h2>{title}</h2>
          <button className="icon-button" aria-label="Tutup" onClick={close}>
            <X size={20} />
          </button>
        </div>
        {children}
      </section>
    </div>
  );
}

export type Field = {
  key: string;
  label: string;
  type?:
    "text" | "email" | "url" | "date" | "time" | "datetime-local" | "number";
  required?: boolean;
  min?: number;
  max?: number;
};
export type ModuleConfig = {
  title: string;
  description: string;
  eyebrow: string;
  singular: string;
  titleLabel?: string;
  assignment?: "required" | "optional";
  due?: boolean;
  request?: boolean;
  fields: Field[];
  statuses: string[];
};
export const statusLabels: Record<string, string> = {
  draft: "Draft",
  published: "Dipublikasikan",
  archived: "Diarsipkan",
  todo: "Belum dimulai",
  in_progress: "Dikerjakan",
  done: "Selesai",
  available: "Tersedia",
  assigned: "Dipinjam",
  maintenance: "Perawatan",
  retired: "Tidak digunakan",
  active: "Aktif",
  applied: "Melamar",
  screening: "Seleksi awal",
  interview: "Wawancara",
  offer: "Penawaran",
  hired: "Diterima",
  rejected: "Ditolak",
  pending: "Menunggu",
  approved: "Disetujui",
  cancelled: "Dibatalkan",
  finalized: "Final",
  paid: "Dibayar",
  void: "Dibatalkan",
};
export const modules: Record<string, ModuleConfig> = {
  overtime: {
    title: "Lembur yang tercatat.",
    description: "Ajukan waktu lembur dan pantau keputusan HR.",
    eyebrow: "OVERTIME",
    singular: "Pengajuan lembur",
    request: true,
    fields: [
      {
        key: "start_at",
        label: "Mulai (WIB)",
        type: "datetime-local",
        required: true,
      },
      {
        key: "end_at",
        label: "Selesai (WIB)",
        type: "datetime-local",
        required: true,
      },
    ],
    statuses: ["pending", "approved", "rejected", "cancelled"],
  },
  corrections: {
    title: "Koreksi kehadiran.",
    description: "Perbaikan catatan absensi melalui persetujuan HR.",
    eyebrow: "ATTENDANCE CORRECTIONS",
    singular: "Koreksi absensi",
    request: true,
    fields: [
      { key: "date", label: "Tanggal kehadiran", type: "date", required: true },
      {
        key: "check_in",
        label: "Jam masuk (WIB)",
        type: "time",
        required: true,
      },
      {
        key: "check_out",
        label: "Jam pulang (WIB)",
        type: "time",
        required: true,
      },
    ],
    statuses: ["pending", "approved", "rejected", "cancelled"],
  },
  announcements: {
    title: "Kabar untuk semua.",
    description: "Informasi dan pengumuman resmi perusahaan.",
    eyebrow: "COMPANY NEWS",
    singular: "Pengumuman",
    fields: [],
    statuses: ["draft", "published", "archived"],
  },
  documents: {
    title: "Panduan dalam satu tempat.",
    description: "Kebijakan dan dokumen perusahaan yang dapat diakses tim.",
    eyebrow: "KNOWLEDGE & POLICIES",
    singular: "Dokumen kebijakan",
    fields: [
      {
        key: "url",
        label: "Tautan dokumen (HTTPS, bila tidak mengunggah file)",
        type: "url",
      },
      { key: "category", label: "Kategori", required: true },
    ],
    statuses: ["draft", "published", "archived"],
  },
  onboarding: {
    title: "Awal yang lebih terarah.",
    description:
      "Tugas orientasi dengan penanggung jawab dan tenggat yang jelas.",
    eyebrow: "ONBOARDING",
    singular: "Tugas onboarding",
    assignment: "required",
    due: true,
    fields: [],
    statuses: ["todo", "in_progress", "done"],
  },
  assets: {
    title: "Aset yang terkelola.",
    description: "Catat inventaris, peminjam, dan kondisi operasional aset.",
    eyebrow: "COMPANY ASSETS",
    singular: "Aset",
    assignment: "optional",
    fields: [
      { key: "code", label: "Kode aset unik", required: true },
      { key: "category", label: "Kategori", required: true },
      { key: "serial", label: "Nomor seri" },
    ],
    statuses: ["available", "assigned", "maintenance", "retired"],
  },
  goals: {
    title: "Tujuan yang terukur.",
    description:
      "Selaraskan target kerja dan catat progres setiap anggota tim.",
    eyebrow: "PERFORMANCE GOALS",
    singular: "Target kinerja",
    assignment: "required",
    due: true,
    fields: [
      { key: "target", label: "Ukuran keberhasilan", required: true },
      {
        key: "progress",
        label: "Progres (%)",
        type: "number",
        min: 0,
        max: 100,
        required: true,
      },
    ],
    statuses: ["active", "done"],
  },
  recruitment: {
    title: "Temukan anggota tim berikutnya.",
    description: "Kelola kandidat dari lamaran hingga keputusan akhir.",
    eyebrow: "RECRUITMENT",
    singular: "Kandidat",
    titleLabel: "Nama kandidat",
    due: true,
    fields: [
      { key: "email", label: "Email kandidat", type: "email", required: true },
      { key: "position", label: "Posisi yang dilamar", required: true },
    ],
    statuses: [
      "applied",
      "screening",
      "interview",
      "offer",
      "hired",
      "rejected",
    ],
  },
};

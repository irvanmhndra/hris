import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { Empty, ErrorBox, Loading, Pager } from "../components/common";
import { Heading } from "../components/Heading";
import { apiPage } from "../services/api";
import type { AuditLog } from "../types";
export function Audit() {
  const [page, setPage] = useState(1);
  const q = useQuery({
    queryKey: ["/audit-logs", page],
    queryFn: () => apiPage<AuditLog>(`/audit-logs?page=${page}&per_page=50`),
    placeholderData: keepPreviousData,
  });
  return (
    <>
      <Heading
        eyebrow="AUDIT TRAIL"
        title="Setiap perubahan, tercatat."
        description="Seluruh aktivitas pada cuti, kalender, profil, modul HR, dan payroll, terbaru lebih dulu."
      />
      <section className="panel">
        <ErrorBox error={q.error} />
        {q.isLoading ? (
          <Loading />
        ) : (
          <div className="record-list">
            {q.data?.items.map((a) => (
              <article className="audit-row" key={a.id}>
                <div className="audit-dot" />
                <div>
                  <strong>{a.summary}</strong>
                  <p>
                    {a.actor} · {a.action} · {a.resource} #{a.resource_id}
                  </p>
                </div>
                <time>
                  {new Date(a.created_at).toLocaleString("id-ID", {
                    timeZone: "Asia/Jakarta",
                  })}
                </time>
              </article>
            ))}
            {!q.data?.items.length && <Empty />}
          </div>
        )}
        <Pager pagination={q.data?.pagination} onPage={setPage} />
      </section>
    </>
  );
}

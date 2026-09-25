import { useMutation, useQueryClient } from "@tanstack/react-query";
import { ArrowUpRight, Building2, Plus } from "lucide-react";
import { useState } from "react";
import { NavLink } from "react-router";
import { Empty, ErrorBox, Loading, Modal, useData } from "../components/common";
import { Heading } from "../components/Heading";
import { api } from "../services/api";
import type { Department } from "../types";
export function Departments() {
  const q = useData<Department[]>("/departments");
  const [open, setOpen] = useState(false);
  const qc = useQueryClient();
  const m = useMutation({
    mutationFn: (v: unknown) => api("/departments", "POST", v),
    onSuccess: () => {
      qc.invalidateQueries();
      setOpen(false);
    },
  });
  return (
    <>
      <Heading
        eyebrow="TEAM STRUCTURE"
        title="Bersama, dalam satu tujuan."
        description="Tempat setiap keahlian menemukan ruangnya."
      >
        <button
          className="primary"
          onClick={() => {
            m.reset();
            setOpen(true);
          }}
        >
          <Plus size={17} />
          Tambah departemen
        </button>
      </Heading>
      <ErrorBox error={q.error} />
      {q.isLoading ? (
        <Loading />
      ) : (
        <div className="department-cards">
          {q.data?.map((d) => (
            <section className="panel department-card" key={d.id}>
              <div className="stat-icon green">
                <Building2 size={22} />
              </div>
              <h2>{d.name}</h2>
              <p>{d.count} karyawan aktif</p>
              <NavLink to="/employees">
                Lihat direktori <ArrowUpRight size={16} />
              </NavLink>
            </section>
          ))}
        </div>
      )}
      {!q.isLoading && !q.data?.length && <Empty />}
      {open && (
        <Modal title="Tambah departemen" close={() => setOpen(false)}>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              m.mutate(Object.fromEntries(new FormData(e.currentTarget)));
            }}
          >
            <label>
              Nama departemen
              <input
                autoFocus
                name="name"
                required
                minLength={2}
                maxLength={100}
              />
            </label>
            <ErrorBox error={m.error} />
            <div className="modal-actions">
              <button className="primary" disabled={m.isPending}>
                Simpan departemen
              </button>
            </div>
          </form>
        </Modal>
      )}
    </>
  );
}

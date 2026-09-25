import { Badge, date, Empty, initials } from "./common";
import type { Employee } from "../types";
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

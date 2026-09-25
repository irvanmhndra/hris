export interface User {
  id: number;
  company_id: number;
  employee_id: number | null;
  name: string;
  email: string;
  role: "admin" | "employee";
  company_name: string;
}
export interface Employee {
  id: number;
  code: string;
  name: string;
  email: string;
  department_id: number;
  department: string;
  position: string;
  status: "active" | "inactive";
  joined_on: string;
}
export interface Department {
  id: number;
  name: string;
  count: number;
}
export interface Attendance {
  id: number;
  employee_id: number;
  name: string;
  date: string;
  check_in: string;
  check_out: string | null;
}
export interface Leave {
  calculation: string;
  id: number;
  employee_id: number;
  name: string;
  kind: string;
  start_date: string;
  end_date: string;
  reason: string;
  status: string;
  days: number;
}
export interface Dashboard {
  employees: number;
  present: number;
  pending: number;
  departments: number;
}
export interface HRItem {
  id: number;
  module: string;
  employee_id: number | null;
  employee_name: string;
  title: string;
  description: string;
  status: string;
  due_date: string;
  data: Record<string, string | number>;
  version: number;
  review_note: string;
  created_at: string;
}
export interface WorkCalendar {
  workdays: number[];
  annual_allowance: number;
  start_time: string;
  end_time: string;
}
export interface Holiday {
  id: number;
  date: string;
  name: string;
}
export interface Balance {
  employee_id: number;
  name: string;
  year: number;
  allowance: number;
  used: number;
  reserved: number;
  available: number;
}
export interface Profile {
  employee_id: number;
  name: string;
  email: string;
  code: string;
  department: string;
  position: string;
  joined_on: string;
  phone: string;
  address: string;
  emergency_name: string;
  emergency_phone: string;
  emergency_relation: string;
}
export interface AuditLog {
  id: number;
  actor: string;
  action: string;
  resource: string;
  resource_id: number;
  summary: string;
  created_at: string;
}
export interface Salary {
  employee_id: number;
  name: string;
  code: string;
  configured: boolean;
  basic_salary: number;
  allowance: number;
  deduction: number;
  note: string;
}
export interface PayrollRun {
  id: number;
  period: string;
  status: string;
  employees: number;
  total: number;
  payment_reference: string;
  created_at: string;
}
export interface Payslip {
  id: number;
  run_id: number;
  employee_id: number;
  employee_name: string;
  employee_code: string;
  position: string;
  period: string;
  status: string;
  basic_salary: number;
  allowance: number;
  deduction: number;
  net: number;
  note: string;
  version: number;
}

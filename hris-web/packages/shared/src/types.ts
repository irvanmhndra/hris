export interface User {
  id: number;
  company_id: number;
  employee_id: number | null;
  name: string;
  email: string;
  role: "admin" | "employee";
  company_name: string;
  is_manager: boolean;
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
  left_on: string | null;
  manager_id: number | null;
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
  stage: "manager" | "hr";
  review_note: string;
  days: number;
}
export interface TeamRequest {
  type: "leave" | "overtime" | "corrections";
  id: number;
  employee_name: string;
  kind: string;
  start_date: string | null;
  end_date: string | null;
  title: string;
  reason: string;
  data: Record<string, string>;
  version: number;
  created_at: string;
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
  stage: "manager" | "hr";
  review_note: string;
  created_at: string;
}
export interface WorkCalendar {
  workdays: number[];
  annual_allowance: number;
  start_time: string;
  end_time: string;
  leave_accrual: "annual" | "monthly";
  carry_over_max: number;
  leave_eligibility_months: number;
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
  accrued: number;
  carried_over: number;
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
export interface SalaryComponent {
  kind: "allowance" | "deduction";
  name: string;
  amount: number;
  fixed: boolean;
  taxable: boolean;
}
export interface Salary {
  employee_id: number;
  name: string;
  code: string;
  configured: boolean;
  basic_salary: number;
  ptkp_status: string;
  tax_method: "gross" | "gross_up" | "none";
  bpjs_kesehatan: boolean;
  bpjs_ketenagakerjaan: boolean;
  bpjs_pensiun: boolean;
  overtime_eligible: boolean;
  note: string;
  components: SalaryComponent[];
}
export interface PayrollSettings {
  jkk_rate: number;
  jp_wage_cap: number;
  kes_wage_cap: number;
}
export interface PayrollRun {
  id: number;
  period: string;
  status: string;
  employees: number;
  total: number;
  tax: number;
  employer_cost: number;
  thr_date: string | null;
  payment_reference: string;
  created_at: string;
}
export interface PayrollLine {
  kind: "earning" | "deduction" | "employer";
  code: string;
  name: string;
  amount: number;
  taxable: boolean;
  fixed: boolean;
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
  ptkp_status: string;
  tax_method: string;
  final_period: boolean;
  worked_days: number;
  period_days: number;
  unpaid_leave_days: number;
  taxable_gross: number;
  pph21: number;
  employer_cost: number;
  note: string;
  version: number;
  lines: PayrollLine[];
}
export interface Pagination {
  current_page: number;
  per_page: number;
  total_records: number;
  total_pages: number;
}
export interface Page<T> {
  items: T[];
  pagination: Pagination;
}

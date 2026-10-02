-- Forward-only: reverting would discard payslip lines, salary components, and overtime payment links.
DO $$ BEGIN RAISE EXCEPTION 'Payroll Indonesia migration requires an explicit data-preserving rollback plan'; END $$;

-- Forward-only: reverting would discard salary history, payroll runs, and payslips.
DO $$ BEGIN RAISE EXCEPTION 'Payroll migration requires an explicit data-preserving rollback plan'; END $$;

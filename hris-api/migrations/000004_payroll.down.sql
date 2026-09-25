DO $$ BEGIN RAISE EXCEPTION 'Payroll migration requires an explicit data-preserving rollback plan'; END $$;

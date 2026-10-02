-- Forward-only: reverting would drop correction runs and tax identifiers.
DO $$ BEGIN RAISE EXCEPTION 'Payroll rules migration requires an explicit data-preserving rollback plan'; END $$;

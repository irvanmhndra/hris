-- Forward-only: reverting would discard balances, workflow records, and audit history.
DO $$ BEGIN RAISE EXCEPTION 'HR suite migration requires an explicit data-preserving rollback plan'; END $$;

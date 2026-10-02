-- Forward-only: reverting would reject unpaid leave rows and drop manager approvals.
DO $$ BEGIN RAISE EXCEPTION 'Leave policy migration requires an explicit data-preserving rollback plan'; END $$;

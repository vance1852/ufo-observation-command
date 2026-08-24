DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'signal_recovery_reports' AND column_name = 'value'
    ) THEN
        ALTER TABLE signal_recovery_reports RENAME COLUMN value TO risk_score;
    END IF;
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'signal_recovery_reports' AND column_name = 'unit'
    ) THEN
        ALTER TABLE signal_recovery_reports RENAME COLUMN unit TO scale;
    END IF;
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'signal_recovery_reports' AND column_name = 'limit_value'
    ) THEN
        ALTER TABLE signal_recovery_reports RENAME COLUMN limit_value TO alert_threshold;
    END IF;
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'signal_recovery_reports' AND column_name = 'measured_at'
    ) THEN
        ALTER TABLE signal_recovery_reports RENAME COLUMN measured_at TO observed_at;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'signal_recovery_reports_risk_score_nonnegative') THEN
        ALTER TABLE signal_recovery_reports
            ADD CONSTRAINT signal_recovery_reports_risk_score_nonnegative CHECK (risk_score >= 0);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'signal_recovery_reports_alert_threshold_nonnegative') THEN
        ALTER TABLE signal_recovery_reports
            ADD CONSTRAINT signal_recovery_reports_alert_threshold_nonnegative CHECK (alert_threshold >= 0);
    END IF;
END $$;

ALTER TABLE integrity_incidents DROP CONSTRAINT IF EXISTS integrity_incidents_kind_check;
UPDATE integrity_incidents SET kind = 'reassess' WHERE kind = 'retask';
ALTER TABLE integrity_incidents
    ADD CONSTRAINT integrity_incidents_kind_check CHECK (kind IN ('reassess', 'repeat_acoustic_buoy', 'safety_adjustment', 'close_record'));

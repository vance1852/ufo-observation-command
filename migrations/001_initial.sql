CREATE TABLE IF NOT EXISTS array_operators (
    id uuid PRIMARY KEY,
    name text NOT NULL,
    role text NOT NULL CHECK (role IN ('acoustic_buoy_operator', 'signal_recovery_report_array_operator', 'quality_reviewer', 'safety_supervisor')),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS survey_missions (
    id uuid PRIMARY KEY,
    code text NOT NULL UNIQUE,
    name text NOT NULL,
    status text NOT NULL CHECK (status IN ('draft', 'scheduled', 'active', 'closed')),
    timezone text NOT NULL,
    starts_at timestamptz NOT NULL,
    ends_at timestamptz NOT NULL,
    version bigint NOT NULL DEFAULT 1,
    created_by uuid NOT NULL REFERENCES array_operators(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (ends_at > starts_at)
);

CREATE TABLE IF NOT EXISTS acoustic_buoys (
    id uuid PRIMARY KEY,
    survey_mission_id uuid NOT NULL REFERENCES survey_missions(id) ON DELETE CASCADE,
    code text NOT NULL,
    arrayops_lane text NOT NULL,
    required_successes integer NOT NULL CHECK (required_successes > 0),
    completed_installs integer NOT NULL DEFAULT 0 CHECK (completed_installs >= 0),
    UNIQUE (survey_mission_id, code)
);

CREATE TABLE IF NOT EXISTS assignments (
    id uuid PRIMARY KEY,
    survey_mission_id uuid NOT NULL REFERENCES survey_missions(id) ON DELETE CASCADE,
    acoustic_buoy_id uuid NOT NULL REFERENCES acoustic_buoys(id) ON DELETE CASCADE,
    array_operator_id uuid NOT NULL REFERENCES array_operators(id),
    status text NOT NULL CHECK (status IN ('queued','active','completed','cancelled')),
    starts_at timestamptz NOT NULL,
    ends_at timestamptz NOT NULL,
    version bigint NOT NULL DEFAULT 1,
    UNIQUE (acoustic_buoy_id, starts_at),
    CHECK (ends_at > starts_at)
);

CREATE TABLE IF NOT EXISTS recovery_jobs (
    id uuid PRIMARY KEY,
    survey_mission_id uuid NOT NULL REFERENCES survey_missions(id),
    acoustic_buoy_id uuid NOT NULL REFERENCES acoustic_buoys(id),
    task_code text NOT NULL UNIQUE,
    status text NOT NULL CHECK (status IN ('queued', 'completed', 'activation_pending', 'accepted', 'in_progress', 'verified', 'rejected', 'archived')),
    completed_at timestamptz,
    accepted_at timestamptz,
    expires_at timestamptz NOT NULL,
    version bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS mooring_events (
    id uuid PRIMARY KEY,
    recovery_job_id uuid NOT NULL REFERENCES recovery_jobs(id) ON DELETE CASCADE,
    from_operator uuid REFERENCES array_operators(id),
    to_operator uuid NOT NULL REFERENCES array_operators(id),
    location text NOT NULL,
    recorded_at timestamptz NOT NULL,
    note text NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS dive_windows (
    id uuid PRIMARY KEY,
    code text NOT NULL UNIQUE,
    status text NOT NULL CHECK (status IN ('queued', 'running', 'completed', 'cancelled')),
    method text NOT NULL,
    capacity integer NOT NULL CHECK (capacity > 0),
    started_at timestamptz,
    completed_at timestamptz,
    version bigint NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS dive_window_items (
    dive_window_id uuid NOT NULL REFERENCES dive_windows(id) ON DELETE CASCADE,
    recovery_job_id uuid NOT NULL REFERENCES recovery_jobs(id),
    PRIMARY KEY (dive_window_id, recovery_job_id)
);

CREATE TABLE IF NOT EXISTS signal_recovery_reports (
    id uuid PRIMARY KEY,
    recovery_job_id uuid NOT NULL REFERENCES recovery_jobs(id),
    dive_window_id uuid NOT NULL REFERENCES dive_windows(id),
    recorded_by uuid NOT NULL REFERENCES array_operators(id),
    status text NOT NULL CHECK (status IN ('pending', 'verified', 'rejected')),
    value numeric(18,6) NOT NULL,
    unit text NOT NULL,
    limit_value numeric(18,6) NOT NULL,
    measured_at timestamptz NOT NULL,
    reviewed_at timestamptz,
    version bigint NOT NULL DEFAULT 1,
    UNIQUE (recovery_job_id, dive_window_id)
);

CREATE TABLE IF NOT EXISTS integrity_incidents (
    id uuid PRIMARY KEY,
    recovery_job_id uuid NOT NULL REFERENCES recovery_jobs(id),
    kind text NOT NULL CHECK (kind IN ('retask', 'repeat_acoustic_buoy', 'safety_adjustment', 'close_record')),
    status text NOT NULL CHECK (status IN ('open', 'in_progress', 'closed')),
    reason text NOT NULL,
    due_at timestamptz NOT NULL,
    closed_at timestamptz,
    UNIQUE (recovery_job_id, kind, status)
);

CREATE TABLE IF NOT EXISTS audit_events (
    id uuid PRIMARY KEY,
    request_id text NOT NULL,
    array_operator_id uuid REFERENCES array_operators(id),
    object_type text NOT NULL,
    object_id uuid NOT NULL,
    action text NOT NULL,
    outcome text NOT NULL,
    detail jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS idempotency_keys (
    key text PRIMARY KEY,
    request_hash text NOT NULL,
    response_code integer NOT NULL,
    response_body jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS recovery_jobs_survey_mission_status_idx ON recovery_jobs(survey_mission_id, status);
CREATE INDEX IF NOT EXISTS recovery_jobs_expiry_idx ON recovery_jobs(status, expires_at);
CREATE INDEX IF NOT EXISTS activation_task_time_idx ON mooring_events(recovery_job_id, recorded_at DESC);
CREATE INDEX IF NOT EXISTS signal_recovery_reports_status_idx ON signal_recovery_reports(status, measured_at);
CREATE INDEX IF NOT EXISTS integrity_incidents_due_idx ON integrity_incidents(status, due_at);
CREATE INDEX IF NOT EXISTS audit_object_idx ON audit_events(object_type, object_id, created_at DESC);

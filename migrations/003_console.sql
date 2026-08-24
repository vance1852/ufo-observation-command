CREATE TABLE console_users (
    id uuid PRIMARY KEY,
    username text NOT NULL UNIQUE,
    password_hash text NOT NULL,
    real_name text NOT NULL,
    phone text NOT NULL DEFAULT '',
    role text NOT NULL CHECK (role IN ('mission_lead', 'array_operator', 'reviewer')),
    status integer NOT NULL DEFAULT 1 CHECK (status IN (0, 1)),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE console_sessions (
    token_hash text PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES console_users(id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    CHECK (expires_at > created_at)
);

CREATE TABLE console_acoustic_buoys (
    id uuid PRIMARY KEY,
    title text NOT NULL,
    buoy_class integer NOT NULL CHECK (buoy_class IN (1, 2)),
    commissioned_on date,
    serial_number text NOT NULL DEFAULT '',
    shore_station_code text NOT NULL DEFAULT '',
    mooring_zone text NOT NULL DEFAULT '',
    manufacturer text NOT NULL DEFAULT '',
    emergency_contact text NOT NULL DEFAULT '',
    integrity_status integer NOT NULL DEFAULT 1 CHECK (integrity_status BETWEEN 1 AND 3),
    status integer NOT NULL DEFAULT 1 CHECK (status IN (0, 1)),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);

CREATE TABLE console_array_operators (
    id uuid PRIMARY KEY,
    name text NOT NULL,
    specialty_level integer NOT NULL CHECK (specialty_level IN (1, 2)),
    phone text NOT NULL DEFAULT '',
    skills text NOT NULL DEFAULT '',
    status integer NOT NULL DEFAULT 1 CHECK (status IN (0, 1)),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);

CREATE TABLE console_arrayops_profiles (
    id uuid PRIMARY KEY,
    name text NOT NULL,
    description text NOT NULL DEFAULT '',
    risk_budget numeric(10,2) NOT NULL DEFAULT 0 CHECK (risk_budget >= 0),
    duration_minutes integer NOT NULL DEFAULT 60 CHECK (duration_minutes BETWEEN 1 AND 1440),
    status integer NOT NULL DEFAULT 1 CHECK (status IN (0, 1)),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);

CREATE TABLE console_command_orders (
    id uuid PRIMARY KEY,
    work_order_no text NOT NULL UNIQUE,
    acoustic_buoy_id uuid NOT NULL REFERENCES console_acoustic_buoys(id) ON DELETE RESTRICT,
    arrayops_profile_id uuid NOT NULL REFERENCES console_arrayops_profiles(id) ON DELETE RESTRICT,
    array_operator_id uuid REFERENCES console_array_operators(id) ON DELETE SET NULL,
    scheduled_at timestamptz,
    status integer NOT NULL DEFAULT 0 CHECK (status BETWEEN 0 AND 3),
    remark text NOT NULL DEFAULT '',
    version bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE console_signal_recovery_reports (
    id uuid PRIMARY KEY,
    acoustic_buoy_id uuid NOT NULL REFERENCES console_acoustic_buoys(id) ON DELETE RESTRICT,
    packet_loss_percent numeric(5,1),
    water_temperature_c numeric(5,1),
    signal_noise_db numeric(8,1),
    clock_drift_ppm numeric(8,3),
    corruption_index numeric(8,3),
    remark text NOT NULL DEFAULT '',
    recorded_at timestamptz NOT NULL DEFAULT now(),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE console_logs (
    id uuid PRIMARY KEY,
    username text NOT NULL,
    operation text NOT NULL,
    method text NOT NULL,
    ip text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX console_sessions_user_expiry_idx ON console_sessions(user_id, expires_at) WHERE revoked_at IS NULL;
CREATE INDEX console_acoustic_buoys_search_idx ON console_acoustic_buoys(title, serial_number) WHERE deleted_at IS NULL;
CREATE INDEX console_array_operators_search_idx ON console_array_operators(name, phone) WHERE deleted_at IS NULL;
CREATE INDEX console_arrayops_profiles_search_idx ON console_arrayops_profiles(name) WHERE deleted_at IS NULL;
CREATE INDEX console_command_orders_status_time_idx ON console_command_orders(status, scheduled_at DESC);
CREATE INDEX console_signal_recovery_reports_acoustic_buoy_time_idx ON console_signal_recovery_reports(acoustic_buoy_id, recorded_at DESC);
CREATE INDEX console_logs_time_idx ON console_logs(created_at DESC);

INSERT INTO console_users(id, username, password_hash, real_name, phone, role) VALUES
('10000000-0000-0000-0000-000000000001', 'admin', '$2a$10$.yXjTzrlVGyHtUlJ3DtbXu/BsE8n0891/6TRfIRXfwaT5b8fqT6Iu', 'Array Mission Lead', '13800138000', 'mission_lead'),
('10000000-0000-0000-0000-000000000002', 'operator', '$2a$10$cLYkgpV5AiyvLDG4AgWHYu7lmGAS7J8vbDB1QMYRBbB4yayIyhZPy', 'Dive Window Operator', '13800138001', 'array_operator'),
('10000000-0000-0000-0000-000000000003', 'reviewer', '$2a$10$1mw/cjufLNttJ2KZ4U/jGeAozEZWGz1XzqWLRjNvvrSqeS/1V6iG2', 'Acoustic Quality Reviewer', '13800138002', 'reviewer');

INSERT INTO console_acoustic_buoys(id, title, buoy_class, commissioned_on, serial_number, shore_station_code, mooring_zone, manufacturer, emergency_contact, integrity_status) VALUES
('20000000-0000-0000-0000-000000000001', 'Hadal Echo One', 1, '2024-03-15', 'ABY-0001', 'SHORE-01', 'Trench North A', 'Pelagic Instruments', '+86-10-10000001', 1),
('20000000-0000-0000-0000-000000000002', 'Basin Relay Seven', 2, '2024-07-22', 'ABY-0002', 'SHORE-02', 'Basin Relay B', 'Ocean Signal Lab', '+86-10-10000002', 2),
('20000000-0000-0000-0000-000000000003', 'Thermocline Listener Three', 1, '2025-01-08', 'ABY-0003', 'SHORE-03', 'Slope East C', 'Pelagic Instruments', '+86-10-10000003', 1);

INSERT INTO console_array_operators(id, name, specialty_level, phone, skills) VALUES
('30000000-0000-0000-0000-000000000001', 'Wang Rui', 2, '13800138011', 'hydrophone decoding, dive scheduling'),
('30000000-0000-0000-0000-000000000002', 'Liu Qing', 2, '13800138012', 'command approval, mooring handoff'),
('30000000-0000-0000-0000-000000000003', 'Chen Mo', 1, '13800138013', 'integrity checks, anomaly triage');

INSERT INTO console_arrayops_profiles(id, name, description, risk_budget, duration_minutes) VALUES
('40000000-0000-0000-0000-000000000001', 'Hydrophone Clock Calibration', 'Measure and correct clock drift before a dive', 20.00, 60),
('40000000-0000-0000-0000-000000000002', 'Mooring Depth Adjustment', 'Apply a bounded depth correction command', 45.00, 120),
('40000000-0000-0000-0000-000000000003', 'Acoustic Relay Reconfiguration', 'Switch the redundant underwater relay path', 65.00, 180),
('40000000-0000-0000-0000-000000000004', 'Integrity Isolation Exit', 'Release a buoy from isolation after quality review', 80.00, 240);

INSERT INTO console_command_orders(id, work_order_no, acoustic_buoy_id, arrayops_profile_id, array_operator_id, scheduled_at, status, remark) VALUES
('50000000-0000-0000-0000-000000000001', 'CMD-20260819001', '20000000-0000-0000-0000-000000000001', '40000000-0000-0000-0000-000000000001', '30000000-0000-0000-0000-000000000001', now() + interval '1 hour', 0, 'Awaiting operator confirmation'),
('50000000-0000-0000-0000-000000000002', 'CMD-20260819002', '20000000-0000-0000-0000-000000000002', '40000000-0000-0000-0000-000000000002', '30000000-0000-0000-0000-000000000002', now() + interval '2 hours', 1, 'Approval in progress');

INSERT INTO console_signal_recovery_reports(id, acoustic_buoy_id, packet_loss_percent, water_temperature_c, signal_noise_db, clock_drift_ppm, corruption_index, remark, recorded_at) VALUES
('60000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000001', 0.2, 2.5, 45, 0.4, 0.1, 'Nominal recovered segment', now() - interval '2 hours'),
('60000000-0000-0000-0000-000000000002', '20000000-0000-0000-0000-000000000002', 2.1, 3.2, 70, 5.2, 0.4, 'Elevated relay noise floor', now() - interval '1 hour');

INSERT INTO console_logs(id, username, operation, method, ip)
VALUES ('70000000-0000-0000-0000-000000000001', 'admin', 'initial seed', 'migration.003', '127.0.0.1');

-- Minimal TeslaMate-compatible schema + sample data for verifying the v2
-- endpoints. Only columns referenced by the v1 / v2 SQL (and a few extras
-- where NOT NULL would otherwise reject inserts) are declared.
--
-- This file is not authoritative; it tracks just enough of TeslaMate's real
-- schema to exercise the handlers.

CREATE TABLE IF NOT EXISTS cars (
    id              smallint PRIMARY KEY,
    eid             bigint,
    vid             bigint,
    name            text,
    model           text,
    efficiency      double precision,
    inserted_at     timestamp,
    updated_at      timestamp,
    vin             text,
    trim_badging    text,
    exterior_color  text,
    spoiler_type    text,
    wheel_type      text
);

CREATE TABLE IF NOT EXISTS car_settings (
    id                       smallint PRIMARY KEY,
    suspend_min              int,
    suspend_after_idle_min   int,
    req_not_unlocked         boolean,
    free_supercharging       boolean,
    use_streaming_api        boolean
);

CREATE TABLE IF NOT EXISTS updates (
    id          bigserial PRIMARY KEY,
    car_id      smallint NOT NULL,
    start_date  timestamp,
    end_date    timestamp,
    version     text
);

CREATE TABLE IF NOT EXISTS settings (
    id                  smallint PRIMARY KEY,
    unit_of_length      text NOT NULL DEFAULT 'km',
    unit_of_temperature text NOT NULL DEFAULT 'C',
    preferred_range     text NOT NULL DEFAULT 'rated',
    language            text NOT NULL DEFAULT 'en'
);

CREATE TABLE IF NOT EXISTS addresses (
    id           bigserial PRIMARY KEY,
    name         text,
    road         text,
    house_number text,
    city         text
);

CREATE TABLE IF NOT EXISTS geofences (
    id           bigserial PRIMARY KEY,
    name         text
);

CREATE TABLE IF NOT EXISTS positions (
    id                      bigserial PRIMARY KEY,
    car_id                  smallint NOT NULL,
    date                    timestamp,
    latitude                double precision,
    longitude               double precision,
    battery_level           int,
    usable_battery_level    int,
    rated_battery_range_km  double precision,
    ideal_battery_range_km  double precision,
    outside_temp            double precision,
    inside_temp             double precision,
    odometer                double precision,
    speed                   int,
    power                   int,
    address_id              bigint REFERENCES addresses(id),
    geofence_id             bigint REFERENCES geofences(id)
);

CREATE TABLE IF NOT EXISTS drives (
    id                     bigserial PRIMARY KEY,
    car_id                 smallint NOT NULL,
    start_date             timestamp,
    end_date               timestamp,
    start_position_id      bigint REFERENCES positions(id),
    end_position_id        bigint REFERENCES positions(id),
    start_address_id       bigint REFERENCES addresses(id),
    end_address_id         bigint REFERENCES addresses(id),
    start_geofence_id      bigint REFERENCES geofences(id),
    end_geofence_id        bigint REFERENCES geofences(id),
    distance               double precision,
    duration_min           int,
    speed_max              int,
    power_max              int,
    power_min              int,
    start_km               double precision,
    end_km                 double precision,
    start_ideal_range_km   double precision,
    end_ideal_range_km     double precision,
    start_rated_range_km   double precision,
    end_rated_range_km     double precision,
    outside_temp_avg       double precision,
    inside_temp_avg        double precision
);

CREATE TABLE IF NOT EXISTS charging_processes (
    id                    bigserial PRIMARY KEY,
    car_id                smallint NOT NULL,
    start_date            timestamp,
    end_date              timestamp,
    position_id           bigint REFERENCES positions(id),
    address_id            bigint REFERENCES addresses(id),
    geofence_id           bigint REFERENCES geofences(id),
    charge_energy_added   double precision,
    charge_energy_used    double precision,
    cost                  double precision,
    start_ideal_range_km  double precision,
    end_ideal_range_km    double precision,
    start_rated_range_km  double precision,
    end_rated_range_km    double precision,
    start_battery_level   int,
    end_battery_level     int,
    duration_min          int,
    outside_temp_avg      double precision
);

CREATE TABLE IF NOT EXISTS charges (
    id                       bigserial PRIMARY KEY,
    charging_process_id      bigint REFERENCES charging_processes(id),
    date                     timestamp,
    battery_level            int,
    usable_battery_level     int,
    charge_energy_added      double precision,
    charger_actual_current   int,
    charger_voltage          int,
    charger_phases           int,
    charger_power            int,
    charger_pilot_current    int,
    conn_charge_cable        text,
    fast_charger_present     boolean DEFAULT false,
    fast_charger_brand       text,
    fast_charger_type        text,
    rated_battery_range_km   double precision,
    ideal_battery_range_km   double precision,
    outside_temp             double precision,
    not_enough_power_to_heat boolean
);

-- ============================================================================
-- Sample data
-- ============================================================================

INSERT INTO settings (id, unit_of_length, unit_of_temperature, preferred_range)
VALUES (1, 'km', 'C', 'rated');

INSERT INTO cars (id, eid, vid, name, model, efficiency, inserted_at, updated_at, vin, trim_badging, exterior_color, spoiler_type, wheel_type) VALUES
    (1, 100001, 200001, 'Pearl', 'S', 0.156, '2020-01-01 00:00:00', '2026-05-30 00:00:00', '5YJSA1E26KF000001', 'P100D', 'DeepBlue', 'None',     'Pinwheel18'),
    (2, 100002, 200002, 'Onyx',  '3', 0.149, '2021-06-01 00:00:00', '2026-05-30 00:00:00', '5YJ3E1EA5KF000002', '74D',   'MidnightSilver', 'None', 'Stiletto19');

INSERT INTO car_settings (id, suspend_min, suspend_after_idle_min, req_not_unlocked, free_supercharging, use_streaming_api) VALUES
    (1, 21, 15, false, true,  true),
    (2, 21, 15, false, false, true);

INSERT INTO updates (id, car_id, start_date, end_date, version) VALUES
    (1, 1, '2024-03-01 03:00:00', '2024-03-01 03:30:00', '2024.6.1'),
    (2, 1, '2025-09-15 02:00:00', '2025-09-15 02:30:00', '2025.32.4'),
    (3, 1, '2026-05-15 03:00:00', '2026-05-15 03:30:00', '2026.20.1');

INSERT INTO geofences (id, name) VALUES
    (1, '家'),
    (2, '公司'),
    (3, '超充站');

INSERT INTO addresses (id, name, road, house_number, city) VALUES
    (1, NULL, '人民路', '1号',  '上海'),
    (2, NULL, '世纪大道', '88号', '上海'),
    (3, '国家电网', '机场高速', '5号', '上海');

-- ----------------------------------------------------------------------------
-- positions: pair (start, end) for each drive; also serve as parking endpoints
-- ----------------------------------------------------------------------------
-- Drive 1: 家 → 公司, 2026-05-26 08:00 → 08:30
INSERT INTO positions (id, car_id, date, latitude, longitude, battery_level, usable_battery_level, rated_battery_range_km, ideal_battery_range_km, outside_temp, odometer, address_id, geofence_id) VALUES
    (1, 1, '2026-05-26 08:00:00', 31.2300, 121.4700, 90, 90, 450, 480, 22.0, 12000.0, 1, 1),
    (2, 1, '2026-05-26 08:30:00', 31.2400, 121.5000, 85, 85, 425, 455, 22.5, 12015.0, 2, 2);

-- Drive 2: 公司 → 超充站, 2026-05-26 18:00 → 18:20
INSERT INTO positions (id, car_id, date, latitude, longitude, battery_level, usable_battery_level, rated_battery_range_km, ideal_battery_range_km, outside_temp, odometer, address_id, geofence_id) VALUES
    (3, 1, '2026-05-26 18:00:00', 31.2400, 121.5000, 60, 60, 300, 320, 24.0, 12030.0, 2, 2),
    (4, 1, '2026-05-26 18:20:00', 31.2500, 121.5200, 55, 55, 275, 295, 24.5, 12042.0, 3, 3);

-- Drive 3: 超充站 → 家, 2026-05-26 19:00 → 19:30 (after charging)
INSERT INTO positions (id, car_id, date, latitude, longitude, battery_level, usable_battery_level, rated_battery_range_km, ideal_battery_range_km, outside_temp, odometer, address_id, geofence_id) VALUES
    (5, 1, '2026-05-26 19:00:00', 31.2500, 121.5200, 95, 95, 475, 505, 23.0, 12042.0, 3, 3),
    (6, 1, '2026-05-26 19:30:00', 31.2300, 121.4700, 88, 88, 440, 470, 22.5, 12057.0, 1, 1);

-- Drive 4: cross-month — 家 → 公司, 2026-06-02 08:00 → 08:30
INSERT INTO positions (id, car_id, date, latitude, longitude, battery_level, usable_battery_level, rated_battery_range_km, ideal_battery_range_km, outside_temp, odometer, address_id, geofence_id) VALUES
    (7, 1, '2026-06-02 08:00:00', 31.2300, 121.4700, 80, 80, 400, 425, 26.0, 12057.0, 1, 1),
    (8, 1, '2026-06-02 08:30:00', 31.2400, 121.5000, 76, 76, 380, 405, 26.5, 12072.0, 2, 2);

-- Mid-parking samples (for parking details endpoint sampling)
INSERT INTO positions (id, car_id, date, latitude, longitude, battery_level, usable_battery_level, rated_battery_range_km, outside_temp, odometer) VALUES
    (101, 1, '2026-05-26 12:00:00', 31.2400, 121.5000, 83, 83, 415, 23.0, 12015.0),
    (102, 1, '2026-05-26 15:00:00', 31.2400, 121.5000, 81, 81, 405, 25.0, 12015.0);

INSERT INTO drives (id, car_id, start_date, end_date, start_position_id, end_position_id, start_address_id, end_address_id, start_geofence_id, end_geofence_id, distance, duration_min, speed_max, power_max, power_min, start_km, end_km, start_ideal_range_km, end_ideal_range_km, start_rated_range_km, end_rated_range_km, outside_temp_avg, inside_temp_avg) VALUES
    (1, 1, '2026-05-26 08:00:00', '2026-05-26 08:30:00', 1, 2, 1, 2, 1, 2, 15.0, 30, 80, 120, -10, 12000, 12015, 480, 455, 450, 425, 22.2, 21.0),
    (2, 1, '2026-05-26 18:00:00', '2026-05-26 18:20:00', 3, 4, 2, 3, 2, 3, 12.0, 20, 75, 110, -8,  12030, 12042, 320, 295, 300, 275, 24.2, 22.0),
    (3, 1, '2026-05-26 19:00:00', '2026-05-26 19:30:00', 5, 6, 3, 1, 3, 1, 15.0, 30, 70, 100, -12, 12042, 12057, 505, 470, 475, 440, 22.7, 21.5),
    (4, 1, '2026-06-02 08:00:00', '2026-06-02 08:30:00', 7, 8, 1, 2, 1, 2, 15.0, 30, 78, 115, -8,  12057, 12072, 425, 405, 400, 380, 26.2, 22.0);

-- charging_processes: one DC fast charge between drives 2 and 3
INSERT INTO charging_processes (id, car_id, start_date, end_date, position_id, address_id, geofence_id, charge_energy_added, charge_energy_used, cost, start_ideal_range_km, end_ideal_range_km, start_rated_range_km, end_rated_range_km, start_battery_level, end_battery_level, duration_min, outside_temp_avg) VALUES
    (1, 1, '2026-05-26 18:25:00', '2026-05-26 18:55:00', 4, 3, 3, 30.0, 32.0, 18.5, 295, 505, 275, 475, 55, 95, 30, 24.0);

-- And a slow AC top-up at home overnight
INSERT INTO charging_processes (id, car_id, start_date, end_date, position_id, address_id, geofence_id, charge_energy_added, charge_energy_used, cost, start_ideal_range_km, end_ideal_range_km, start_rated_range_km, end_rated_range_km, start_battery_level, end_battery_level, duration_min, outside_temp_avg) VALUES
    (2, 1, '2026-05-26 22:00:00', '2026-05-27 06:00:00', 6, 1, 1, 20.0, 21.0, 8.0, 470, 580, 440, 540, 88, 99, 480, 18.0);

-- charges (time-series): a few rows per process, indicating fast / slow
INSERT INTO charges (charging_process_id, date, battery_level, usable_battery_level, charge_energy_added, charger_actual_current, charger_voltage, charger_phases, charger_power, charger_pilot_current, conn_charge_cable, fast_charger_present, fast_charger_brand, fast_charger_type, rated_battery_range_km, ideal_battery_range_km, outside_temp) VALUES
    (1, '2026-05-26 18:26:00', 56, 56,  1.0, 350, 480, NULL, 168, NULL, '<invalid>', true,  'Tesla', 'Combo', 285, 300, 24.0),
    (1, '2026-05-26 18:35:00', 70, 70, 10.0, 320, 480, NULL, 154, NULL, '<invalid>', true,  'Tesla', 'Combo', 350, 368, 24.2),
    (1, '2026-05-26 18:45:00', 85, 85, 22.0, 200, 480, NULL,  96, NULL, '<invalid>', true,  'Tesla', 'Combo', 425, 446, 24.5),
    (1, '2026-05-26 18:55:00', 95, 95, 30.0, 100, 480, NULL,  48, NULL, '<invalid>', true,  'Tesla', 'Combo', 475, 499, 24.5),

    (2, '2026-05-26 22:30:00', 90, 90,  2.5,  16, 230,    3,  11,   32, 'IEC',       false, NULL,    NULL,    480, 504, 18.0),
    (2, '2026-05-27 02:00:00', 95, 95, 12.0,  16, 230,    3,  11,   32, 'IEC',       false, NULL,    NULL,    520, 546, 17.0),
    (2, '2026-05-27 06:00:00', 99, 99, 20.0,  16, 230,    3,  11,   32, 'IEC',       false, NULL,    NULL,    540, 567, 16.0);

-- Bring sequences past the manually-inserted ids so subsequent inserts don't collide.
SELECT setval('addresses_id_seq',          (SELECT MAX(id) FROM addresses));
SELECT setval('geofences_id_seq',          (SELECT MAX(id) FROM geofences));
SELECT setval('positions_id_seq',          (SELECT MAX(id) FROM positions));
SELECT setval('drives_id_seq',             (SELECT MAX(id) FROM drives));
SELECT setval('charging_processes_id_seq', (SELECT MAX(id) FROM charging_processes));
SELECT setval('charges_id_seq',            (SELECT MAX(id) FROM charges));

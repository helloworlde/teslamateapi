-- Query-facing subset of TeslaMate v4.1.1, commit
-- d6c43bc8c48784da8f0b701945b80b20911b3d1a. This is not a full database clone.
-- Preserve the actual types of every charging metric and the relevant indexes.
-- Sources under https://github.com/teslamate-org/teslamate/blob/d6c43bc8c48784da8f0b701945b80b20911b3d1a/priv/repo/migrations/:
-- 20191117042320_add_cost_field_to_charges.exs: cost numeric(6,2)
-- 20200410112005_database_efficiency_improvements.exs: energy numeric(8,2), duration smallint
-- 20190717184003_add_fkey_indexes.exs: car_id indexes
-- 20191007105010_add_new_fkey_indexes.exs: drive geofence indexes
-- No invented (car_id, start_date) composite indexes.

CREATE TABLE cars (
    id smallint PRIMARY KEY,
    name text
);
CREATE TABLE settings (
    id bigint PRIMARY KEY,
    unit_of_length text NOT NULL DEFAULT 'km',
    unit_of_temperature text NOT NULL DEFAULT 'C'
);
CREATE TABLE geofences (
    id serial PRIMARY KEY,
    name text
);
CREATE TABLE drives (
    id serial PRIMARY KEY,
    car_id smallint NOT NULL REFERENCES cars(id),
    start_date timestamp NOT NULL,
    end_date timestamp,
    start_geofence_id integer REFERENCES geofences(id),
    end_geofence_id integer REFERENCES geofences(id)
);
CREATE INDEX drives_car_id_index ON drives(car_id);
CREATE INDEX drives_start_geofence_id_index ON drives(start_geofence_id);
CREATE INDEX drives_end_geofence_id_index ON drives(end_geofence_id);

CREATE TABLE charging_processes (
    id serial PRIMARY KEY,
    car_id smallint NOT NULL REFERENCES cars(id),
    start_date timestamp NOT NULL,
    end_date timestamp,
    geofence_id integer REFERENCES geofences(id),
    charge_energy_added numeric(8,2),
    charge_energy_used numeric(8,2),
    cost numeric(6,2),
    duration_min smallint
);
CREATE INDEX charging_processes_car_id_index ON charging_processes(car_id);

package v1

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	defaultDetailMaxPoints = 800
	minDetailMaxPoints     = 100
	maxDetailMaxPoints     = 3000
)

func detailMaxPoints(raw string) int {
	if raw == "" {
		return defaultDetailMaxPoints
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return defaultDetailMaxPoints
	}
	if value < minDetailMaxPoints {
		return minDetailMaxPoints
	}
	if value > maxDetailMaxPoints {
		return maxDetailMaxPoints
	}
	return value
}

func detailBucketCount(maxPoints int) int {
	buckets := maxPoints / 10
	if buckets < 1 {
		return 1
	}
	return buckets
}

func normalizedDetailSampleMode(mode, fallback string) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		mode = fallback
	}
	switch mode {
	case "full", "every_5s", "every_30s", "auto", "visual", "m4":
		return mode
	default:
		return fallback
	}
}

func driveDetailsQuery(driveID int, sampleMode string, maxPoints int) (string, []interface{}) {
	switch normalizedDetailSampleMode(sampleMode, "auto") {
	case "full":
		return fullDriveDetailsQuery, []interface{}{driveID}
	case "every_30s":
		return timeBucketDriveDetailsQuery(30), []interface{}{driveID}
	case "auto", "visual", "m4":
		return autoDriveDetailsQuery, []interface{}{driveID, maxPoints, detailBucketCount(maxPoints)}
	case "every_5s":
		fallthrough
	default:
		return timeBucketDriveDetailsQuery(5), []interface{}{driveID}
	}
}

func chargeDetailsQuery(chargeID int, sampleMode string, maxPoints int) (string, []interface{}) {
	switch normalizedDetailSampleMode(sampleMode, "auto") {
	case "full":
		return fullChargeDetailsQuery, []interface{}{chargeID}
	case "every_30s":
		return timeBucketChargeDetailsQuery(30), []interface{}{chargeID}
	case "every_5s":
		return timeBucketChargeDetailsQuery(5), []interface{}{chargeID}
	case "auto", "visual", "m4":
		fallthrough
	default:
		return autoChargeDetailsQuery, []interface{}{chargeID, maxPoints, detailBucketCount(maxPoints)}
	}
}

func timeBucketDriveDetailsQuery(bucketSeconds int) string {
	return fmt.Sprintf(`
			WITH raw AS (
				SELECT
					%s
				FROM positions
				WHERE drive_id = $1
			),
			bucketed_ids AS (
				SELECT DISTINCT ON (bucket)
					detail_id
				FROM (
					SELECT
						detail_id,
						date_trunc('minute', date) + (FLOOR(EXTRACT(SECOND FROM date) / %d) * %d) * INTERVAL '1 second' AS bucket
					FROM raw
				) bucketed
				ORDER BY bucket, detail_id ASC
			),
			route_extrema_ids AS (
				SELECT DISTINCT unnest(array_remove(array[
					(array_agg(detail_id ORDER BY date ASC))[1],
					(array_agg(detail_id ORDER BY date DESC))[1],
					(array_agg(detail_id ORDER BY latitude ASC NULLS LAST))[1],
					(array_agg(detail_id ORDER BY latitude DESC NULLS LAST))[1],
					(array_agg(detail_id ORDER BY longitude ASC NULLS LAST))[1],
					(array_agg(detail_id ORDER BY longitude DESC NULLS LAST))[1]
				], NULL)) AS detail_id
				FROM raw
			),
			sampled_ids AS (
				SELECT detail_id FROM bucketed_ids
				UNION
				SELECT detail_id FROM route_extrema_ids
			)
			SELECT
				raw.detail_id,
				raw.date,
				raw.latitude,
				raw.longitude,
				raw.speed,
				raw.power,
				raw.odometer,
				raw.battery_level,
				raw.usable_battery_level,
				raw.elevation,
				raw.inside_temp,
				raw.outside_temp,
				raw.is_climate_on,
				raw.fan_status,
				raw.driver_temp_setting,
				raw.passenger_temp_setting,
				raw.is_rear_defroster_on,
				raw.is_front_defroster_on,
				raw.est_battery_range_km,
				raw.ideal_battery_range_km,
				raw.rated_battery_range_km,
				raw.battery_heater,
				raw.battery_heater_on,
				raw.battery_heater_no_power
			FROM raw
			JOIN sampled_ids USING (detail_id)
			ORDER BY raw.detail_id ASC;`, driveDetailsSelectList, bucketSeconds, bucketSeconds)
}

func timeBucketChargeDetailsQuery(bucketSeconds int) string {
	return fmt.Sprintf(`
            SELECT * FROM (
                SELECT DISTINCT ON (date_trunc('minute', charges.date) + (FLOOR(EXTRACT(SECOND FROM charges.date) / %d) * %d) * INTERVAL '1 second')
					%s
				FROM charges
				JOIN charging_processes cp ON cp.id = charges.charging_process_id
				WHERE charging_process_id=$1
				ORDER BY date_trunc('minute', charges.date) + (FLOOR(EXTRACT(SECOND FROM charges.date) / %d) * %d) * INTERVAL '1 second', charges.id ASC
            ) bucketed
            ORDER BY detail_id ASC;`, bucketSeconds, bucketSeconds, chargeDetailsSelectList, bucketSeconds, bucketSeconds)
}

const driveDetailsSelectList = `id AS detail_id,
					date,
					latitude,
					longitude,
					COALESCE(speed, 0) AS speed,
					power,
					odometer,
					battery_level,
					usable_battery_level,
					elevation,
					inside_temp,
					outside_temp,
					is_climate_on,
					fan_status,
					driver_temp_setting,
					passenger_temp_setting,
					is_rear_defroster_on,
					is_front_defroster_on,
					est_battery_range_km,
					ideal_battery_range_km,
					rated_battery_range_km,
					battery_heater,
					battery_heater_on,
					battery_heater_no_power`

const fullDriveDetailsQuery = `
            SELECT
				` + driveDetailsSelectList + `
            FROM positions
            WHERE drive_id = $1
            ORDER BY id ASC;`

const autoDriveDetailsQuery = `
			WITH raw AS (
				SELECT
					` + driveDetailsSelectList + `
				FROM positions
				WHERE drive_id = $1
			),
			bounds AS (
				SELECT min(date) AS t0, max(date) AS t1, count(*) AS raw_count
				FROM raw
			),
			bucketed AS (
				SELECT
					raw.*,
					width_bucket(
						EXTRACT(EPOCH FROM raw.date)::double precision,
						EXTRACT(EPOCH FROM bounds.t0)::double precision,
						EXTRACT(EPOCH FROM bounds.t1)::double precision + 0.001,
						$3
					) AS bucket,
					lag(speed) OVER (ORDER BY detail_id) AS prev_speed,
					lag(power) OVER (ORDER BY detail_id) AS prev_power,
					lag(battery_level) OVER (ORDER BY detail_id) AS prev_battery_level,
					lag(usable_battery_level) OVER (ORDER BY detail_id) AS prev_usable_battery_level
				FROM raw, bounds
				WHERE bounds.raw_count > $2
			),
			sampled_ids AS (
				SELECT detail_id
				FROM raw, bounds
				WHERE bounds.raw_count <= $2
				UNION
				SELECT DISTINCT unnest(array_remove(array[
					(array_agg(detail_id ORDER BY date ASC))[1],
					(array_agg(detail_id ORDER BY date DESC))[1],
					(array_agg(detail_id ORDER BY speed ASC NULLS LAST))[1],
					(array_agg(detail_id ORDER BY speed DESC NULLS LAST))[1],
					(array_agg(detail_id ORDER BY power ASC NULLS LAST))[1],
					(array_agg(detail_id ORDER BY power DESC NULLS LAST))[1],
					(array_agg(detail_id ORDER BY elevation ASC NULLS LAST))[1],
					(array_agg(detail_id ORDER BY elevation DESC NULLS LAST))[1],
					(array_agg(detail_id ORDER BY battery_level ASC NULLS LAST))[1],
					(array_agg(detail_id ORDER BY battery_level DESC NULLS LAST))[1]
				], NULL))
				FROM bucketed
				GROUP BY bucket
				UNION
				SELECT DISTINCT unnest(array_remove(array[
					(array_agg(detail_id ORDER BY date ASC))[1],
					(array_agg(detail_id ORDER BY date DESC))[1],
					(array_agg(detail_id ORDER BY latitude ASC NULLS LAST))[1],
					(array_agg(detail_id ORDER BY latitude DESC NULLS LAST))[1],
					(array_agg(detail_id ORDER BY longitude ASC NULLS LAST))[1],
					(array_agg(detail_id ORDER BY longitude DESC NULLS LAST))[1]
				], NULL))
				FROM bucketed
				UNION
				SELECT detail_id
				FROM bucketed
				WHERE abs(COALESCE(speed, 0) - COALESCE(prev_speed, COALESCE(speed, 0))) >= 20
					OR abs(COALESCE(power, 0) - COALESCE(prev_power, COALESCE(power, 0))) >= 50
					OR battery_level IS DISTINCT FROM prev_battery_level
					OR usable_battery_level IS DISTINCT FROM prev_usable_battery_level
			)
			SELECT
				raw.detail_id,
				raw.date,
				raw.latitude,
				raw.longitude,
				raw.speed,
				raw.power,
				raw.odometer,
				raw.battery_level,
				raw.usable_battery_level,
				raw.elevation,
				raw.inside_temp,
				raw.outside_temp,
				raw.is_climate_on,
				raw.fan_status,
				raw.driver_temp_setting,
				raw.passenger_temp_setting,
				raw.is_rear_defroster_on,
				raw.is_front_defroster_on,
				raw.est_battery_range_km,
				raw.ideal_battery_range_km,
				raw.rated_battery_range_km,
				raw.battery_heater,
				raw.battery_heater_on,
				raw.battery_heater_no_power
			FROM raw
			JOIN sampled_ids USING (detail_id)
			ORDER BY raw.detail_id ASC;`

const chargeDetailsSelectList = `charges.id AS detail_id,
				charges.date,
				battery_level,
				usable_battery_level,
				charges.charge_energy_added,
				not_enough_power_to_heat,
				COALESCE(charger_actual_current, 0) as charger_actual_current,
				COALESCE(charger_phases, 0) AS charger_phases,
				COALESCE(charger_pilot_current, 0) as charger_pilot_current,
				COALESCE(charger_power, 0) as charger_power,
				COALESCE(charger_voltage, 0) as charger_voltage,
				ideal_battery_range_km AS ideal_battery_range,
				rated_battery_range_km AS rated_battery_range,
				battery_heater,
				battery_heater_on,
				battery_heater_no_power,
				conn_charge_cable,
				fast_charger_present,
				fast_charger_brand,
				fast_charger_type,
				outside_temp`

const fullChargeDetailsQuery = `
			SELECT
				` + chargeDetailsSelectList + `
			FROM charges
			JOIN charging_processes cp ON cp.id = charges.charging_process_id
			WHERE charging_process_id=$1
			ORDER BY charges.id ASC;`

const autoChargeDetailsQuery = `
			WITH raw AS (
				SELECT
					` + chargeDetailsSelectList + `
				FROM charges
				JOIN charging_processes cp ON cp.id = charges.charging_process_id
				WHERE charging_process_id=$1
			),
			bounds AS (
				SELECT min(date) AS t0, max(date) AS t1, count(*) AS raw_count
				FROM raw
			),
			bucketed AS (
				SELECT
					raw.*,
					width_bucket(
						EXTRACT(EPOCH FROM raw.date)::double precision,
						EXTRACT(EPOCH FROM bounds.t0)::double precision,
						EXTRACT(EPOCH FROM bounds.t1)::double precision + 0.001,
						$3
					) AS bucket,
					lag(battery_level) OVER (ORDER BY detail_id) AS prev_battery_level,
					lag(usable_battery_level) OVER (ORDER BY detail_id) AS prev_usable_battery_level,
					lag(charger_phases) OVER (ORDER BY detail_id) AS prev_charger_phases,
					lag(charger_power) OVER (ORDER BY detail_id) AS prev_charger_power,
					lag(charger_voltage) OVER (ORDER BY detail_id) AS prev_charger_voltage,
					lag(conn_charge_cable) OVER (ORDER BY detail_id) AS prev_conn_charge_cable,
					lag(fast_charger_present) OVER (ORDER BY detail_id) AS prev_fast_charger_present,
					lag(fast_charger_brand) OVER (ORDER BY detail_id) AS prev_fast_charger_brand,
					lag(fast_charger_type) OVER (ORDER BY detail_id) AS prev_fast_charger_type
				FROM raw, bounds
				WHERE bounds.raw_count > $2
			),
			sampled_ids AS (
				SELECT detail_id
				FROM raw, bounds
				WHERE bounds.raw_count <= $2
				UNION
				SELECT DISTINCT unnest(array_remove(array[
					(array_agg(detail_id ORDER BY date ASC))[1],
					(array_agg(detail_id ORDER BY date DESC))[1],
					(array_agg(detail_id ORDER BY charger_power ASC NULLS LAST))[1],
					(array_agg(detail_id ORDER BY charger_power DESC NULLS LAST))[1],
					(array_agg(detail_id ORDER BY charger_actual_current ASC NULLS LAST))[1],
					(array_agg(detail_id ORDER BY charger_actual_current DESC NULLS LAST))[1],
					(array_agg(detail_id ORDER BY charger_voltage ASC NULLS LAST))[1],
					(array_agg(detail_id ORDER BY charger_voltage DESC NULLS LAST))[1],
					(array_agg(detail_id ORDER BY charge_energy_added ASC NULLS LAST))[1],
					(array_agg(detail_id ORDER BY charge_energy_added DESC NULLS LAST))[1],
					(array_agg(detail_id ORDER BY outside_temp ASC NULLS LAST))[1],
					(array_agg(detail_id ORDER BY outside_temp DESC NULLS LAST))[1]
				], NULL))
				FROM bucketed
				GROUP BY bucket
				UNION
				SELECT detail_id
				FROM bucketed
				WHERE battery_level IS DISTINCT FROM prev_battery_level
					OR usable_battery_level IS DISTINCT FROM prev_usable_battery_level
					OR charger_phases IS DISTINCT FROM prev_charger_phases
					OR abs(COALESCE(charger_power, 0) - COALESCE(prev_charger_power, COALESCE(charger_power, 0))) >= 5
					OR abs(COALESCE(charger_voltage, 0) - COALESCE(prev_charger_voltage, COALESCE(charger_voltage, 0))) >= 20
					OR conn_charge_cable IS DISTINCT FROM prev_conn_charge_cable
					OR fast_charger_present IS DISTINCT FROM prev_fast_charger_present
					OR fast_charger_brand IS DISTINCT FROM prev_fast_charger_brand
					OR fast_charger_type IS DISTINCT FROM prev_fast_charger_type
			)
			SELECT
				raw.detail_id,
				raw.date,
				raw.battery_level,
				raw.usable_battery_level,
				raw.charge_energy_added,
				raw.not_enough_power_to_heat,
				raw.charger_actual_current,
				raw.charger_phases,
				raw.charger_pilot_current,
				raw.charger_power,
				raw.charger_voltage,
				raw.ideal_battery_range,
				raw.rated_battery_range,
				raw.battery_heater,
				raw.battery_heater_on,
				raw.battery_heater_no_power,
				raw.conn_charge_cable,
				raw.fast_charger_present,
				raw.fast_charger_brand,
				raw.fast_charger_type,
				raw.outside_temp
			FROM raw
			JOIN sampled_ids USING (detail_id)
			ORDER BY raw.detail_id ASC;`

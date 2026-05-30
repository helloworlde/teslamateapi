// Package command holds the Tesla owner-api / TeslaMate logging command
// allow-list and the small region helpers used by the v1 command handler.
package command

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/tobiasehlert/teslamateapi/internal/config"
)

// AllowList holds the resolved set of `/command/...` and `/logging/...`
// paths permitted by the current configuration. Construct via NewAllowList
// and treat the value as immutable.
type AllowList struct {
	items []string
}

// Items returns the underlying allow-list. Do not mutate.
func (a *AllowList) Items() []string { return a.items }

// Contains reports whether the given command path is in the allow-list.
func (a *AllowList) Contains(cmd string) bool {
	for _, v := range a.items {
		if v == cmd {
			return true
		}
	}
	return false
}

// commandList groups the available commands by environment-variable feature
// flag. Mirrors the source-of-truth in the legacy initCommandAllowList.
func commandList() map[string][]string {
	return map[string][]string{
		// https://github.com/teslamate-org/teslamate/discussions/1433
		"COMMANDS_LOGGING": {
			"/logging/resume",
			"/logging/suspend",
		},
		// https://tesla-api.timdorr.com/vehicle/commands/wake
		"COMMANDS_WAKE": {
			"/wake_up",
		},
		// https://tesla-api.timdorr.com/vehicle/commands/alerts
		"COMMANDS_ALERT": {
			"/command/honk_horn",
			"/command/flash_lights",
		},
		// https://tesla-api.timdorr.com/vehicle/commands/remotestart
		"COMMANDS_REMOTESTART": {
			"/command/remote_start_drive",
		},
		// https://tesla-api.timdorr.com/vehicle/commands/homelink
		"COMMANDS_HOMELINK": {
			"/command/trigger_homelink",
		},
		// https://tesla-api.timdorr.com/vehicle/commands/speedlimit
		"COMMANDS_SPEEDLIMIT": {
			"/command/speed_limit_set_limit",
			"/command/speed_limit_activate",
			"/command/speed_limit_deactivate",
			"/command/speed_limit_clear_pin",
		},
		// https://tesla-api.timdorr.com/vehicle/commands/valet
		"COMMANDS_VALET": {
			"/command/set_valet_mode",
			"/command/reset_valet_pin",
		},
		// https://tesla-api.timdorr.com/vehicle/commands/sentrymode
		"COMMANDS_SENTRYMODE": {
			"/command/set_sentry_mode",
		},
		// https://tesla-api.timdorr.com/vehicle/commands/doors
		"COMMANDS_DOORS": {
			"/command/door_unlock",
			"/command/door_lock",
		},
		// https://tesla-api.timdorr.com/vehicle/commands/trunk
		"COMMANDS_TRUNK": {
			"/command/actuate_trunk",
		},
		// https://tesla-api.timdorr.com/vehicle/commands/windows
		"COMMANDS_WINDOWS": {
			"/command/window_control",
		},
		// https://tesla-api.timdorr.com/vehicle/commands/sunroof
		"COMMANDS_SUNROOF": {
			"/command/sun_roof_control",
		},
		// https://tesla-api.timdorr.com/vehicle/commands/charging
		"COMMANDS_CHARGING": {
			"/command/charge_port_door_open",
			"/command/charge_port_door_close",
			"/command/charge_start",
			"/command/charge_stop",
			"/command/charge_standard",
			"/command/charge_max_range",
			"/command/set_charge_limit",
			"/command/set_charging_amps",
			"/command/set_scheduled_charging",
			"/command/set_scheduled_departure",
		},
		// https://tesla-api.timdorr.com/vehicle/commands/climate
		"COMMANDS_CLIMATE": {
			"/command/auto_conditioning_start",
			"/command/auto_conditioning_stop",
			"/command/set_temps",
			"/command/set_preconditioning_max",
			"/command/remote_seat_heater_request",
			"/command/remote_seat_cooler_request",
			"/command/remote_steering_wheel_heater_request",
			"/command/set_bioweapon_mode",
			"/command/set_climate_keeper_mode",
			"/command/remote_auto_seat_climate_request",
			"/command/set_cop_temp",
			"/command/set_cabin_overheat_protection",
			"/command/remote_auto_steering_wheel_heat_climate_request",
			"/command/remote_steering_wheel_heat_level_request",
		},
		// https://tesla-api.timdorr.com/vehicle/commands/media
		"COMMANDS_MEDIA": {
			"/command/media_toggle_playback",
			"/command/media_next_track",
			"/command/media_prev_track",
			"/command/media_next_fav",
			"/command/media_prev_fav",
			"/command/media_volume_up",
			"/command/media_volume_down",
			"/command/adjust_volume",
		},
		// https://tesla-api.timdorr.com/vehicle/commands/sharing
		"COMMANDS_SHARING": {
			"/command/share",
		},
		// https://tesla-api.timdorr.com/vehicle/commands/softwareupdate
		"COMMANDS_SOFTWAREUPDATE": {
			"/command/schedule_software_update",
			"/command/cancel_software_update",
		},
		// not documented and unsorted new endpoints
		"COMMANDS_UNKNOWN": {
			"/command/upcoming_calendar_entries",
			"/command/dashcam_save_clip",
			"/command/navigation_sc_request",
			"/command/remote_boombox",
			"/command/get_active_route",
			"/command/get_managed_charging_sites",
			"/command/add_managed_charging_site",
			"/command/remove_managed_charging_site",
			"/command/update_charge_on_solar_feature",
			"/command/get_charge_on_solar_feature",
			"/command/take_drivenote",
			"/command/navigation_gps_request",
		},
	}
}

// NewAllowList resolves the allow-list from cfg's per-feature toggles
// (with COMMANDS_ALL trumping individual flags) and falls back to reading
// COMMANDS_ALLOWLIST as a JSON file when no env feature is enabled.
func NewAllowList(cfg config.Config) *AllowList {
	a := &AllowList{}
	envEnabled := map[string]bool{
		"COMMANDS_LOGGING":        cfg.CommandsLogging,
		"COMMANDS_WAKE":           cfg.CommandsWake,
		"COMMANDS_ALERT":          cfg.CommandsAlert,
		"COMMANDS_REMOTESTART":    cfg.CommandsRemotestart,
		"COMMANDS_HOMELINK":       cfg.CommandsHomelink,
		"COMMANDS_SPEEDLIMIT":     cfg.CommandsSpeedlimit,
		"COMMANDS_VALET":          cfg.CommandsValet,
		"COMMANDS_SENTRYMODE":     cfg.CommandsSentryMode,
		"COMMANDS_DOORS":          cfg.CommandsDoors,
		"COMMANDS_TRUNK":          cfg.CommandsTrunk,
		"COMMANDS_WINDOWS":        cfg.CommandsWindows,
		"COMMANDS_SUNROOF":        cfg.CommandsSunroof,
		"COMMANDS_CHARGING":       cfg.CommandsCharging,
		"COMMANDS_CLIMATE":        cfg.CommandsClimate,
		"COMMANDS_MEDIA":          cfg.CommandsMedia,
		"COMMANDS_SHARING":        cfg.CommandsSharing,
		"COMMANDS_SOFTWAREUPDATE": cfg.CommandsSoftwareupdate,
		"COMMANDS_UNKNOWN":        cfg.CommandsUnknown,
	}
	for key, items := range commandList() {
		if envEnabled[key] || cfg.CommandsAll {
			a.items = append(a.items, items...)
		}
	}

	if len(a.items) == 0 {
		f, err := os.Open(cfg.CommandsAllowList)
		if err != nil {
			log.Println("[error] getAllowList error with COMMANDS_ALLOWLIST: " + cfg.CommandsAllowList + " not found and will be ignored")
			return a
		}
		defer f.Close()
		byteValue, err := io.ReadAll(f)
		if err != nil {
			log.Println("[error] getAllowList error while reading COMMANDS_ALLOWLIST: " + cfg.CommandsAllowList + " it will be ignored")
			return a
		}
		var allowListFile []string
		if err := json.Unmarshal(byteValue, &allowListFile); err != nil {
			log.Println("[error] getAllowList error while parsing JSON.. COMMANDS_ALLOWLIST: " + cfg.CommandsAllowList + " it will be ignored")
			return a
		}
		a.items = append(a.items, allowListFile...)
	} else {
		log.Print("[info] getAllowList COMMANDS from environment variables set, " + cfg.CommandsAllowList + " will be ignored.")
	}

	if gin.IsDebugging() {
		log.Println("[info] initCommandAllowList - generated following list of allowed commands: " + strings.Join(a.items, ", "))
	}
	return a
}

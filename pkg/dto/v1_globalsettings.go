package dto

// V1GlobalSettingsAccountInfo captures inserted/updated audit timestamps.
type V1GlobalSettingsAccountInfo struct {
	InsertedAt string `json:"inserted_at" example:"2020-01-01T00:00:00+01:00"`
	UpdatedAt  string `json:"updated_at" example:"2024-01-01T00:00:00+01:00"`
}

// V1GlobalSettingsTeslaMateUnits captures the user's preferred display units.
type V1GlobalSettingsTeslaMateUnits struct {
	UnitsLength      string `json:"unit_of_length" example:"km" enums:"km,mi"`
	UnitsTemperature string `json:"unit_of_temperature" example:"C" enums:"C,F"`
}

// V1GlobalSettingsTeslaMateGUI captures GUI preferences.
type V1GlobalSettingsTeslaMateGUI struct {
	PreferredRange string `json:"preferred_range" example:"rated" enums:"rated,ideal"`
	Language       string `json:"language" example:"en"`
}

// V1GlobalSettingsTeslaMateURLs captures TeslaMate / Grafana URLs.
type V1GlobalSettingsTeslaMateURLs struct {
	BaseURL    string `json:"base_url" example:"https://teslamate.example.com"`
	GrafanaURL string `json:"grafana_url" example:"https://grafana.example.com"`
}

// V1GlobalSettings represents the single row of TeslaMate settings.
type V1GlobalSettings struct {
	SettingID      int                            `json:"setting_id" example:"1"`
	AccountInfo    V1GlobalSettingsAccountInfo    `json:"account_info"`
	TeslaMateUnits V1GlobalSettingsTeslaMateUnits `json:"teslamate_units"`
	TeslaMateGUI   V1GlobalSettingsTeslaMateGUI   `json:"teslamate_webgui"`
	TeslaMateURLs  V1GlobalSettingsTeslaMateURLs  `json:"teslamate_urls"`
}

// V1GlobalSettingsData is the `data` field of V1GlobalSettingsResponse.
type V1GlobalSettingsData struct {
	GlobalSettings V1GlobalSettings `json:"settings"`
}

// V1GlobalSettingsResponse is the envelope for /api/v1/globalsettings.
type V1GlobalSettingsResponse struct {
	Data V1GlobalSettingsData `json:"data"`
}

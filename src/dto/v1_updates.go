package dto

// V1Update is one row in /api/v1/cars/{CarID}/updates.
type V1Update struct {
	UpdateID  int    `json:"update_id" example:"42"`
	StartDate string `json:"start_date" example:"2024-01-01T20:00:00+01:00"`
	EndDate   string `json:"end_date" example:"2024-01-01T20:30:00+01:00"`
	Version   string `json:"version" example:"2024.32.12.2"`
}

// V1UpdatesData is the `data` field of V1UpdatesResponse.
type V1UpdatesData struct {
	Car     Car        `json:"car"`
	Updates []V1Update `json:"updates"`
}

// V1UpdatesResponse is the envelope for /api/v1/cars/{CarID}/updates.
type V1UpdatesResponse struct {
	Data V1UpdatesData `json:"data"`
}

package main

type SwaggerMessageResponse struct {
	Message string `json:"message" example:"pong"`
	Path    string `json:"path,omitempty" example:"/api/v1"`
}

type SwaggerErrorResponse struct {
	Error string `json:"error" example:"Unable to load summary."`
}

type SwaggerDataResponse struct {
	Data map[string]interface{} `json:"data"`
}

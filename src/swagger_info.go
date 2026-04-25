package main

// @title TeslaMateApi
// @version 1.0
// @description RESTful API for TeslaMate-backed vehicle, charge, drive, statistics, chart, timeline, and insight data. Original TeslaMateApi routes remain compatible; redesigned extension routes may introduce breaking changes. Date query parameters support RFC3339, timezone offsets, decoded-space offsets, local datetime, and date-only formats. When using `+08:00` in URLs, prefer `%2B08:00`, though decoded-space offsets are also accepted.
// @BasePath /api
// @schemes http https
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

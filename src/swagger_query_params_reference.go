package main

/*
Analytics endpoints share these query parameters (copy the @Param lines into each handler godoc).

Standard analytics block:

	@Param period query string false "Aggregation period" Enums(day, week, month, quarter, year, custom)
	@Param start query string false "Start datetime in RFC3339 format"
	@Param end query string false "End datetime in RFC3339 format"
	@Param timezone query string false "IANA timezone"
	@Param compare query string false "Comparison mode" Enums(none, previous_period)

Comparison is only supported on /analytics/summary; non-summary endpoints omit the compare line.
*/

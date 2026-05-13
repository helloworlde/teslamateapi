package v1

/*
Analytics endpoints share these query parameters (copy the @Param lines into each handler godoc).

Standard analytics block:

	@Param period query string false "聚合周期" Enums(day, week, month, quarter, year, custom)
	@Param start query string false "起始时间（RFC3339）"
	@Param end query string false "结束时间（RFC3339）"
	@Param timezone query string false "IANA 时区"
	@Param compare query string false "对比模式" Enums(none, previous_period)

Comparison is only supported on /analytics/summary; non-summary endpoints omit the compare line.
*/

# Agent Notes

## Time Handling

- TeslaMate timestamp columns are UTC instants stored as `timestamp without time zone`.
- For API output, scan DB timestamps as UTC and format through `timeInTZ` / `timefmt.GetTimeInTimeZone`.
- For user-supplied date filters, parse with the handler `parseDate` helper so RFC3339 offsets and local naive timestamps are normalized to UTC before SQL comparison.
- For local calendar bucketing in SQL (`date_trunc`, `EXTRACT` hour/day/month, weekday heatmaps, local day/month counts), do not use `ts AT TIME ZONE $tz` directly on TeslaMate timestamp columns. First mark the column as UTC, then convert to the user's timezone:

```sql
((ts AT TIME ZONE 'UTC') AT TIME ZONE $tz)
```

- Prefer the shared helper `timefmt.UTCTimestampToLocalSQL(tsExpr, tzParam)` for that expression.
- When calculating elapsed time against an open-ended UTC timestamp column, do not subtract raw `NOW()` from the column. Use `timefmt.UTCNowSQL()` so both sides are UTC `timestamp without time zone`.

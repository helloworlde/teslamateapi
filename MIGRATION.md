# Extension API Migration

## Calendar

- 旧: `/api/v1/cars/:CarID/calendar/drives`
- 旧: `/api/v1/cars/:CarID/calendar/charges`
- 新: `/api/v1/cars/:CarID/calendar`

## Drive charts

- 旧: `/api/v1/cars/:CarID/charts/drives/distance`
- 旧: `/api/v1/cars/:CarID/charts/drives/efficiency`
- 旧: `/api/v1/cars/:CarID/charts/drives/energy`
- 旧: `/api/v1/cars/:CarID/charts/drives/speed`
- 新: `/api/v1/cars/:CarID/series?scope=drives&metrics=distance,efficiency,energy,speed`

## Charge charts

- 旧: `/api/v1/cars/:CarID/charts/charges/*`
- 新: `/api/v1/cars/:CarID/series?scope=charges&metrics=...`

## Statistics and overview

- 旧: `/api/v1/cars/:CarID/analytics/activity`
- 旧: `/api/v1/cars/:CarID/summary`
- 旧: `/api/v1/cars/:CarID/charts/overview`
- 新: `/api/v1/cars/:CarID/statistics` 或 `/api/v1/cars/:CarID/dashboard`

## Regeneration

- 旧: `/api/v1/cars/:CarID/analytics/regeneration`
- 新: `/api/v1/cars/:CarID/statistics` 或 `/api/v1/cars/:CarID/series?metrics=regeneration`

## Insight events

- 旧: `/api/v1/cars/:CarID/insights/events`
- 新: `/api/v1/cars/:CarID/insights` 或 `/api/v1/cars/:CarID/timeline`

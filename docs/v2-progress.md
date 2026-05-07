# V2 API Implementation Progress

## Rules

- Each completed task must update this file.
- Each API must have handler, service, repository, response DTO, Swagger comments, and tests.
- Each API must be visible in Scalar documentation.

## Progress

| Task | Status | PR/Commit | APIs | Swagger | Scalar | Tests | Notes |
|---|---|---|---|---|---|---|---|
| T00 | Done |  | V2 base framework, `/api/v2`, `/api/docs`, `/api/docs/swagger.json`, `/api/docs/scalar` | Yes | Yes | Yes | scalar-go wired to the generated OpenAPI JSON handler. |
| T01 | Done |  | `/api/v2/cars/{CarID}/analytics/summary` | Yes | Yes | Yes | Supports `none` and `previous_period`; `previous_year` and `lifetime_average` return explicit not implemented errors. |
| T02 | Pending |  | driving | No | No | No |  |
| T03 | Pending |  | charging | No | No | No |  |
| T04 | Pending |  | parking | No | No | No |  |
| T05 | Pending |  | battery | No | No | No |  |
| T06 | Pending |  | efficiency | No | No | No |  |
| T07 | Pending |  | cost | No | No | No |  |
| T08 | Pending |  | locations | No | No | No |  |
| T09 | Pending |  | updates | No | No | No |  |
| T10 | Pending |  | lifecycle, timeline | No | No | No |  |
| T11 | Pending |  | calendar | No | No | No |  |
| T12 | Pending |  | reports | No | No | No |  |
| T13 | Pending |  | insights | No | No | No |  |

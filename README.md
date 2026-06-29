# ical-filter-proxy

A small Go service that proxies remote iCalendar (`.ics`) feeds and filters their
events according to per-calendar rules defined in YAML.

It is a **drop-in replacement** for the Ruby
[`darkphnx/ical-filter-proxy`](https://github.com/darkphnx/ical-filter-proxy):
same HTTP surface, same `config.yml` schema, same filter semantics — but a single
static binary in a distroless container.

> Status: work in progress. See [AGENTS.md](AGENTS.md) for contributor/agent notes.

## Quick start

```sh
cp config.example.yml config.yml   # then edit
go run ./cmd/ical-filter-proxy --config ./config.yml --addr :8000
```

Request a filtered calendar:

```sh
curl 'http://localhost:8000/my_calendar_name?key=myapikey'
```

### Docker

```sh
docker run -p 8000:8000 -v "$PWD/config.yml:/app/config.yml:ro" \
  ghcr.io/tamcore/ical-filter-proxy:latest
```

Or with Compose (see `docker-compose.yaml`):

```sh
docker compose up
```

For a local source build, use `make snapshot` (goreleaser) — the production
image ships a pre-built binary and has no compile stage.

### Kubernetes

A Helm chart lives in `charts/ical-filter-proxy` and is published as an OCI
artifact on each release. Put your calendars under `config:` in a values file
and install:

```sh
helm install ical-filter-proxy \
  oci://ghcr.io/tamcore/charts/ical-filter-proxy \
  -f my-values.yaml
```

## Configuration

The flags `--config` (default `/app/config.yml`) and `--addr` (default `:8000`)
can also be set via `ICAL_FILTER_PROXY_CONFIG` and `ICAL_FILTER_PROXY_ADDR`.


`config.yml` is a map of calendar name to calendar definition:

```yaml
my_calendar_name:
  ical_url: https://source-calendar.com/my_calendar.ics
  api_key: myapikey            # optional; if set, requests must pass ?key=
  timezone: Europe/London      # optional; default UTC. Used for time/date rules.
  rules:                       # optional; ALL rules must match (logical AND)
    - field: summary           # one of: start_time end_time start_date end_date
      operator: startswith     #         summary description blocking
      val:                     # a list is OR-ed within the rule
        - Planning
        - Daily Standup
    - field: summary
      operator: matches        # /pattern/flags regular expression
      val:
        - /Team A/i
    - field: start_time        # HH:MM in the calendar timezone
      operator: not-equals     # any operator may be negated with the not- prefix
      val: "09:00"
  alarms:                      # optional alarm manipulation
    clear_existing: true       # drop the feed's own VALARMs
    triggers:                  # add new alarms (ISO8601 duration or natural)
      - -P1DT0H0M0S
      - 2 days
      - 10 minutes
```

### Operators

`equals`, `startswith`, `includes`, `matches` — each can be negated by prefixing
`not-` (e.g. `not-equals`, `not-matches`).

### Fields

- `summary`, `description` — string match.
- `start_time`, `end_time` — `HH:MM` (24h) in the calendar timezone.
- `start_date`, `end_date` — `YYYY-MM-DD` in the calendar timezone.
- `blocking` — boolean; `true` when the event is `TRANSP:OPAQUE` (or has no
  `TRANSP`), `false` when `TRANSP:TRANSPARENT`.

### Environment substitution

Any `${ICAL_FILTER_PROXY_NAME}` placeholder in the config is replaced with the
value of the corresponding environment variable (empty string if unset) before
the YAML is parsed.

## HTTP behavior

| Request                              | Response                                   |
| ------------------------------------ | ------------------------------------------ |
| `GET /`                              | `200` welcome text                         |
| `GET /<cal>?key=<key>`               | `200` `text/calendar` filtered feed        |
| missing/wrong `key` (when required)  | `403`                                      |
| unknown calendar                     | `404`                                      |
| upstream feed unreachable/invalid    | `5xx`                                      |

## Migrating from the Ruby version

The config schema and URL format are identical, so the existing `config.yml`
works unchanged. Swap the container image and keep the same ConfigMap mount at
`/app/config.yml`; the service listens on `:8000` by default.

## License

[MIT](LICENSE)

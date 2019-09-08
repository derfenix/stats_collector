# Statistic collector

Statistics collected via Websocket connection.

Collector stores incoming data within in-memory cache and flushes it into database every `$SYNC_INTERVAL`
seconds (by default — each 15 seconds).

Accepts only one message format:
```json
{"id": 123456, "label":  "some_label"}
```
Each message will increase corresponding counter by 1.

## Usage

### Start service manually

Build the service binary:

```bash
go build -o collector ./cmd/collector/collector.go
```

Server can be configured via command line options or via environment variables (specified in 
squared braces `[]`):

```
collector [OPTIONS]

Application Options:
      --dbuser=     Database user [$DB_USER]
      --dbpass=     Database password [$DB_PASS]
      --dbname=     Database name (default: statistics) [$DB_NAME]
      --dbhost=     Database host (default: db) [$DB_HOST]
      --dbport=     Database port (default: 3306) [$DB_PORT]
      --wshost=     WS server host (default: 0.0.0.0) [$WS_HOST]
      --wsport=     WS server port (default: 8881) [$WS_PORT]
      --wsendpoint= WS endpoint (default: /ws/) [$WS_ENDPOINT]
  -i, --interval=   Flush to database each seconds (sync interval) (default: 15) [$SYNC_INTERVAL]

Help Options:
  -h, --help        Show this help message
```

Example:

```bash
collector -i 60 --wsendpoint=/ws/stats/ --dbname=foobar
```

### Start service via Docker-compose

Start the service with database
```bash
cd stats_collector
docker-compose up -d
```

Websocket server will be available via `ws://127.0.0.1/ws/`. 

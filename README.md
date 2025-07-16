The server side of the digital signage software

### 1. Project Layout
```
.
├── Dockerfile
├── Makefile
├── README.md
├── docker-compose.yml
├── go.mod
├── go.sum
│   
├── cmd
│   └── server
│       └── main.go
├── internal
│   ├── api
│   │   ├── admin
│   │   │   ├── auth.go
│   │   │   ├── content.go
│   │   │   ├── schedules.go
│   │   │   └── screens.go
│   │   └── tv
│   │       ├── socket/
│   │       │   ├── mqtt.go
│   │       │   └── README.md
│   │       ├── endpoints/
│   │       └── packets/
│   ├── auth
│   │   └── auth.go
│   ├── config
│   │   └── config.go
│   ├── db
│   │   ├── db.go
│   │   └── store.go
│   └── model
│       ├── screen.go
│       └── user.go
└── migrations
    ├── _init.down.sql
    └── _init.up.sql
```

`cmd/server/main.go` is the entry point of the server implementation. 

All software implementation is contained in the `internal` directory.

The API endpoints are in the `api/` directory, with `api/admin/` corresponding to accounts/webapp endpoints and `api/tv/` corresponding to the TV client side endpoints.

The `model/` directory defines the global `struct` definitions used throughout the application. Refer to the implementations in these files to add definitions in the future. 
This is where things like `schedules`, `canvasses`, `groups`, etc. would be defined when they are implemented.

The `db/` directory is for postgres API implemention, it exposes an internal API for the rest of the application to use. Each database interaction is facilitated by the `db` package. 
`store.go` defines what a database store implementation must follow, so whenever a function is added to `db.go`, its declaration must be in the `Store` interface.

The `auth/` directory is for auth helpers at the moment, I might restructure it into a `utils` or `helpers` directory later if more instances pop up.

The `config/` directory is for runtime environment variable implementation and runtime configuration.

### 2. Prerequisites

- **Go 1.24+**  
- **PostgreSQL 13+** (client & server)  
- **MQTT Broker** (for TV device communication)
- **Docker** (optional, for containerized testing)  
- **golang-migrate CLI** (only if you want to run migrations manually)  
  ```bash
  go install -tags 'postgres file' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
  export PATH="$PATH:$(go env GOPATH)/bin"

### 3. MQTT Communication

The system uses MQTT for real-time communication with TV devices. This replaces the previous WebSocket implementation.

#### Setup MQTT Broker

You'll need an MQTT broker running. Popular options include:
- **Mosquitto** (open source): `docker run -p 1883:1883 eclipse-mosquitto:latest`
- **Eclipse HiveMQ**
- **AWS IoT Core**
- **Azure IoT Hub**

#### Configuration

Set the `MQTT_BROKER_URL` environment variable (default: `tcp://localhost:1883`):

```bash
export MQTT_BROKER_URL="tcp://localhost:1883"
```

#### TV Device Connection

TV devices connect via the `/api/tv/socket` endpoint with a `device_id` parameter. The system automatically:
- Creates an MQTT client for each device
- Subscribes to device-specific topics (`tv/{device_id}/commands`)
- Handles automatic reconnection

For detailed MQTT implementation information, see `internal/http/api/tv/socket/README.md`.

### 4. Setup
Set up PostgreSQL db + user
```bash
sudo -iu postgres initdb --locale en_US.UTF-8 -D /var/lib/postgres/data
sudo systemctl enable --now postgresql

sudo -u postgres createuser --pwprompt medusa_user
sudo -u postgres createdb medusa_dev --owner=medusa_user
```

Fill out the `.env-template` with the necessary information then run `source ./.env-template`

Run database migrations
```bash
migrate -path ./migrations -database "$DATABASE_URL" up
```

### 5. Build and run

```bash
# From project root
go mod tidy
go build -o medusa cmd/server/main.go
./medusa
```

OR with Docker 

```bash
# Build the image
docker build -t medusa:local .

# Run container
docker run -d --name medusa \
  --env-file ./env-template\
  -p 8080:8080 \
  medusa:local
```

Then you can curl localhost on the port it's listening on to test, e.g.
```bash
curl -i https://localhost:8080/api/admin/screens
```

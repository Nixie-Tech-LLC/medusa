The server side of the digital signage software

### 1. Prerequisites

- **Go 1.24+**  
- **PostgreSQL 13+** (client & server)  
- **Docker** (optional, for containerized testing)  
- **golang-migrate CLI** (only if you want to run migrations manually)  
  ```bash
  go install -tags 'postgres file' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
  export PATH="$PATH:$(go env GOPATH)/bin"

### 2. Setup
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

### 3. Build and run

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

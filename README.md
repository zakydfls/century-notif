# Notification Gateway

Notif Gateway handles SMS, WhatsApp, and Email notifications via REST enqueue, Postgres, RabbitMQ, and worker consumers.

It can send without template. For plain text notification, send `body` directly:

```json
{
  "channel": "sms",
  "recipient": "+628111111111",
  "body": "Your OTP is 123456"
}
```
Set caller service identity with `X-Century-Service`; missing or blank values are stored as `unknown`.

Common commands:

```bash
go run ./cmd/migration
go run ./cmd/http
go run ./cmd/mq
```

Docker Compose deployment runs API and worker as separate services:

```bash
docker compose --env-file .env.deploy up -d api worker
docker compose logs -f api
docker compose logs -f worker
```

On the VPS, keep the full application env at `~/notif-gateway/.env`. The GitHub Actions deploy workflow pushes the image to GHCR, copies `docker-compose.yml` to `~/notif-gateway`, pulls the tagged image, runs the `migrate` service (`/app/notif-migration`), then restarts `api` and `worker`.

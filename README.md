# Notification Gateway

Notif Gateway handles SMS, WhatsApp, and Email notifications via REST enqueue, Postgres, RabbitMQ, and worker consumers.

Whatsockt is the default WhatsApp provider. Configure it with:

```env
PROVIDER_DEFAULT_WHATSAPP=whatsockt
WHATSOCKT_BASE_URL=https://whatsockt.example.com
WHATSOCKT_API_KEY=wag_xxxxxxxxx
WHATSOCKT_DEVICE_ID=00000000-0000-0000-0000-000000000000
WHATSOCKT_WEBHOOK_SECRET=your-webhook-signing-secret
```

Point the Whatsockt `message.outbound.*` webhook subscription to
`https://notifications.example.com/api/v1/webhooks/whatsockt/whatsapp`.

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
docker compose --profile migration run --rm migrate
docker compose up -d
docker compose logs -f api
docker compose logs -f worker
```

On the VPS, keep the full application env at `~/notif-gateway/.env`. The GitHub Actions deploy workflow pushes the image to GHCR, copies `docker-compose.yml` to `~/notif-gateway`, updates `NOTIF_IMAGE` inside `.env`, pulls the tagged image, runs the `migrate` service (`/app/notif-migration`), then restarts `api` and `worker`.

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
Common commands:

```bash
go run ./cmd/http
go run ./cmd/mq
```

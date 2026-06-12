package waha

import (
	"net/http"
	"testing"
)

func TestNormalizeChatID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		recipient string
		want      string
	}{
		{name: "e164 with plus", recipient: "+6285959186585", want: "6285959186585@c.us"},
		{name: "e164 without plus", recipient: "6285959186585", want: "6285959186585@c.us"},
		{name: "full chat id with plus", recipient: "+6285959186585@c.us", want: "6285959186585@c.us"},
		{name: "full chat id without plus", recipient: "6285959186585@c.us", want: "6285959186585@c.us"},
		{name: "group id unchanged", recipient: "12132132130@g.us", want: "12132132130@g.us"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := normalizeChatID(tt.recipient); got != tt.want {
				t.Fatalf("normalizeChatID(%q) = %q, want %q", tt.recipient, got, tt.want)
			}
		})
	}
}

func TestExtractWAHAErrorDetail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "nest message string",
			body: `{"statusCode":500,"message":"Session 'default' is not connected","error":"Internal Server Error"}`,
			want: "Session 'default' is not connected",
		},
		{
			name: "nest message array",
			body: `{"statusCode":422,"message":["chatId must be a valid chat id"],"error":"Unprocessable Entity"}`,
			want: "chatId must be a valid chat id",
		},
		{
			name: "plain text body",
			body: "upstream timeout",
			want: "upstream timeout",
		},
		{
			name: "waha exception with minified puppeteer error",
			body: `{"statusCode":500,"timestamp":"2026-06-12T08:40:24.712Z","exception":{"message":"t","name":"t","stack":"t: t\n    at #evaluate (/app/node_modules/puppeteer-core/...)"}}`,
			want: "whatsapp engine error at 2026-06-12T08:40:24.712Z (verify WAHA session is connected)",
		},
		{
			name: "waha exception with readable message",
			body: `{"statusCode":500,"timestamp":"2026-06-12T08:40:24.712Z","exception":{"message":"Session status is STOPPED","name":"UnprocessableEntityException"}}`,
			want: "UnprocessableEntityException: Session status is STOPPED",
		},
		{
			name: "empty body",
			body: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := extractWAHAErrorDetail([]byte(tt.body)); got != tt.want {
				t.Fatalf("extractWAHAErrorDetail() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatSendTextError(t *testing.T) {
	t.Parallel()

	got := formatSendTextError(http.StatusInternalServerError, "default", "6285959186585@c.us", []byte(`{"message":"engine not ready"}`))
	want := "waha sendText failed (http 500, session=default, chatId=6285959186585@c.us): engine not ready"
	if got != want {
		t.Fatalf("formatSendTextError() = %q, want %q", got, want)
	}
}

package godaddy

import (
	"errors"
	"strings"
	"testing"

	mail "github.com/wneessen/go-mail"

	"notif-gateway/internal/core/command"
	"notif-gateway/internal/core/enum"
	"notif-gateway/internal/infrastructure"
	apperror "notif-gateway/internal/value_object/handling/error"
	types "notif-gateway/internal/value_object/util/type"
)

func assertBadRequest(t *testing.T, err error, wantKey, wantMessage string) {
	t.Helper()
	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("err = %T(%v), want *apperror.Error", err, err)
	}
	if appErr.HTTPStatus != 400 {
		t.Fatalf("status = %d, want 400", appErr.HTTPStatus)
	}
	if appErr.GetKey() != wantKey {
		t.Fatalf("key = %q, want %q", appErr.GetKey(), wantKey)
	}
	if appErr.GetMessage() != wantMessage {
		t.Fatalf("message = %q, want %q", appErr.GetMessage(), wantMessage)
	}
}

func TestBuildMessage_plainTextDefaults(t *testing.T) {
	msg, err := BuildMessageForTest(command.ProviderSend{
		Recipient: "user@example.com",
		Body:      "Hello world",
		Metadata: types.JSONMap{
			enum.EmailMetaSubject: "Test subject",
		},
	}, infrastructure.GoDaddyConfig{
		FromAddress: "sender@example.com",
		FromName:    "Sender",
	})
	if err != nil {
		t.Fatalf("buildMessage: %v", err)
	}
	from := strings.Join(msg.GetFromString(), " ")
	if !strings.Contains(from, "sender@example.com") || !strings.Contains(from, "Sender") {
		t.Fatalf("from = %q, want sender address and display name", from)
	}
	to := strings.Join(msg.GetToString(), " ")
	if !strings.Contains(to, "user@example.com") {
		t.Fatalf("to = %q", to)
	}
	subjects := msg.GetGenHeader(mail.HeaderSubject)
	if len(subjects) != 1 || subjects[0] != "Test subject" {
		t.Fatalf("subject = %v", subjects)
	}
}

func TestBuildMessage_htmlContentType(t *testing.T) {
	msg, err := BuildMessageForTest(command.ProviderSend{
		Recipient: "user@example.com",
		Body:      "<p>Hi</p>",
		Metadata: types.JSONMap{
			enum.EmailMetaSubject:     "HTML",
			enum.EmailMetaContentType: enum.EmailContentTypeHTML,
			enum.EmailMetaFromAddress: "custom@example.com",
		},
	}, infrastructure.GoDaddyConfig{
		FromAddress: "fallback@example.com",
	})
	if err != nil {
		t.Fatalf("buildMessage: %v", err)
	}
	from := strings.Join(msg.GetFromString(), " ")
	if !strings.Contains(from, "custom@example.com") {
		t.Fatalf("from = %q, want metadata override", from)
	}
}

func TestBuildMessage_missingSubject(t *testing.T) {
	_, err := BuildMessageForTest(command.ProviderSend{
		Recipient: "user@example.com",
		Body:      "hi",
		Metadata:  types.JSONMap{},
	}, infrastructure.GoDaddyConfig{FromAddress: "a@b.com"})
	assertBadRequest(t, err, apperror.KeyEmailSubjectRequired, "email subject is required in metadata")
}

func TestBuildMessage_ccBcc(t *testing.T) {
	msg, err := BuildMessageForTest(command.ProviderSend{
		Recipient: "user@example.com",
		Body:      "hi",
		Metadata: types.JSONMap{
			enum.EmailMetaSubject: "Sub",
			enum.EmailMetaCC:      []any{"cc@example.com"},
			enum.EmailMetaBCC:     []string{"bcc@example.com"},
		},
	}, infrastructure.GoDaddyConfig{FromAddress: "from@example.com"})
	if err != nil {
		t.Fatalf("buildMessage: %v", err)
	}
	cc := strings.Join(msg.GetCcString(), " ")
	if !strings.Contains(cc, "cc@example.com") {
		t.Fatalf("cc = %q", cc)
	}
	bcc := strings.Join(msg.GetBccString(), " ")
	if !strings.Contains(bcc, "bcc@example.com") {
		t.Fatalf("bcc = %q", bcc)
	}
}

func TestBuildMessage_replyTo(t *testing.T) {
	msg, err := BuildMessageForTest(command.ProviderSend{
		Recipient: "user@example.com",
		Body:      "hi",
		Metadata: types.JSONMap{
			enum.EmailMetaSubject: "Sub",
			enum.EmailMetaReplyTo: "reply@example.com",
		},
	}, infrastructure.GoDaddyConfig{FromAddress: "from@example.com"})
	if err != nil {
		t.Fatalf("buildMessage: %v", err)
	}
	replyTo := msg.GetReplyTo()
	if len(replyTo) == 0 || replyTo[0].Address != "reply@example.com" {
		t.Fatalf("reply-to = %v, want reply@example.com", replyTo)
	}
}

func TestBuildMessage_nilMetadata(t *testing.T) {
	_, err := BuildMessageForTest(command.ProviderSend{
		Recipient: "user@example.com",
		Body:      "hi",
		Metadata:  nil,
	}, infrastructure.GoDaddyConfig{FromAddress: "from@example.com"})
	assertBadRequest(t, err, apperror.KeyEmailSubjectRequired, "email subject is required in metadata")
}

func TestBuildMessage_invalidContentType(t *testing.T) {
	_, err := BuildMessageForTest(command.ProviderSend{
		Recipient: "user@example.com",
		Body:      "hi",
		Metadata: types.JSONMap{
			enum.EmailMetaSubject:     "Sub",
			enum.EmailMetaContentType: "application/json",
		},
	}, infrastructure.GoDaddyConfig{FromAddress: "from@example.com"})
	assertBadRequest(t, err, apperror.KeyEmailContentTypeInvalid, "metadata.content_type must be text/plain or text/html")
}

func TestBuildMessage_missingFrom(t *testing.T) {
	_, err := BuildMessageForTest(command.ProviderSend{
		Recipient: "user@example.com",
		Body:      "hi",
		Metadata: types.JSONMap{
			enum.EmailMetaSubject: "Sub",
		},
	}, infrastructure.GoDaddyConfig{})
	assertBadRequest(t, err, apperror.KeyEmailFromRequired, "from address is required (metadata.from_address or GODADDY_FROM_ADDRESS)")
}

func TestBuildMessage_multipleCcBcc(t *testing.T) {
	msg, err := BuildMessageForTest(command.ProviderSend{
		Recipient: "user@example.com",
		Body:      "hi",
		Metadata: types.JSONMap{
			enum.EmailMetaSubject: "Sub",
			enum.EmailMetaCC:      []any{"cc1@example.com", "cc2@example.com"},
			enum.EmailMetaBCC:     []string{"bcc1@example.com", "bcc2@example.com"},
		},
	}, infrastructure.GoDaddyConfig{FromAddress: "from@example.com"})
	if err != nil {
		t.Fatalf("buildMessage: %v", err)
	}
	ccAll := strings.Join(msg.GetCcString(), " ")
	for _, want := range []string{"cc1@example.com", "cc2@example.com"} {
		if !strings.Contains(ccAll, want) {
			t.Fatalf("cc = %q, missing %s", ccAll, want)
		}
	}
	bccAll := strings.Join(msg.GetBccString(), " ")
	for _, want := range []string{"bcc1@example.com", "bcc2@example.com"} {
		if !strings.Contains(bccAll, want) {
			t.Fatalf("bcc = %q, missing %s", bccAll, want)
		}
	}
}

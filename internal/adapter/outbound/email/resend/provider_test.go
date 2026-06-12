package resend

import (
	"context"
	"errors"
	"strings"
	"testing"

	resendgo "github.com/resend/resend-go/v3"

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

func TestBuildRequest_plainTextDefaults(t *testing.T) {
	request, err := BuildRequestForTest(command.ProviderSend{
		Recipient: "user@example.com",
		Body:      "Hello world",
		Metadata: types.JSONMap{
			enum.EmailMetaSubject: "Test subject",
		},
	}, infrastructure.ResendConfig{
		FromAddress: "sender@example.com",
		FromName:    "Sender",
	})
	if err != nil {
		t.Fatalf("buildRequest: %v", err)
	}
	if !strings.Contains(request.From, "sender@example.com") || !strings.Contains(request.From, "Sender") {
		t.Fatalf("from = %q, want sender address and display name", request.From)
	}
	if len(request.To) != 1 || request.To[0] != "user@example.com" {
		t.Fatalf("to = %v", request.To)
	}
	if request.Subject != "Test subject" {
		t.Fatalf("subject = %q", request.Subject)
	}
	if request.Text != "Hello world" {
		t.Fatalf("text = %q", request.Text)
	}
	if request.Html != "" {
		t.Fatalf("html = %q, want empty", request.Html)
	}
}

func TestBuildRequest_htmlContentType(t *testing.T) {
	request, err := BuildRequestForTest(command.ProviderSend{
		Recipient: "user@example.com",
		Body:      "<p>Hi</p>",
		Metadata: types.JSONMap{
			enum.EmailMetaSubject:     "HTML",
			enum.EmailMetaContentType: enum.EmailContentTypeHTML,
			enum.EmailMetaFromAddress: "custom@example.com",
		},
	}, infrastructure.ResendConfig{
		FromAddress: "fallback@example.com",
	})
	if err != nil {
		t.Fatalf("buildRequest: %v", err)
	}
	if request.From != "custom@example.com" {
		t.Fatalf("from = %q, want metadata override", request.From)
	}
	if request.Html != "<p>Hi</p>" {
		t.Fatalf("html = %q", request.Html)
	}
	if request.Text != "" {
		t.Fatalf("text = %q, want empty", request.Text)
	}
}

func TestBuildRequest_providerTemplate(t *testing.T) {
	request, err := BuildRequestForTest(command.ProviderSend{
		Recipient:            "user@example.com",
		ProviderTemplateID:   "welcome",
		ProviderTemplateVars: types.JSONMap{"name": "Zak"},
		Metadata: types.JSONMap{
			enum.EmailMetaSubject: "Welcome",
		},
	}, infrastructure.ResendConfig{FromAddress: "from@example.com"})
	if err != nil {
		t.Fatalf("buildRequest: %v", err)
	}
	if request.Template == nil {
		t.Fatal("template = nil, want Resend template")
	}
	if request.Template.Id != "welcome" {
		t.Fatalf("template id = %q", request.Template.Id)
	}
	if request.Template.Variables["name"] != "Zak" {
		t.Fatalf("template variables = %v", request.Template.Variables)
	}
	if request.Text != "" || request.Html != "" {
		t.Fatalf("text/html = %q/%q, want empty for template send", request.Text, request.Html)
	}
}

func TestBuildRequest_ccBccReplyTo(t *testing.T) {
	request, err := BuildRequestForTest(command.ProviderSend{
		Recipient: "user@example.com",
		Body:      "hi",
		Metadata: types.JSONMap{
			enum.EmailMetaSubject: "Sub",
			enum.EmailMetaCC:      []any{"cc@example.com"},
			enum.EmailMetaBCC:     []string{"bcc@example.com"},
			enum.EmailMetaReplyTo: "reply@example.com",
		},
	}, infrastructure.ResendConfig{FromAddress: "from@example.com"})
	if err != nil {
		t.Fatalf("buildRequest: %v", err)
	}
	if len(request.Cc) != 1 || request.Cc[0] != "cc@example.com" {
		t.Fatalf("cc = %v", request.Cc)
	}
	if len(request.Bcc) != 1 || request.Bcc[0] != "bcc@example.com" {
		t.Fatalf("bcc = %v", request.Bcc)
	}
	if request.ReplyTo != "reply@example.com" {
		t.Fatalf("reply-to = %q", request.ReplyTo)
	}
}

func TestBuildRequest_missingSubject(t *testing.T) {
	_, err := BuildRequestForTest(command.ProviderSend{
		Recipient: "user@example.com",
		Body:      "hi",
		Metadata:  types.JSONMap{},
	}, infrastructure.ResendConfig{FromAddress: "from@example.com"})
	assertBadRequest(t, err, apperror.KeyEmailSubjectRequired, "email subject is required in metadata")
}

func TestBuildRequest_missingFrom(t *testing.T) {
	_, err := BuildRequestForTest(command.ProviderSend{
		Recipient: "user@example.com",
		Body:      "hi",
		Metadata: types.JSONMap{
			enum.EmailMetaSubject: "Sub",
		},
	}, infrastructure.ResendConfig{})
	assertBadRequest(t, err, apperror.KeyEmailFromRequired, "from address is required (metadata.from_address or RESEND_FROM_ADDRESS)")
}

func TestBuildRequest_invalidContentType(t *testing.T) {
	_, err := BuildRequestForTest(command.ProviderSend{
		Recipient: "user@example.com",
		Body:      "hi",
		Metadata: types.JSONMap{
			enum.EmailMetaSubject:     "Sub",
			enum.EmailMetaContentType: "application/json",
		},
	}, infrastructure.ResendConfig{FromAddress: "from@example.com"})
	assertBadRequest(t, err, apperror.KeyEmailContentTypeInvalid, "metadata.content_type must be text/plain or text/html")
}

func TestBuildRequest_invalidRecipient(t *testing.T) {
	_, err := BuildRequestForTest(command.ProviderSend{
		Recipient: "not-an-email",
		Body:      "hi",
		Metadata: types.JSONMap{
			enum.EmailMetaSubject: "Sub",
		},
	}, infrastructure.ResendConfig{FromAddress: "from@example.com"})
	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("err = %T(%v), want *apperror.Error", err, err)
	}
	if appErr.GetKey() != apperror.KeyEmailInvalidRecipient {
		t.Fatalf("key = %q, want %q", appErr.GetKey(), apperror.KeyEmailInvalidRecipient)
	}
}

type fakeEmailSender struct {
	request  *resendgo.SendEmailRequest
	response *resendgo.SendEmailResponse
	err      error
}

func (f *fakeEmailSender) SendWithContext(_ context.Context, params *resendgo.SendEmailRequest) (*resendgo.SendEmailResponse, error) {
	f.request = params
	return f.response, f.err
}

func TestSend_success(t *testing.T) {
	sender := &fakeEmailSender{response: &resendgo.SendEmailResponse{Id: "email_123"}}
	provider := &Provider{
		emails: sender,
		config: infrastructure.ResendConfig{FromAddress: "from@example.com"},
	}

	result, err := provider.Send(context.Background(), command.ProviderSend{
		Recipient: "user@example.com",
		Body:      "hi",
		Metadata: types.JSONMap{
			enum.EmailMetaSubject: "Sub",
		},
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if result.MessageID != "email_123" {
		t.Fatalf("message id = %q", result.MessageID)
	}
	if sender.request == nil || sender.request.Subject != "Sub" {
		t.Fatalf("request = %#v", sender.request)
	}
	if result.Raw["id"] != "email_123" {
		t.Fatalf("raw = %v", result.Raw)
	}
}

func TestSend_wrapsProviderError(t *testing.T) {
	provider := &Provider{
		emails: &fakeEmailSender{err: errors.New("boom")},
		config: infrastructure.ResendConfig{FromAddress: "from@example.com"},
	}

	_, err := provider.Send(context.Background(), command.ProviderSend{
		Recipient: "user@example.com",
		Body:      "hi",
		Metadata: types.JSONMap{
			enum.EmailMetaSubject: "Sub",
		},
	})
	assertBadRequest(t, err, apperror.KeyProviderSendFailed, "resend email send failed")
}

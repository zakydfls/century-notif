package rest

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"notif-gateway/internal/core/command"
	"notif-gateway/internal/core/entity"
	"notif-gateway/internal/core/service"
	"notif-gateway/internal/pkg/validator"
	"notif-gateway/internal/value_object/dto/pagination"
	response "notif-gateway/internal/value_object/dto/response"
	apperror "notif-gateway/internal/value_object/handling/error"
)

// NotificationREST is the inbound HTTP port for notification APIs.
type NotificationREST interface {
	RouteModule

	Enqueue(w http.ResponseWriter, r *http.Request) error
	EnqueueBatch(w http.ResponseWriter, r *http.Request) error
	List(w http.ResponseWriter, r *http.Request) error
	FindByID(w http.ResponseWriter, r *http.Request) error
}

func NewNotificationREST(notifications service.Notification) NotificationREST {
	return &notificationREST{notifications: notifications}
}

type notificationREST struct {
	notifications service.Notification
}

func (n *notificationREST) GetRoutePath() string {
	return "/notifications"
}

func (n *notificationREST) Start(_ context.Context) chi.Router {
	r := chi.NewRouter()
	n.Mount(r)
	return r
}

func (n *notificationREST) Mount(r chi.Router) {
	r.Post("/", RootHandler(n.Enqueue))
	r.Post("/batch", RootHandler(n.EnqueueBatch))
	r.Get("/", RootHandler(n.List))
	r.Get("/{id}", RootHandler(n.FindByID))
}

func (n *notificationREST) Enqueue(w http.ResponseWriter, r *http.Request) error {
	var request command.EnqueueSingle
	if err := validator.Bind(r, &request); err != nil {
		return err
	}
	input := request.ToEnqueue()
	input.ServiceName = serviceNameFromContext(r.Context())
	result, err := n.notifications.Enqueue(r.Context(), input)
	if err != nil {
		return err
	}
	response.SuccessData(w, "notification enqueued", http.StatusAccepted, result)
	return nil
}

func (n *notificationREST) EnqueueBatch(w http.ResponseWriter, r *http.Request) error {
	var request command.Enqueue
	if err := validator.Bind(r, &request); err != nil {
		return err
	}
	request.ServiceName = serviceNameFromContext(r.Context())
	result, err := n.notifications.EnqueueBatch(r.Context(), request)
	if err != nil {
		return err
	}
	response.SuccessData(w, "notifications enqueued", http.StatusAccepted, result)
	return nil
}

func (n *notificationREST) List(w http.ResponseWriter, r *http.Request) error {
	var query command.ListNotifications
	if err := validator.BindQuery(r, &query); err != nil {
		return err
	}

	pg := pagination.NewPagination(r.URL.Query())
	if query.Channel != "" {
		pg.QueryAdd("channel", query.Channel)
	}
	if query.ServiceName != "" {
		pg.QueryAdd("service_name", query.ServiceName)
	}
	if query.Status != "" {
		pg.QueryAdd("status", query.Status)
	}
	if query.Provider != "" {
		pg.QueryAdd("provider", query.Provider)
	}
	if query.Recipient != "" {
		pg.QueryAdd("recipient", query.Recipient)
	}
	if query.UserID != "" {
		pg.QueryAdd("user_id", query.UserID)
	}

	if err := n.notifications.FindAll(r.Context(), pg); err != nil {
		return err
	}
	response.SuccessPaginate(w, "", http.StatusOK, pg.Rows, response.MetaFromPagination(pg))
	return nil
}

func (n *notificationREST) FindByID(w http.ResponseWriter, r *http.Request) error {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return apperror.ErrBadRequest(err, "id must be a valid UUID", apperror.KeyInvalidUUID)
	}
	notification := new(entity.Notification)
	notification.ID = id
	if err := n.notifications.Find(r.Context(), notification); err != nil {
		return err
	}
	response.SuccessData(w, "", http.StatusOK, notification)
	return nil
}

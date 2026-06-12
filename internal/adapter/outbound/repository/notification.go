package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"notif-gateway/internal/core/entity"
	"notif-gateway/internal/value_object/dto/pagination"
)

type Notification interface {
	BaseRepo[Notification]

	Create(ctx context.Context, e *entity.Notification) error
	CreateMany(ctx context.Context, es []*entity.Notification) error
	Find(ctx context.Context, e *entity.Notification) error
	Save(ctx context.Context, e *entity.Notification) error
	FindAll(ctx context.Context, pg *pagination.Pagination) error
}

func NewNotification(db *gorm.DB) Notification {
	fac := func(base *baseRepo[Notification]) Notification {
		return &notification{baseRepo: base}
	}
	return newRepository(fac, db)
}

type notification struct {
	*baseRepo[Notification]
}

func (r *notification) Create(ctx context.Context, e *entity.Notification) error {
	return r.dbCtx(ctx).Create(e).Error
}

func (r *notification) CreateMany(ctx context.Context, es []*entity.Notification) error {
	if len(es) == 0 {
		return nil
	}
	return r.dbCtx(ctx).Create(&es).Error
}

func (r *notification) Find(ctx context.Context, e *entity.Notification) error {
	if e == nil {
		return fmt.Errorf("notification is required")
	}

	query := r.dbCtx(ctx).Model(&entity.Notification{})
	switch {
	case e.ID != uuid.Nil:
		query = query.Where("id = ?", e.ID)
	case e.Provider != "" && e.ProviderMessageID != "":
		query = query.Where("provider = ? AND provider_message_id = ?", e.Provider, e.ProviderMessageID)
	default:
		return fmt.Errorf("notification lookup criteria is required")
	}

	err := query.First(e).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}

func (r *notification) Save(ctx context.Context, e *entity.Notification) error {
	return r.dbCtx(ctx).Save(e).Error
}

func (r *notification) FindAll(ctx context.Context, pg *pagination.Pagination) error {
	if pg == nil {
		return fmt.Errorf("pagination is required")
	}

	query := r.dbCtx(ctx).Model(&entity.Notification{})
	if v := pg.QueryGet("channel"); v != "" {
		query = query.Where("channel = ?", v)
	}
	if v := pg.QueryGet("provider"); v != "" {
		query = query.Where("provider = ?", v)
	}
	if v := pg.QueryGet("status"); v != "" {
		query = query.Where("status = ?", v)
	}
	if v := pg.QueryGet("recipient"); v != "" {
		query = query.Where("recipient = ?", v)
	}
	if v := pg.QueryGet("user_id"); v != "" {
		query = query.Where("user_id = ?", v)
	}

	var items []entity.Notification
	if err := query.
		Scopes(pagination.Paginate(&entity.Notification{}, pg, query)).
		Order("created_at DESC").
		Find(&items).Error; err != nil {
		return err
	}
	pg.Rows = items
	return nil
}

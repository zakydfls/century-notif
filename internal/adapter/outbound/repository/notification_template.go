package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"notif-gateway/internal/core/entity"
)

type Template interface {
	BaseRepo[Template]

	Find(ctx context.Context, e *entity.NotificationTemplate) error
}

func NewTemplate(db *gorm.DB) Template {
	fac := func(base *baseRepo[Template]) Template {
		return &template{baseRepo: base}
	}
	return newRepository(fac, db)
}

type template struct {
	*baseRepo[Template]
}

func (r *template) Find(ctx context.Context, e *entity.NotificationTemplate) error {
	if e == nil {
		return fmt.Errorf("notification template is required")
	}
	if e.Key == "" || e.Channel == "" {
		return fmt.Errorf("notification template lookup criteria is required")
	}

	query := r.dbCtx(ctx).
		Where("key = ? AND channel = ? AND active = ?", e.Key, e.Channel, true)
	if e.Provider != "" {
		query = query.Where("provider = ?", e.Provider)
	}
	err := query.First(e).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}

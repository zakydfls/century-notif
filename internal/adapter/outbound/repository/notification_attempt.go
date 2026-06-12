package repository

import (
	"context"

	"gorm.io/gorm"

	"notif-gateway/internal/core/entity"
)

type Attempt interface {
	BaseRepo[Attempt]

	Create(ctx context.Context, e *entity.NotificationAttempt) error
}

func NewAttempt(db *gorm.DB) Attempt {
	fac := func(base *baseRepo[Attempt]) Attempt {
		return &attempt{baseRepo: base}
	}
	return newRepository(fac, db)
}

type attempt struct {
	*baseRepo[Attempt]
}

func (r *attempt) Create(ctx context.Context, e *entity.NotificationAttempt) error {
	return r.dbCtx(ctx).Create(e).Error
}

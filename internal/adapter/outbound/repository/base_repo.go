package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("record not found")

type RepoFactory[T comparable] func(base *baseRepo[T]) T

type BaseRepoCustom interface {
	Custom(ctx context.Context, fn func(db *gorm.DB) *gorm.DB) error
	CustomError(ctx context.Context, fn func(db *gorm.DB) error) error
}

type BaseRepo[T comparable] interface {
	WithTrx(db *gorm.DB) T
	BaseRepoCustom
}

func newRepository[T comparable](fac RepoFactory[T], db *gorm.DB) T {
	b := newBaseRepo[T](db)
	t := fac(b)
	b.fac = fac
	return t
}

func newBaseRepo[T comparable](db *gorm.DB) *baseRepo[T] {
	return &baseRepo[T]{db: db}
}

type baseRepo[T comparable] struct {
	db  *gorm.DB
	fac RepoFactory[T]
}

func (b *baseRepo[T]) WithTrx(db *gorm.DB) T {
	return b.fac(newBaseRepo[T](db))
}

func (b *baseRepo[T]) Custom(ctx context.Context, fn func(db *gorm.DB) *gorm.DB) error {
	return fn(b.db.WithContext(ctx)).Error
}

func (b *baseRepo[T]) CustomError(ctx context.Context, fn func(db *gorm.DB) error) error {
	return fn(b.db.WithContext(ctx))
}

func (b *baseRepo[T]) dbCtx(ctx context.Context) *gorm.DB {
	return b.db.WithContext(ctx)
}

func (b *baseRepo[T]) softDelete(ctx context.Context, model any, deletedBy uint) error {
	return b.db.WithContext(ctx).
		Model(model).
		Updates(map[string]any{
			"deleted_by": deletedBy,
			"deleted_at": gorm.Expr("NOW()"),
		}).Error
}

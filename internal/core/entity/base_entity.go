package entity

import (
	"time"

	"github.com/google/uuid"

	"gorm.io/gorm"
)

type ModelMemo interface {
	HasMemo(key string) bool
	GetMemo(key string) any
	SetMemo(key string, value any)
	RemoveMemo(key string)
}

type modelHasOriginal struct {
	memo
}

func (m *modelHasOriginal) HasOriginal() bool {
	if !m.HasMemo("original") {
		return false
	}

	return m.GetMemo("original") != nil
}

func (m *modelHasOriginal) GetOriginal() any {
	if !m.HasOriginal() {
		return nil
	}
	return m.GetMemo("original")
}

func (m *modelHasOriginal) SetOriginal(original any) {
	if original == nil {
		m.RemoveMemo("original")
		return
	}
	m.SetMemo("original", original)
}

type BaseUintModel struct {
	modelHasOriginal

	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type DeleteableUUIDModel struct {
	modelHasOriginal

	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (m *DeleteableUUIDModel) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

type SoftDeleteModel struct {
	DeleteableUUIDModel

	DeletedBy *uint          `json:"-"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

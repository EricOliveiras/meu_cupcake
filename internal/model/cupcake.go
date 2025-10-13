package model

import (
	"time"

	"gorm.io/gorm"
)

// Cupcake representa um produto a ser vendido na loja.
type Cupcake struct {
	ID          uint           `gorm:"primaryKey"`
	Nome        string         `gorm:"not null;size:100"`
	Descricao   string         `gorm:"type:text"`
	Preco       float64        `gorm:"not null"`
	ImagemURL   string         `gorm:"not null"` // Armazenaremos o caminho/URL da imagem
	Disponivel  bool           `gorm:"default:true"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"` // Para "soft delete"
}
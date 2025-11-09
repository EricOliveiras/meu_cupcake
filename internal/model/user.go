// /internal/model/usuario.go
package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	RoleCliente = "cliente"
	RoleLojista = "lojista"
)

type Usuario struct {
	ID          uint   `gorm:"primaryKey"`
	Nome        string `gorm:"not null"`
	Email       string `gorm:"unique;not null"`
	SenhaHash   string `gorm:"not null"`
	Telefone    string `gorm:"size:20"`
	CEP         string `gorm:"size:10"`
	Rua         string `gorm:"size:255"`
	Numero      string `gorm:"size:20"`
	Complemento string `gorm:"size:100"`
	Bairro      string `gorm:"size:100"`
	Cidade      string `gorm:"size:100"`
	Estado      string `gorm:"size:2"`
	Tipo        string `gorm:"default:'cliente';not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

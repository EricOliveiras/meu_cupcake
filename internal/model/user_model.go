package model

import "time"

const (
	RoleCliente = "cliente"
	RoleLojista = "lojista"
)

type Usuario struct {
	ID        uint      `gorm:"primaryKey"`
	Nome      string    `gorm:"not null"`
	Email     string    `gorm:"unique;not null"`
	SenhaHash string    `gorm:"not null"`
	Telefone  string
	Endereco  string
	Tipo      string    `gorm:"default:'cliente';not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
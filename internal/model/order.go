// /internal/model/pedido.go
package model

import (
	"time"

	"gorm.io/gorm"
)

// StatusPedido define os possíveis status de um pedido
type StatusOrder string

const (
	StatusPendente  StatusOrder = "pendente"
	StatusPago      StatusOrder = "pago"
	StatusFalhou    StatusOrder = "falhou"
	StatusEnviado   StatusOrder = "enviado"
	StatusEntregue  StatusOrder = "entregue"
	StatusCancelado StatusOrder = "cancelado"
)

// Pedido representa uma ordem de compra no sistema.
type Pedido struct {
	ID        uint        `gorm:"primaryKey"`
	UsuarioID uint        `gorm:"not null"`             // Chave estrangeira para o usuário cliente
	Usuario   Usuario     `gorm:"foreignKey:UsuarioID"` // Relacionamento com Usuario
	Status    StatusOrder `gorm:"type:varchar(20);not null;default:'pendente'"`
	Total     float64     `gorm:"not null"`
	// --- Informações do Pagamento ---
	PagamentoMPID   *int64 `gorm:"uniqueIndex"` // ID do pagamento no Mercado Pago (ponteiro para ser opcional no início)
	MetodoPagamento string // Ex: "credit_card"
	Parcelas        int
	// -------------------------------
	ExternalReference string      `gorm:"uniqueIndex"`         // Nosso ID único para enviar ao MP
	Items             []ItemOrder `gorm:"foreignKey:PedidoID"` // Um pedido tem muitos itens
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         gorm.DeletedAt `gorm:"index"`
}

// ItemOrder representa um item dentro de um Pedido.
type ItemOrder struct {
	ID            uint    `gorm:"primaryKey"`
	PedidoID      uint    `gorm:"not null"`             // Chave estrangeira para o Pedido
	CupcakeID     uint    `gorm:"not null"`             // Chave estrangeira para o Cupcake
	Cupcake       Cupcake `gorm:"foreignKey:CupcakeID"` // Relacionamento com Cupcake (para buscar dados depois)
	Quantidade    int     `gorm:"not null"`
	PrecoUnitario float64 `gorm:"not null"` // Preço no momento da compra (importante!)
	Subtotal      float64 `gorm:"not null"`
	CreatedAt     time.Time
}

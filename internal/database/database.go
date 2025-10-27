// /internal/database/database.go
package database

import (
	"fmt"
	"log"
	"os"

	"github.com/ericoliveiras/meu-cupcake/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDB() {
	var err error

	// Lê a URL completa do ambiente
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL não encontrado no .env")
	}

	// Tenta abrir a conexão com GORM usando a URL completa
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Falha ao conectar ao banco de dados Neon usando URL: %v\nURL usada: %s", err, dsn)
	}

	fmt.Println("Conexão com o banco de dados Neon (via URL) estabelecida com sucesso.")

	// --- Auto Migration  ---
	fmt.Println("Executando migrações do banco de dados...")
	err = DB.AutoMigrate(
		&model.Usuario{}, &model.Cupcake{}, &model.Order{}, &model.ItemOrder{},
	)
	if err != nil {
		log.Fatal("Falha ao executar migrações:", err)
	}
	fmt.Println("Migrações concluídas com sucesso.")
}

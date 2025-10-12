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
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=America/Sao_Paulo",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)

	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Falha ao conectar ao banco de dados:", err)
	}
	fmt.Println("Conexão com o banco de dados estabelecida com sucesso.")

	fmt.Println("Executando migrações do banco de dados...")
	err = DB.AutoMigrate(&model.Usuario{})
	if err != nil {
		log.Fatal("Falha ao executar migrações:", err)
	}
	fmt.Println("Migrações concluídas com sucesso.")
}
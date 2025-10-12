// /cmd/web/main.go
package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/ericoliveiras/meu-cupcake/internal/database"
	"github.com/ericoliveiras/meu-cupcake/internal/handler"
	"github.com/gin-gonic/gin"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Erro ao carregar o arquivo .env")
	}
	
	database.ConnectDB()

	router := gin.Default()
	router.LoadHTMLGlob("internal/view/templates/*")
	router.GET("/", handler.ShowHomePage)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Servidor rodando na porta %s", port)
	router.Run(":" + port)
}

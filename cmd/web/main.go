// /cmd/web/main.go
package main

import (
	"log"
	"os"

	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"

	"github.com/ericoliveiras/meu-cupcake/internal/database"
	"github.com/ericoliveiras/meu-cupcake/internal/handler"
	"github.com/ericoliveiras/meu-cupcake/internal/model"
	"github.com/gin-gonic/gin"
)

var store *sessions.CookieStore

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Erro ao carregar o arquivo .env")
	}

	store = sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))

	authHandler := &handler.AuthHandler{Store: store}
	homeHandler := &handler.HomeHandler{Store: store}
	lojistaHandler := &handler.LojistaHandler{Store: store}

	database.ConnectDB()
	database.SeedLojista()

	router := gin.Default()
	router.LoadHTMLGlob("internal/view/templates/*")

	router.Static("/uploads", "./uploads")
	router.Static("/static", "./static")

	router.GET("/", homeHandler.ShowHomePage)
	router.GET("/vitrine", homeHandler.ShowVitrinePage)
	
	router.GET("/cadastro", handler.ShowCadastroPage)
	router.POST("/cadastro", handler.ProcessCadastroForm)
	router.GET("/login", authHandler.ShowLoginPage)       
	router.POST("/login", authHandler.ProcessLoginForm) 
	router.GET("/logout", authHandler.Logout)

	protected := router.Group("/")
	protected.Use(authHandler.AuthRequired())
	{
		protected.GET("/perfil", homeHandler.ShowProfilePage)
	}

	clienteRoutes := router.Group("/cliente")
	clienteRoutes.Use(authHandler.AuthRequired()) // Garante que só usuários logados acessem
	{
		clienteRoutes.GET("/dashboard", homeHandler.ShowClienteDashboard)
	}

	lojistaRoutes := router.Group("/lojista")
	lojistaRoutes.Use(authHandler.AuthRequired())
	lojistaRoutes.Use(authHandler.RoleRequired(model.RoleLojista))
	{
		lojistaRoutes.GET("/dashboard", lojistaHandler.ShowLojistaDashboard)
		lojistaRoutes.GET("/cupcakes", lojistaHandler.ShowCupcakesPage)
		lojistaRoutes.POST("/cupcakes/novo", lojistaHandler.ProcessNewCupcakeForm)
    lojistaRoutes.POST("/cupcakes/editar/:id", lojistaHandler.ProcessEditCupcakeForm)
    lojistaRoutes.GET("/cupcakes/excluir/:id", lojistaHandler.DeleteCupcake)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Servidor rodando na porta %s", port)
	router.Run(":" + port)
}

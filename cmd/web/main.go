// /cmd/web/main.go
package main

import (
	"encoding/gob"
	"log"
	"os"

	"github.com/ericoliveiras/meu-cupcake/internal/database"
	"github.com/ericoliveiras/meu-cupcake/internal/handler"
	"github.com/ericoliveiras/meu-cupcake/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"github.com/mercadopago/sdk-go/pkg/config"
)

var store *sessions.CookieStore

func main() {
	gob.Register(map[uint]int{})

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Erro ao carregar o arquivo .env")
	}

	mpAccessToken := os.Getenv("MP_ACCESS_TOKEN")
	if mpAccessToken == "" {
		log.Fatal("MP_ACCESS_TOKEN não encontrado no .env")
	}

	// Cria a configuração e armazena em 'cfg'
	cfg, err := config.New(mpAccessToken)
	if err != nil {
		log.Fatalf("Erro ao criar configuração do Mercado Pago: %v", err)
	}

	store = sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))

	// Cria instâncias dos handlers, passando dependências necessárias
	authHandler := &handler.AuthHandler{Store: store}
	homeHandler := &handler.HomeHandler{Store: store}
	lojistaHandler := &handler.LojistaHandler{Store: store}
	cartHandler := &handler.CartHandler{Store: store, MPCfg: cfg}

	database.ConnectDB()
	database.SeedLojista()

	router := gin.Default()
	router.LoadHTMLGlob("internal/view/templates/*")

	// Servir arquivos estáticos
	router.Static("/uploads", "./uploads")
	router.Static("/static", "./static")

	// --- Rotas Públicas ---
	router.GET("/", homeHandler.ShowHomePage)
	router.GET("/vitrine", homeHandler.ShowVitrinePage)
	router.POST("/carrinho/adicionar/:id", cartHandler.AddToCart)
	router.GET("/carrinho", cartHandler.ShowCartPage)
	router.POST("/carrinho/remover/:id", cartHandler.RemoveFromCart)
	router.POST("/carrinho/diminuir/:id", cartHandler.DecreaseQuantity)
	router.POST("/carrinho/limpar", cartHandler.ClearCart)
	router.GET("/pagamento/sucesso", homeHandler.ShowPagamentoSucessoPage)

	// --- Rotas de Autenticação ---
	router.GET("/cadastro", authHandler.ShowCadastroPage)
	router.POST("/cadastro", authHandler.ProcessCadastroForm)
	router.GET("/login", authHandler.ShowLoginPage)
	router.POST("/login", authHandler.ProcessLoginForm)
	router.GET("/logout", authHandler.Logout)

	// --- Rotas Protegidas Gerais (qualquer usuário logado) ---
	protected := router.Group("/")
	protected.Use(authHandler.AuthRequired())
	{
		protected.GET("/perfil", homeHandler.ShowProfilePage)
	}

	// --- Rotas Protegidas do Cliente ---
	clienteRoutes := router.Group("/cliente")
	clienteRoutes.Use(authHandler.AuthRequired())
	{
		clienteRoutes.GET("/dashboard", homeHandler.ShowClienteDashboard)
		clienteRoutes.GET("/checkout", cartHandler.ShowCheckoutPage)
		clienteRoutes.GET("/pedidos", homeHandler.ShowClientePedidosPage)
		clienteRoutes.POST("/processar-pagamento", cartHandler.ProcessPayment)
	}

	// --- Rotas Protegidas do Lojista ---
	lojistaRoutes := router.Group("/lojista")
	lojistaRoutes.Use(authHandler.AuthRequired())
	lojistaRoutes.Use(authHandler.RoleRequired(model.RoleLojista))
	{
		lojistaRoutes.GET("/dashboard", lojistaHandler.ShowLojistaDashboard)
		lojistaRoutes.GET("/cupcakes", lojistaHandler.ShowCupcakesPage)
		lojistaRoutes.POST("/cupcakes/novo", lojistaHandler.ProcessNewCupcakeForm)
		lojistaRoutes.POST("/cupcakes/editar/:id", lojistaHandler.ProcessEditCupcakeForm)
		lojistaRoutes.GET("/cupcakes/exclu</strong>ir/:id", lojistaHandler.DeleteCupcake)
		lojistaRoutes.GET("/vendas", lojistaHandler.ShowLojistaVendasPage)
	}

	// --- Inicialização do Servidor ---
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Servidor rodando na porta %s", port)
	router.Run(":" + port)
}

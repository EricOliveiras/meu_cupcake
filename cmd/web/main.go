// /cmd/web/main.go
package main

import (
	"encoding/gob"
	"fmt"
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
		fmt.Println("Aviso: Erro ao carregar o arquivo .env:", err)
		fmt.Println("Continuando execução, esperando variáveis de ambiente do sistema/segredos.")
	} else {
		fmt.Println("Arquivo .env carregado com sucesso.")
	}
	// ----------------------

	mpAccessToken := os.Getenv("MP_ACCESS_TOKEN")
	if mpAccessToken == "" {
		log.Fatal("FATAL: MP_ACCESS_TOKEN não encontrado no ambiente.")
	}
	cfg, err := config.New(mpAccessToken)
	if err != nil {
		log.Fatalf("Erro ao criar configuração do Mercado Pago: %v", err)
	}
	log.Println("SDK do Mercado Pago v2 configurado...")

	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		log.Fatal("FATAL: SESSION_SECRET não encontrado no ambiente.")
	}
	store = sessions.NewCookieStore([]byte(sessionSecret))

	// Cria instâncias dos handlers
	authHandler := &handler.AuthHandler{Store: store}
	homeHandler := &handler.HomeHandler{Store: store, MPCfg: cfg}
	lojistaHandler := &handler.LojistaHandler{Store: store, MPCfg: cfg}
	cartHandler := &handler.CartHandler{Store: store, MPCfg: cfg}

	// Conecta ao DB (ConnectDB deve ler DATABASE_URL do ambiente)
	database.ConnectDB()
	database.SeedLojista()

	router := gin.Default()

	// Configura GIN_MODE (lendo do ambiente ou padrão)
	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router.LoadHTMLGlob("internal/view/templates/*") // Caminho dentro do container

	// Servir arquivos estáticos (caminhos dentro do container)
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
	router.GET("/cadastro", authHandler.ShowCadastroPage)     // Assumindo método
	router.POST("/cadastro", authHandler.ProcessCadastroForm) // Assumindo método
	router.GET("/login", authHandler.ShowLoginPage)
	router.POST("/login", authHandler.ProcessLoginForm)
	router.GET("/logout", authHandler.Logout)

	// --- Rotas Protegidas Gerais ---
	protected := router.Group("/")
	protected.Use(authHandler.AuthRequired())
	{
		protected.GET("/perfil", homeHandler.ShowProfilePage)
		protected.GET("/perfil/editar", homeHandler.ShowEditProfilePage)   // Rota para mostrar o formulário
		protected.POST("/perfil/editar", homeHandler.ProcessEditProfileForm) // Rota para processar o formulário
	}

	// --- Rotas Protegidas do Cliente ---
	clienteRoutes := router.Group("/cliente")
	clienteRoutes.Use(authHandler.AuthRequired())
	{
		clienteRoutes.GET("/dashboard", homeHandler.ShowClienteDashboard)
		clienteRoutes.GET("/checkout", cartHandler.ShowCheckoutPage)
		clienteRoutes.GET("/pedidos", homeHandler.ShowClientePedidosPage)
		clienteRoutes.POST("/processar-pagamento", cartHandler.ProcessPayment)
		clienteRoutes.POST("/processar-pagamento-pix", cartHandler.ProcessPixPayment)
		clienteRoutes.GET("/pedido/pagamento/:id", homeHandler.ShowPedidoPagamentoPage)
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
		lojistaRoutes.GET("/cupcakes/excluir/:id", lojistaHandler.DeleteCupcake)
		lojistaRoutes.GET("/vendas", lojistaHandler.ShowLojistaVendasPage)
		lojistaRoutes.POST("/vendas/status/:id", lojistaHandler.UpdatePedidoStatus)
	}

	// --- Inicialização do Servidor ---
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Padrão
	}
	// Importante para Fly.io: Ouvir em 0.0.0.0
	listenAddr := fmt.Sprintf("0.0.0.0:%s", port)
	log.Printf("Servidor rodando em %s (Modo Gin: %s)", listenAddr, gin.Mode())
	err = router.Run(listenAddr) // Usa listenAddr
	if err != nil {
		log.Fatalf("Falha ao iniciar o servidor Gin: %v", err)
	}
}

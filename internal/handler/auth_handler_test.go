// /internal/handler/auth_handler_test.go
package handler

import (
	"fmt"
	"log" // Import log for error handling
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"            // Import os for Stat
	"path/filepath" // Import filepath to build paths
	"runtime"       // Import runtime to get current file path
	"strings"
	"testing"
	"time"

	"github.com/ericoliveiras/meu-cupcake/internal/database"
	"github.com/ericoliveiras/meu-cupcake/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/joho/godotenv" // Import godotenv
	"golang.org/x/crypto/bcrypt"
	// Import gorm for ProcessLoginForm test
)

// Helper function to get project root directory based on the test file's location
// Adjust the number of "../" if your test file moves relative to the root.
func getProjectRoot() string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("Could not get caller information")
	}
	// Assuming auth_handler_test.go is in internal/handler, go up two directories
	return filepath.Join(filepath.Dir(currentFile), "..", "..")
}

// TestShowLoginPage testa se a página de login é renderizada corretamente.
func TestShowLoginPage(t *testing.T) {
	// --- Configuração ---
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// --- Carregar Templates de Forma Robusta ---
	projectRoot := getProjectRoot()
	templatePattern := filepath.Join(projectRoot, "internal", "view", "templates", "*.html") // Usa *.html
	templateDir := filepath.Dir(templatePattern)

	fmt.Printf("DEBUG (TestShowLoginPage): Tentando carregar templates de: %s\n", templatePattern)

	// Verifica se o diretório existe
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		t.Fatalf("Diretório de templates não encontrado em: %s", templateDir)
	}

	// Tenta carregar os templates. LoadHTMLGlob causa pânico se o padrão não achar NADA.
	// Se der pânico aqui, verifique se existem arquivos .html na pasta templates.
	router.LoadHTMLGlob(templatePattern)
	// Adiciona uma verificação básica (não perfeita) se o renderer foi setado.
	if router.HTMLRender == nil {
		t.Fatal("Falha ao carregar templates HTML. Verifique o caminho e se existem arquivos .html.")
	}
	fmt.Println("DEBUG (TestShowLoginPage): Templates carregados com sucesso.")
	// ---------------------------------------------

	store := sessions.NewCookieStore([]byte("secret-key-for-test"))
	authHandler := &AuthHandler{Store: store}
	router.GET("/login", authHandler.ShowLoginPage)

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	recorder := httptest.NewRecorder()

	// --- Execução ---
	// Usamos um bloco recover para capturar o pânico e dar uma mensagem melhor
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("PÂNICO durante router.ServeHTTP em TestShowLoginPage: %v. Verifique se os templates foram carregados corretamente.", r)
		}
	}()
	router.ServeHTTP(recorder, req) // Esta linha estava causando o pânico

	// --- Verificação ---
	if status := recorder.Code; status != http.StatusOK {
		t.Errorf("ShowLoginPage status code incorreto: esperado %v obteve %v. Corpo: %s", http.StatusOK, status, recorder.Body.String())
	} else {
		// Continua verificando o corpo apenas se o status for OK
		expectedContentType := "text/html; charset=utf-8"
		if ctype := recorder.Header().Get("Content-Type"); !strings.HasPrefix(ctype, expectedContentType) {
			t.Errorf("ShowLoginPage Content-Type errado: esperado prefixo %s mas obteve %s", expectedContentType, ctype)
		}

		body := recorder.Body.String()
		expectedTitle := "<title>Login - Meu Cupcake</title>"
		if !strings.Contains(body, expectedTitle) {
			t.Errorf("ShowLoginPage: Corpo da resposta não contém o título esperado '%s'.", expectedTitle)
		}
		expectedH1 := "<h1>Acesse sua Conta</h1>"
		if !strings.Contains(body, expectedH1) {
			t.Errorf("ShowLoginPage: Corpo da resposta não contém o H1 esperado '%s'.", expectedH1)
		}
		expectedLink := `<a href="/cadastro">Cadastre-se</a>`
		if !strings.Contains(body, expectedLink) {
			t.Errorf("ShowLoginPage: Corpo da resposta não contém o link de cadastro esperado '%s'.", expectedLink)
		}
	}
}

// --- Função Auxiliar para Setup do Teste de ProcessLogin ---
// (Sem alterações significativas, apenas garante clareza)
func setupLoginTestRouter() (*gin.Engine, *AuthHandler) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	store := sessions.NewCookieStore([]byte("secret-key-for-test-login"))
	authHandler := &AuthHandler{Store: store}

	// Registra rotas necessárias
	router.POST("/login", authHandler.ProcessLoginForm)
	router.GET("/login", authHandler.ShowLoginPage) // Necessário para redirect de erro
	router.GET("/cliente/dashboard", func(c *gin.Context) { c.String(http.StatusOK, "Dashboard Cliente OK") })
	router.GET("/lojista/dashboard", func(c *gin.Context) { c.String(http.StatusOK, "Dashboard Lojista OK") })

	// Não carrega templates aqui, pois ProcessLoginForm só redireciona
	return router, authHandler
}

// --- Teste Principal para ProcessLoginForm ---
func TestProcessLoginForm(t *testing.T) {
	// --- Carregar Variáveis de Ambiente ---
	projectRoot := getProjectRoot()
	envPath := filepath.Join(projectRoot, ".env")
	fmt.Printf("DEBUG (TestProcessLoginForm): Tentando carregar .env de: %s\n", envPath)
	err := godotenv.Load(envPath) // Usa o caminho construído
	if err != nil {
		t.Fatalf("Erro crítico: Não foi possível carregar o arquivo .env para teste: %v.", err)
	}
	if os.Getenv("DATABASE_URL") == "" { // Verifica a variável específica
		t.Fatal("Erro crítico: Variável DATABASE_URL não encontrada após carregar .env.")
	}

	// --- Conectar ao Banco de Dados ---
	database.ConnectDB()
	if database.DB == nil {
		t.Fatal("Erro crítico: A conexão com o banco de dados (database.DB) é nula.")
	}

	router, _ := setupLoginTestRouter()

	// --- Dados e Setup de Teste no DB ---
	testPassword := "senhaValidaParaTeste123"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Erro ao gerar hash: %v", err)
	}

	clienteEmail := fmt.Sprintf("teste.login.cliente_%d@example.com", time.Now().UnixNano())
	lojistaEmail := fmt.Sprintf("teste.login.lojista_%d@example.com", time.Now().UnixNano())

	testUserCliente := model.Usuario{Nome: "Cliente Teste", Email: clienteEmail, SenhaHash: string(hashedPassword), Tipo: model.RoleCliente}
	testUserLojista := model.Usuario{Nome: "Lojista Teste", Email: lojistaEmail, SenhaHash: string(hashedPassword), Tipo: model.RoleLojista}

	// Limpeza inicial e criação
	database.DB.Unscoped().Where("email = ?", clienteEmail).Delete(&model.Usuario{})
	database.DB.Unscoped().Where("email = ?", lojistaEmail).Delete(&model.Usuario{})
	if err := database.DB.Create(&testUserCliente).Error; err != nil {
		t.Fatalf("Erro DB (cliente): %v", err)
	}
	if err := database.DB.Create(&testUserLojista).Error; err != nil {
		database.DB.Unscoped().Where("email = ?", clienteEmail).Delete(&model.Usuario{})
		t.Fatalf("Erro DB (lojista): %v", err)
	}

	t.Cleanup(func() {
		fmt.Println("Limpando usuários de teste do login...")
		database.DB.Unscoped().Where("email = ?", clienteEmail).Delete(&model.Usuario{})
		database.DB.Unscoped().Where("email = ?", lojistaEmail).Delete(&model.Usuario{})
	})

	// --- Cenários de Teste ---

	// Cenário 1: Sucesso Login Cliente
	t.Run("Sucesso Login Cliente", func(t *testing.T) {
		formData := url.Values{"email": {clienteEmail}, "senha": {testPassword}}
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(formData.Encode()))
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		if status := recorder.Code; status != http.StatusFound {
			t.Errorf("Sucesso Cliente: Status code incorreto: esperado %v obteve %v", http.StatusFound, status)
		}
		location := recorder.Header().Get("Location")
		if location != "/cliente/dashboard" {
			t.Errorf("Sucesso Cliente: URL de redirecionamento incorreta: esperado %s obteve %s", "/cliente/dashboard", location)
		}
	})

	// Cenário 2: Sucesso Login Lojista
	t.Run("Sucesso Login Lojista", func(t *testing.T) {
		formData := url.Values{"email": {lojistaEmail}, "senha": {testPassword}}
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(formData.Encode()))
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		if status := recorder.Code; status != http.StatusFound {
			t.Errorf("Sucesso Lojista: Status code incorreto: esperado %v obteve %v", http.StatusFound, status)
		}
		location := recorder.Header().Get("Location")
		if location != "/lojista/dashboard" {
			t.Errorf("Sucesso Lojista: URL de redirecionamento incorreta: esperado %s obteve %s", "/lojista/dashboard", location)
		}
	})

	// Cenário 3: Senha Incorreta
	t.Run("Senha Incorreta", func(t *testing.T) {
		formData := url.Values{"email": {clienteEmail}, "senha": {"senhaerrada"}}
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(formData.Encode()))
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		if status := recorder.Code; status != http.StatusFound {
			t.Errorf("Senha Incorreta: Status code incorreto: esperado %v obteve %v", http.StatusFound, status)
		}
		location := recorder.Header().Get("Location")
		if location != "/login" {
			t.Errorf("Senha Incorreta: URL de redirecionamento incorreta: esperado %s obteve %s", "/login", location)
		}
		// TODO: Testar flash message
	})

	// Cenário 4: Usuário Não Encontrado
	t.Run("Usuário Não Encontrado", func(t *testing.T) {
		formData := url.Values{"email": {"naoexiste@example.com"}, "senha": {"qualquercoisa"}}
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(formData.Encode()))
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		if status := recorder.Code; status != http.StatusFound {
			t.Errorf("Não Encontrado: Status code incorreto: esperado %v obteve %v", http.StatusFound, status)
		}
		location := recorder.Header().Get("Location")
		if location != "/login" {
			t.Errorf("Não Encontrado: URL de redirecionamento incorreta: esperado %s obteve %s", "/login", location)
		}
		// TODO: Testar flash message
	})
}

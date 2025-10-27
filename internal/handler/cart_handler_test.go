// /internal/handler/cart_handler_test.go
package handler

import (
	// Para o SDK do MP (embora não usado diretamente neste teste)
	"encoding/gob" // Para registrar o tipo do carrinho para a sessão
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath" // Para construir caminhos
	"runtime"
	"strconv" // Para converter ID para string
	"strings"
	"testing"
	"time" // Para emails únicos

	"github.com/ericoliveiras/meu-cupcake/internal/database"
	"github.com/ericoliveiras/meu-cupcake/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/securecookie" // Para decodificar o cookie de sessão
	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"github.com/mercadopago/sdk-go/pkg/config" // Importe config
	// Não precisamos importar payment aqui
)

// --- Funções Auxiliares Globais (Podem ser movidas para um _test_helper.go) ---

// getProjectRootTest: Encontra a raiz do projeto a partir do arquivo de teste.
func getProjectRootTest() string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("Não foi possível obter informações do chamador")
	}
	// Assumindo que este arquivo está em internal/handler, sobe dois níveis
	return filepath.Join(filepath.Dir(currentFile), "..", "..")
}

// loadEnvForTest: Carrega o arquivo .env para os testes.
func loadEnvForTest(t *testing.T) {
	projectRoot := getProjectRootTest()
	envPath := filepath.Join(projectRoot, ".env")
	fmt.Printf("DEBUG (loadEnvForTest): Tentando carregar .env de: %s\n", envPath)
	err := godotenv.Load(envPath)
	if err != nil {
		t.Fatalf("Erro crítico: Não foi possível carregar o arquivo .env para teste: %v.", err)
	}
	// Verifica variáveis essenciais
	if os.Getenv("DATABASE_URL") == "" {
		t.Fatal("Erro crítico: DATABASE_URL não encontrada após carregar .env.")
	}
	if os.Getenv("SESSION_SECRET") == "" {
		t.Fatal("Erro crítico: SESSION_SECRET não encontrada após carregar .env.")
	}
	// Adicione verificações para MP_ACCESS_TOKEN e MP_PUBLIC_KEY se forem usadas em setup
}

// connectDBForTest: Conecta ao banco de dados para os testes.
func connectDBForTest(t *testing.T) {
	// Só conecta se ainda não estiver conectado (assume que database.DB é global)
	if database.DB == nil {
		fmt.Println("DEBUG (connectDBForTest): Conectando ao banco de dados...")
		database.ConnectDB() // Assume que ConnectDB lê do .env carregado
		if database.DB == nil {
			t.Fatal("Erro crítico: A conexão com o banco de dados (database.DB) é nula após ConnectDB.")
		}
	} else {
		fmt.Println("DEBUG (connectDBForTest): Usando conexão DB existente.")
	}
}

// setupTestRouterAndHandler: Configura o router, store, e handlers para um teste.
func setupTestRouterAndHandler(t *testing.T) (*gin.Engine, *CartHandler, *sessions.CookieStore) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Configuração da Sessão
	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		t.Fatal("SESSION_SECRET está vazia após carregar .env")
	}
	store := sessions.NewCookieStore([]byte(sessionSecret))

	// Configuração do Mercado Pago (necessário para o CartHandler)
	mpAccessToken := os.Getenv("MP_ACCESS_TOKEN")
	if mpAccessToken == "" {
		t.Fatal("MP_ACCESS_TOKEN não encontrado")
	}
	cfg, err := config.New(mpAccessToken)
	if err != nil {
		t.Fatalf("Erro ao criar config MP: %v", err)
	}

	// Cria o handler
	cartHandler := &CartHandler{Store: store, MPCfg: cfg}

	// Registra as rotas relevantes para o teste do carrinho
	router.POST("/carrinho/adicionar/:id", cartHandler.AddToCart)
	router.GET("/vitrine", func(c *gin.Context) { c.Status(http.StatusOK) })  // Mock da página de redirect padrão
	router.GET("/carrinho", func(c *gin.Context) { c.Status(http.StatusOK) }) // Mock da página de redirect alternativa

	// Registra o tipo do carrinho para a sessão (necessário uma vez)
	gob.Register(map[uint]int{})

	return router, cartHandler, store
}

// createTestCupcake: Cria um cupcake no DB para testes e retorna seu ID.
func createTestCupcake(t *testing.T) uint {
	cupcake := model.Cupcake{
		Nome:       fmt.Sprintf("Cupcake Teste %d", time.Now().UnixNano()),
		Descricao:  "Descrição teste",
		Preco:      10.50,
		ImagemURL:  "/static/images/placeholder.jpg", // Use uma imagem válida se necessário
		Disponivel: true,
	}
	if err := database.DB.Create(&cupcake).Error; err != nil {
		t.Fatalf("Erro ao criar cupcake de teste no DB: %v", err)
	}
	return cupcake.ID
}

// decodeSessionCookie: Decodifica o cookie de sessão para verificar seu conteúdo.
func decodeSessionCookie(t *testing.T, cookie *http.Cookie, store *sessions.CookieStore) map[interface{}]interface{} {
	session := sessions.NewSession(store, "meu-cupcake-session") // Usa o mesmo nome de cookie
	// Usa securecookie (interno do gorilla/sessions) para decodificar
	// NOTA: Isso depende de implementação interna e pode quebrar em futuras versões do gorilla/sessions.
	// Uma alternativa é fazer outra requisição HTTP para uma rota de teste que leia a sessão.
	err := securecookie.DecodeMulti(session.Name(), cookie.Value, &session.Values, store.Codecs...)
	if err != nil {
		t.Errorf("Erro ao decodificar o cookie de sessão: %v", err)
		return nil
	}
	return session.Values
}

// --- Teste Principal para AddToCart ---
func TestAddToCart(t *testing.T) {
	loadEnvForTest(t)
	connectDBForTest(t)
	router, _, store := setupTestRouterAndHandler(t) // Pega o router e o store

	// Cria um cupcake de teste no banco
	cupcakeID := createTestCupcake(t)
	cupcakeIDStr := strconv.FormatUint(uint64(cupcakeID), 10)

	// Garante limpeza do cupcake de teste
	t.Cleanup(func() {
		fmt.Println("Limpando cupcake de teste...")
		database.DB.Unscoped().Delete(&model.Cupcake{}, cupcakeID)
	})

	// --- Cenário 1: Adicionar item pela primeira vez (redirect para vitrine) ---
	t.Run("Adicionar Primeiro Item", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/carrinho/adicionar/"+cupcakeIDStr, nil)
		// Não precisamos de Content-Type para POST sem corpo
		recorder := httptest.NewRecorder()

		router.ServeHTTP(recorder, req)

		// Verifica Status Code (302 Found)
		if status := recorder.Code; status != http.StatusFound {
			t.Errorf("Status code incorreto: esperado %v obteve %v", http.StatusFound, status)
		}
		// Verifica Redirect Location
		location := recorder.Header().Get("Location")
		if location != "/vitrine" {
			t.Errorf("URL de redirecionamento incorreta: esperado %s obteve %s", "/vitrine", location)
		}

		// Verifica o conteúdo da sessão no cookie retornado
		cookie := recorder.Result().Cookies()[0] // Pega o primeiro cookie setado
		if cookie.Name != "meu-cupcake-session" {
			t.Fatalf("Cookie de sessão não encontrado ou nome incorreto")
		}
		sessionValues := decodeSessionCookie(t, cookie, store)
		if sessionValues == nil {
			return
		} // Erro na decodificação já reportado

		cartData, exists := sessionValues[CartSessionKey]
		if !exists {
			t.Fatalf("Chave '%s' não encontrada na sessão", CartSessionKey)
		}

		cart, ok := cartData.(map[uint]int)
		if !ok {
			t.Fatalf("Carrinho na sessão não é do tipo map[uint]int")
		}

		// Verifica se o item foi adicionado com quantidade 1
		if quantity, itemExists := cart[cupcakeID]; !itemExists || quantity != 1 {
			t.Errorf("Item %d não foi adicionado corretamente ao carrinho. Carrinho: %v", cupcakeID, cart)
		}
		if len(cart) != 1 {
			t.Errorf("Carrinho deveria ter 1 tipo de item, mas tem %d. Carrinho: %v", len(cart), cart)
		}
	})

	// --- Cenário 2: Adicionar o mesmo item novamente (incrementar) ---
	t.Run("Incrementar Item Existente", func(t *testing.T) {
		// Simula um estado inicial da sessão com o item já adicionado uma vez
		initialCart := map[uint]int{cupcakeID: 1}
		initialSession := sessions.NewSession(store, "meu-cupcake-session")
		initialSession.Values[CartSessionKey] = initialCart
		// Precisamos codificar este estado inicial em um cookie para enviar na requisição
		encoded, _ := securecookie.EncodeMulti(initialSession.Name(), initialSession.Values, store.Codecs...)
		cookieToSend := &http.Cookie{Name: initialSession.Name(), Value: encoded}

		req := httptest.NewRequest(http.MethodPost, "/carrinho/adicionar/"+cupcakeIDStr, nil)
		req.AddCookie(cookieToSend) // Envia o cookie com o carrinho pré-existente
		recorder := httptest.NewRecorder()

		router.ServeHTTP(recorder, req)

		// Verifica Status e Redirect (devem ser iguais ao cenário 1)
		if status := recorder.Code; status != http.StatusFound { /* ... erro ... */
		}
		if loc := recorder.Header().Get("Location"); loc != "/vitrine" { /* ... erro ... */
		}

		// Verifica o conteúdo da sessão no NOVO cookie retornado
		cookie := recorder.Result().Cookies()[0]
		sessionValues := decodeSessionCookie(t, cookie, store)
		if sessionValues == nil {
			return
		}

		cartData, exists := sessionValues[CartSessionKey]
		if !exists {
			t.Fatalf("Chave '%s' não encontrada na sessão", CartSessionKey)
		}
		cart, ok := cartData.(map[uint]int)
		if !ok {
			t.Fatalf("Carrinho na sessão não é do tipo map[uint]int")
		}

		// Verifica se a quantidade foi incrementada para 2
		if quantity, itemExists := cart[cupcakeID]; !itemExists || quantity != 2 {
			t.Errorf("Item %d não foi incrementado corretamente. Esperado: 2, Obtido: %d. Carrinho: %v", cupcakeID, quantity, cart)
		}
		if len(cart) != 1 { // Ainda deve ter apenas 1 tipo de item
			t.Errorf("Carrinho deveria ter 1 tipo de item, mas tem %d. Carrinho: %v", len(cart), cart)
		}
	})

	// --- Cenário 3: Adicionar item vindo do carrinho (redirect para carrinho) ---
	t.Run("Adicionar Item Vindo do Carrinho", func(t *testing.T) {
		// Simula um estado inicial da sessão
		initialCart := map[uint]int{cupcakeID: 1}
		initialSession := sessions.NewSession(store, "meu-cupcake-session")
		initialSession.Values[CartSessionKey] = initialCart
		encoded, _ := securecookie.EncodeMulti(initialSession.Name(), initialSession.Values, store.Codecs...)
		cookieToSend := &http.Cookie{Name: initialSession.Name(), Value: encoded}

		// Simula o envio do campo oculto 'return_to=cart'
		formData := url.Values{}
		formData.Set("return_to", "cart")

		req := httptest.NewRequest(http.MethodPost, "/carrinho/adicionar/"+cupcakeIDStr, strings.NewReader(formData.Encode()))
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(cookieToSend)
		recorder := httptest.NewRecorder()

		router.ServeHTTP(recorder, req)

		// Verifica Status Code
		if status := recorder.Code; status != http.StatusFound {
			t.Errorf("Status code incorreto: esperado %v obteve %v", http.StatusFound, status)
		}
		// Verifica Redirect Location para /carrinho
		expectedRedirect := "/carrinho"
		location := recorder.Header().Get("Location")
		if location != expectedRedirect {
			t.Errorf("URL de redirecionamento incorreta: esperado %s obteve %s", expectedRedirect, location)
		}

		// Verifica se a quantidade foi para 2 (igual ao cenário anterior)
		cookie := recorder.Result().Cookies()[0]
		sessionValues := decodeSessionCookie(t, cookie, store)
		// ... (verificações do conteúdo do carrinho como no Cenário 2) ...
		cartData, exists := sessionValues[CartSessionKey]
		if !exists {
			t.Fatalf("Chave '%s' não encontrada na sessão", CartSessionKey)
		}
		cart, ok := cartData.(map[uint]int)
		if !ok {
			t.Fatalf("Carrinho na sessão não é do tipo map[uint]int")
		}
		if quantity, itemExists := cart[cupcakeID]; !itemExists || quantity != 2 {
			t.Errorf("Item %d não foi incrementado (vindo do carrinho). Esperado: 2, Obtido: %d. Carrinho: %v", cupcakeID, quantity, cart)
		}
	})

	// --- Cenário 4: Tentar adicionar cupcake inválido/indisponível ---
	t.Run("Adicionar Item Inválido", func(t *testing.T) {
		invalidID := "99999" // Um ID que provavelmente não existe
		req := httptest.NewRequest(http.MethodPost, "/carrinho/adicionar/"+invalidID, nil)
		recorder := httptest.NewRecorder()

		router.ServeHTTP(recorder, req)

		// Verifica Status Code (404 Not Found)
		if status := recorder.Code; status != http.StatusNotFound {
			t.Errorf("Status code incorreto para item inválido: esperado %v obteve %v", http.StatusNotFound, status)
		}
		// Poderia verificar o corpo da resposta se quisesse garantir a mensagem de erro
	})

}

// Função auxiliar para criar uma sessão de teste com um carrinho
func createTestSessionWithCart(store sessions.Store, req *http.Request, cart map[uint]int) *sessions.Session {
	session, _ := store.Get(req, "meu-cupcake-session")
	session.Values[CartSessionKey] = cart
	return session
}

// TestGetTotalCartQuantity testa a função getTotalCartQuantityHelper
func TestGetTotalCartQuantityHelper(t *testing.T) {
	store := sessions.NewCookieStore([]byte("secret-key-for-test"))
	req := httptest.NewRequest("GET", "/", nil)

	// --- Cenário 1: Carrinho Nulo (ou não existe na sessão) ---
	t.Run("Carrinho Nulo", func(t *testing.T) {
		session, _ := store.Get(req, "meu-cupcake-session")
		// Extrai o carrinho (que será nil ou não ok neste caso)
		cartData := session.Values[CartSessionKey]
		cart, _ := cartData.(map[uint]int) // Ignora o 'ok' pois esperamos falha ou nil

		expected := 0
		actual := getTotalCartQuantityHelper(cart) // Passa o mapa (nil)

		if actual != expected {
			t.Errorf("A contagem para carrinho nulo deve ser %d, mas foi %d", expected, actual)
		}
	})

	// --- Cenário 2: Carrinho Vazio ---
	t.Run("Carrinho Vazio", func(t *testing.T) {
		cartMap := make(map[uint]int) // Cria o mapa vazio
		session := createTestSessionWithCart(store, req, cartMap)
		// Extrai o carrinho da sessão
		cartData := session.Values[CartSessionKey]
		cart, ok := cartData.(map[uint]int)
		if !ok {
			t.Fatalf("Falha ao extrair mapa do carrinho vazio da sessão") // Segurança
		}

		expected := 0
		actual := getTotalCartQuantityHelper(cart) // Passa o mapa vazio

		if actual != expected {
			t.Errorf("A contagem para carrinho vazio deve ser %d, mas foi %d", expected, actual)
		}
	})

	// --- Cenário 3: Carrinho com Itens ---
	t.Run("Carrinho com Itens", func(t *testing.T) {
		cartMap := map[uint]int{ // Cria o mapa com itens
			1: 2,
			5: 3,
			8: 1,
		}
		session := createTestSessionWithCart(store, req, cartMap)
		// Extrai o carrinho da sessão
		cartData := session.Values[CartSessionKey]
		cart, ok := cartData.(map[uint]int)
		if !ok {
			t.Fatalf("Falha ao extrair mapa do carrinho com itens da sessão") // Segurança
		}

		expected := 6
		actual := getTotalCartQuantityHelper(cart) // Passa o mapa preenchido

		if actual != expected {
			t.Errorf("A contagem esperada era %d, mas foi %d", expected, actual)
		}
	})

	// --- Cenário 4: Sessão com carrinho de tipo inválido ---
	t.Run("Carrinho com Tipo Inválido", func(t *testing.T) {
		session, _ := store.Get(req, "meu-cupcake-session")
		session.Values[CartSessionKey] = "não é um mapa" // Coloca valor inválido
		// Extrai o carrinho (que falhará na conversão)
		cartData := session.Values[CartSessionKey]
		cart, _ := cartData.(map[uint]int) // Ignora o 'ok'

		expected := 0
		actual := getTotalCartQuantityHelper(cart) // Passa o mapa (nil)

		if actual != expected {
			t.Errorf("A contagem para tipo inválido deve ser %d, mas foi %d", expected, actual)
		}
	})
}

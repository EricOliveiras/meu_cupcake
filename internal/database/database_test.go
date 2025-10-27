// /internal/database/database_test.go
package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"testing" // Pacote de testes padrão

	"github.com/joho/godotenv" // Para carregar .env
	// Import GORM só se precisar interagir com DB diretamente no teste (não necessário aqui)
	// "gorm.io/gorm"
)

// --- Funções Auxiliares (mantidas) ---

// getProjectRootTest: Encontra a raiz do projeto.
func getProjectRootTest() string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		// Panic aqui é aceitável em um helper de teste se algo muito errado ocorrer
		log.Panic("Não foi possível obter informações do chamador no teste")
	}
	return filepath.Join(filepath.Dir(currentFile), "..", "..")
}

// loadEnvForTest: Carrega o arquivo .env.
func loadEnvForTest(t *testing.T) {
	projectRoot := getProjectRootTest()
	envPath := filepath.Join(projectRoot, ".env")
	fmt.Printf("DEBUG (loadEnvForTest - database): Tentando carregar .env de: %s\n", envPath)
	err := godotenv.Load(envPath)
	if err != nil {
		t.Fatalf("Erro crítico: Não foi possível carregar o arquivo .env para teste: %v.", err)
	}
	if os.Getenv("DATABASE_URL") == "" {
		t.Fatal("Erro crítico: Variável DATABASE_URL não encontrada/vazia após carregar .env.")
	}
	fmt.Println("DEBUG (loadEnvForTest - database): .env carregado com sucesso.")
}

// --- Teste Principal para ConnectDB (Simplificado) ---

func TestConnectDB(t *testing.T) {
	// Garante que a variável DB seja resetada após o teste
	// para não interferir em outros testes (se houver)
	originalDB := DB                   // Salva o estado atual (provavelmente nil)
	defer func() { DB = originalDB }() // Restaura no final

	// 1. Carrega as variáveis de ambiente
	loadEnvForTest(t)

	// 2. Chama a função ConnectDB (que pode chamar log.Fatalf em caso de erro real)
	// Não precisamos mais mockar log.Fatalf nem usar goroutine aqui.
	// Se ConnectDB falhar e chamar log.Fatalf, o processo de teste inteiro vai parar,
	// o que é um resultado aceitável para um erro crítico de conexão.
	ConnectDB()

	// 3. Verifica se a variável global DB foi inicializada
	if DB == nil {
		// Se ConnectDB tivesse retornado um erro em vez de Fatal, testaríamos o erro aqui.
		// Como ele usa Fatal, só podemos testar se DB foi setado após a chamada (se não houve Fatal).
		t.Fatal("ConnectDB completou (não chamou Fatalf), mas database.DB ainda é nil. Verifique a lógica de atribuição em ConnectDB.")
	}

	// 4. Tenta obter o sql.DB e fazer Ping
	sqlDB, err := DB.DB()
	if err != nil {
		t.Fatalf("Falha ao obter o objeto sql.DB do GORM após conexão bem-sucedida: %v", err)
	}

	err = sqlDB.Ping()
	if err != nil {
		// Se o Ping falhar, a conexão estabelecida pelo GORM não está funcional.
		t.Fatalf("Falha ao fazer ping no banco de dados após ConnectDB: %v", err)
	}

	// Se chegou aqui, a conexão foi estabelecida e está ativa.
	t.Log("Teste ConnectDB passou: Conexão estabelecida e Ping bem-sucedido.")
}

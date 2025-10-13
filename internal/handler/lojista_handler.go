package handler

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/google/uuid"

	"github.com/ericoliveiras/meu-cupcake/internal/database"
	"github.com/ericoliveiras/meu-cupcake/internal/model"
	"github.com/gin-gonic/gin"
)

func ShowLojistaDashboard(c *gin.Context) {
	c.HTML(http.StatusOK, "lojista_dashboard.html", nil)
}

func ShowCupcakesPage(c *gin.Context) {
	var cupcakes []model.Cupcake
	// Busca todos os cupcakes no banco, ordenados do mais recente para o mais antigo
	if err := database.DB.Order("created_at desc").Find(&cupcakes).Error; err != nil {
		// Em um caso real, seria bom logar o erro
		c.String(http.StatusInternalServerError, "Erro ao buscar cupcakes.")
		return
	}

	// Renderiza a view passando a lista de cupcakes
	c.HTML(http.StatusOK, "lojista_cupcakes.html", gin.H{
		"Cupcakes": cupcakes,
	})
}

// ShowNewCupcakeForm apenas renderiza a página com o formulário de cadastro.
func ShowNewCupcakeForm(c *gin.Context) {
	c.HTML(http.StatusOK, "lojista_cupcake_form.html", nil)
}

func ProcessNewCupcakeForm(c *gin.Context) {
	// 1. Parse dos campos do formulário
	nome := c.PostForm("nome")
	descricao := c.PostForm("descricao")
	precoStr := c.PostForm("preco")
	disponivelStr := c.PostForm("disponivel") // Retorna "true" se marcado, "" se não

	// 2. Processamento do Upload da Imagem
	file, err := c.FormFile("imagem")
	if err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("Erro ao obter arquivo: %s", err.Error()))
		return
	}

	// Gere um nome de arquivo único para evitar conflitos
	extensao := filepath.Ext(file.Filename)
	novoNomeArquivo := uuid.New().String() + extensao
	caminhoDestino := filepath.Join("uploads", novoNomeArquivo)

	// Salve o arquivo no destino
	if err := c.SaveUploadedFile(file, caminhoDestino); err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("Erro ao salvar arquivo: %s", err.Error()))
		return
	}

	// 3. Conversão e Validação dos Dados
	preco, err := strconv.ParseFloat(precoStr, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "O preço fornecido é inválido.")
		return
	}
	disponivel := disponivelStr == "true"

	// 4. Criação do Objeto Cupcake
	novoCupcake := model.Cupcake{
		Nome:      nome,
		Descricao: descricao,
		Preco:     preco,
		Disponivel: disponivel,
		ImagemURL:  fmt.Sprintf("/uploads/%s", novoNomeArquivo),
	}

	// 5. Salvar no Banco de Dados
	if err := database.DB.Create(&novoCupcake).Error; err != nil {
		c.String(http.StatusInternalServerError, "Erro ao salvar cupcake no banco de dados.")
		return
	}

	// 6. Redirecionar para a página de listagem em caso de sucesso
	c.Redirect(http.StatusFound, "/lojista/cupcakes")
}
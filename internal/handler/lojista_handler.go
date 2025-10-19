package handler

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/ericoliveiras/meu-cupcake/internal/database"
	"github.com/ericoliveiras/meu-cupcake/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/sessions" // Importe o pacote de sessões
)

// LojistaHandler agrupa os handlers do lojista e suas dependências.
type LojistaHandler struct {
	Store *sessions.CookieStore
}

// getSessionData é uma função helper para buscar os dados do usuário da sessão.
func (h *LojistaHandler) getSessionData(c *gin.Context) (model.Usuario, bool) {
	session, _ := h.Store.Get(c.Request, "meu-cupcake-session")
	userID, ok := session.Values["userID"].(uint)
	if !ok {
		return model.Usuario{}, false
	}

	var user model.Usuario
	if err := database.DB.First(&user, userID).Error; err != nil {
		return model.Usuario{}, false
	}
	return user, true
}

// ShowLojistaDashboard renderiza o painel principal do lojista.
func (h *LojistaHandler) ShowLojistaDashboard(c *gin.Context) {
	user, isLoggedIn := h.getSessionData(c)

	c.HTML(http.StatusOK, "lojista_dashboard.html", gin.H{
		"IsLoggedIn": isLoggedIn,
		"User":       user,
	})
}

// ShowCupcakesPage busca todos os cupcakes e renderiza a página de gerenciamento.
func (h *LojistaHandler) ShowCupcakesPage(c *gin.Context) {
	user, isLoggedIn := h.getSessionData(c)
	var cupcakes []model.Cupcake

	if err := database.DB.Order("created_at desc").Find(&cupcakes).Error; err != nil {
		c.String(http.StatusInternalServerError, "Erro ao buscar cupcakes.")
		return
	}

	c.HTML(http.StatusOK, "lojista_cupcakes.html", gin.H{
		"IsLoggedIn": isLoggedIn,
		"User":       user,
		"Cupcakes":   cupcakes,
	})
}

// ProcessNewCupcakeForm processa o formulário de criação de cupcake.
func (h *LojistaHandler) ProcessNewCupcakeForm(c *gin.Context) {
	nome := c.PostForm("nome")
	descricao := c.PostForm("descricao")
	precoStr := c.PostForm("preco")
	disponivelStr := c.PostForm("disponivel")

	file, err := c.FormFile("imagem")
	if err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("Erro ao obter arquivo: %s", err.Error()))
		return
	}

	extensao := filepath.Ext(file.Filename)
	novoNomeArquivo := uuid.New().String() + extensao
	caminhoDestino := filepath.Join("uploads", novoNomeArquivo)

	if err := c.SaveUploadedFile(file, caminhoDestino); err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("Erro ao salvar arquivo: %s", err.Error()))
		return
	}

	preco, err := strconv.ParseFloat(precoStr, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "O preço fornecido é inválido.")
		return
	}
	disponivel := disponivelStr == "true"

	novoCupcake := model.Cupcake{
		Nome:       nome,
		Descricao:  descricao,
		Preco:      preco,
		Disponivel: disponivel,
		ImagemURL:  fmt.Sprintf("/uploads/%s", novoNomeArquivo),
	}

	if err := database.DB.Create(&novoCupcake).Error; err != nil {
		c.String(http.StatusInternalServerError, "Erro ao salvar cupcake no banco de dados.")
		return
	}

	c.Redirect(http.StatusFound, "/lojista/cupcakes")
}

func (h *LojistaHandler) DeleteCupcake(c *gin.Context) {
    idStr := c.Param("id")
    id, err := strconv.ParseUint(idStr, 10, 32)
    if err != nil {
        c.String(http.StatusBadRequest, "ID inválido.")
        return
    }

    // 1. Antes de deletar, buscar o registro do cupcake para pegar o caminho da imagem.
    var cupcake model.Cupcake
    if err := database.DB.First(&cupcake, id).Error; err != nil {
        c.String(http.StatusNotFound, "Cupcake não encontrado.")
        return
    }

    // 2. Construir o caminho do arquivo no sistema de arquivos.
    // (Removemos a barra inicial se ela existir para criar um caminho local)
    filePath := cupcake.ImagemURL
    if len(filePath) > 0 && filePath[0] == '/' {
        filePath = filePath[1:]
    }

    // 3. Tentar remover o arquivo de imagem do disco.
    if err := os.Remove(filePath); err != nil {
        // Se o arquivo não existir, não é um erro crítico.
        // Logamos o erro no console, mas continuamos o processo.
        fmt.Printf("Aviso: não foi possível remover o arquivo %s: %v\n", filePath, err)
    }

    // 4. Executa a exclusão do registro no banco (soft delete).
    if err := database.DB.Delete(&model.Cupcake{}, id).Error; err != nil {
        c.String(http.StatusInternalServerError, "Erro ao excluir o cupcake do banco de dados.")
        return
    }

    c.Redirect(http.StatusFound, "/lojista/cupcakes")
}

// ProcessEditCupcakeForm processa os dados do formulário de edição.
func (h *LojistaHandler) ProcessEditCupcakeForm(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "ID inválido.")
		return
	}

	var cupcake model.Cupcake
	if err := database.DB.First(&cupcake, id).Error; err != nil {
		c.String(http.StatusNotFound, "Cupcake não encontrado.")
		return
	}

	// Atualiza os campos com os novos valores do formulário
	cupcake.Nome = c.PostForm("nome")
	cupcake.Descricao = c.PostForm("descricao")
	preco, _ := strconv.ParseFloat(c.PostForm("preco"), 64)
	cupcake.Preco = preco
	cupcake.Disponivel = c.PostForm("disponivel") == "true"

	// Processa a imagem SOMENTE se uma nova foi enviada
	file, err := c.FormFile("imagem")
	if err == nil { // Se err for nil, significa que um novo arquivo foi enviado
		extensao := filepath.Ext(file.Filename)
		novoNomeArquivo := uuid.New().String() + extensao
		caminhoDestino := filepath.Join("uploads", novoNomeArquivo)
		if err := c.SaveUploadedFile(file, caminhoDestino); err == nil {
			cupcake.ImagemURL = fmt.Sprintf("/uploads/%s", novoNomeArquivo)
		}
	}

	// Salva as alterações no banco de dados
	if err := database.DB.Save(&cupcake).Error; err != nil {
		c.String(http.StatusInternalServerError, "Erro ao atualizar o cupcake.")
		return
	}

	c.Redirect(http.StatusFound, "/lojista/cupcakes")
}
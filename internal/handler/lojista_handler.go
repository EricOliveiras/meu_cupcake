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
	"github.com/gorilla/sessions"
)

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
	user.Tipo = model.RoleLojista
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

// ShowEditCupcakeForm busca um cupcake pelo ID e exibe o formulário de edição (via modal, esta função não é mais usada para renderizar HTML diretamente)
func (h *LojistaHandler) ShowEditCupcakeForm(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido."})
		return
	}

	var cupcake model.Cupcake
	if err := database.DB.First(&cupcake, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cupcake não encontrado."})
		return
	}
	c.JSON(http.StatusOK, cupcake)
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

	cupcake.Nome = c.PostForm("nome")
	cupcake.Descricao = c.PostForm("descricao")
	preco, _ := strconv.ParseFloat(c.PostForm("preco"), 64)
	cupcake.Preco = preco
	cupcake.Disponivel = c.PostForm("disponivel") == "true"

	file, err := c.FormFile("imagem")
	if err == nil {
		extensao := filepath.Ext(file.Filename)
		novoNomeArquivo := uuid.New().String() + extensao
		caminhoDestino := filepath.Join("uploads", novoNomeArquivo)
		if err := c.SaveUploadedFile(file, caminhoDestino); err == nil {
			// Opcional: remover imagem antiga antes de salvar a nova URL
			// os.Remove(cupcake.ImagemURL[1:]) // Remove a barra inicial antes de tentar deletar
			cupcake.ImagemURL = fmt.Sprintf("/uploads/%s", novoNomeArquivo)
		}
	}

	if err := database.DB.Save(&cupcake).Error; err != nil {
		c.String(http.StatusInternalServerError, "Erro ao atualizar o cupcake.")
		return
	}

	c.Redirect(http.StatusFound, "/lojista/cupcakes")
}

// DeleteCupcake remove um cupcake do banco de dados e o arquivo de imagem associado.
func (h *LojistaHandler) DeleteCupcake(c *gin.Context) {
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

	filePath := cupcake.ImagemURL
	if len(filePath) > 0 && filePath[0] == '/' {
		filePath = filePath[1:]
	}

	if err := os.Remove(filePath); err != nil {
		fmt.Printf("Aviso: não foi possível remover o arquivo %s: %v\n", filePath, err)
	}

	if err := database.DB.Delete(&model.Cupcake{}, id).Error; err != nil {
		c.String(http.StatusInternalServerError, "Erro ao excluir o cupcake do banco de dados.")
		return
	}

	c.Redirect(http.StatusFound, "/lojista/cupcakes")
}

func (h *LojistaHandler) ShowLojistaVendasPage(c *gin.Context) {
	user, isLoggedIn := h.getSessionData(c)

	var vendas []model.Order
	err := database.DB.Preload("Usuario").
		Preload("Items.Cupcake").
		Order("created_at desc").
		Find(&vendas).Error

	if err != nil {
		fmt.Printf("Erro ao buscar vendas para o lojista: %v\n", err)
		c.HTML(http.StatusOK, "lojista_vendas.html", gin.H{
			"IsLoggedIn": isLoggedIn,
			"User":       user,
			"Vendas":     []model.Order{},
			"ErrorMsg":   "Erro ao carregar histórico de vendas.",
		})
		return
	}

	c.HTML(http.StatusOK, "lojista_vendas.html", gin.H{
		"IsLoggedIn": isLoggedIn,
		"User":       user,
		"Vendas":     vendas,
	})
}

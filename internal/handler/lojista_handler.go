package handler

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/ericoliveiras/meu-cupcake/internal/database"
	"github.com/ericoliveiras/meu-cupcake/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/mercadopago/sdk-go/pkg/config"
)

const defaultCupcakeImage = "/static/images/placeholder.png"

type LojistaHandler struct {
	Store *sessions.CookieStore
	MPCfg *config.Config
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

	preco, err := strconv.ParseFloat(precoStr, 64)
	if err != nil {
		log.Printf("Erro ao converter preço: %v", err)
		c.Redirect(http.StatusSeeOther, "/lojista/cupcakes?error=Preço inválido")
		return
	}

	disponivel := disponivelStr == "true"

	var imagemURL = defaultCupcakeImage

	file, err := c.FormFile("imagem")

	if err == nil {
		ext := filepath.Ext(file.Filename)
		newFileName := uuid.New().String() + ext
		uploadPath := filepath.Join("uploads", newFileName)

		if err := c.SaveUploadedFile(file, uploadPath); err != nil {
			log.Printf("Erro ao salvar imagem: %v", err)
			c.Redirect(http.StatusSeeOther, "/lojista/cupcakes?error=Erro ao salvar imagem")
			return
		}

		imagemURL = "/" + uploadPath

	} else if err != http.ErrMissingFile {
		log.Printf("Erro ao processar form-file: %v", err)
		c.Redirect(http.StatusSeeOther, "/lojista/cupcakes?error=Erro ao processar arquivo de imagem")
		return
	}

	cupcake := model.Cupcake{
		Nome:       nome,
		Descricao:  descricao,
		Preco:      preco,
		Disponivel: disponivel,
		ImagemURL:  imagemURL,
	}

	if err := database.DB.Create(&cupcake).Error; err != nil {
		log.Printf("Erro ao criar cupcake no DB: %v", err)
		c.Redirect(http.StatusSeeOther, "/lojista/cupcakes?error=Erro ao salvar no banco de dados")
		return
	}

	log.Println("Novo cupcake criado com sucesso.")
	c.Redirect(http.StatusSeeOther, "/lojista/cupcakes")
}

// ShowEditCupcakeForm busca um cupcake pelo ID e exibe o formulário de edição (via modal, esta função não é mais usada para renderizar HTML diretamente)
func (h *LojistaHandler) ProcessEditCupcakeForm(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/lojista/cupcakes?error=ID inválido")
		return
	}

	var cupcake model.Cupcake
	if err := database.DB.First(&cupcake, uint(id)).Error; err != nil {
		c.Redirect(http.StatusSeeOther, "/lojista/cupcakes?error=Cupcake não encontrado")
		return
	}

	oldImagePath := cupcake.ImagemURL

	cupcake.Nome = c.PostForm("nome")
	cupcake.Descricao = c.PostForm("descricao")
	cupcake.Disponivel = c.PostForm("disponivel") == "true"
	precoStr := c.PostForm("preco")
	preco, err := strconv.ParseFloat(precoStr, 64)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/lojista/cupcakes?error=Preço inválido")
		return
	}
	cupcake.Preco = preco

	file, err := c.FormFile("imagem")
	if err == nil {
		ext := filepath.Ext(file.Filename)
		newFileName := uuid.New().String() + ext
		uploadPath := filepath.Join("uploads", newFileName)

		if err := c.SaveUploadedFile(file, uploadPath); err != nil {
			log.Printf("Erro ao salvar nova imagem: %v", err)
			c.Redirect(http.StatusSeeOther, "/lojista/cupcakes?error=Erro ao salvar imagem")
			return
		}

		cupcake.ImagemURL = "/" + uploadPath

		if oldImagePath != defaultCupcakeImage && oldImagePath != "" {
			fsOldPath := filepath.Clean(oldImagePath[1:])
			if err := os.Remove(fsOldPath); err != nil {
				log.Printf("AVISO: Não foi possível remover a imagem antiga '%s': %v", fsOldPath, err)
			} else {
				log.Printf("Imagem antiga '%s' removida com sucesso.", fsOldPath)
			}
		}

	} else if err != http.ErrMissingFile {
		log.Printf("Erro ao processar form-file (edição): %v", err)
		c.Redirect(http.StatusSeeOther, "/lojista/cupcakes?error=Erro ao processar arquivo de imagem")
		return
	}

	if err := database.DB.Save(&cupcake).Error; err != nil {
		log.Printf("Erro ao atualizar cupcake no DB: %v", err)
		c.Redirect(http.StatusSeeOther, "/lojista/cupcakes?error=Erro ao salvar no banco de dados")
		return
	}

	log.Println("Cupcake atualizado com sucesso.")
	c.Redirect(http.StatusSeeOther, "/lojista/cupcakes")
}

// DeleteCupcake remove um cupcake do banco de dados e o arquivo de imagem associado.
func (h *LojistaHandler) DeleteCupcake(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/lojista/cupcakes?error=ID inválido")
		return
	}

	var cupcake model.Cupcake
	if err := database.DB.First(&cupcake, uint(id)).Error; err != nil {
		c.Redirect(http.StatusSeeOther, "/lojista/cupcakes?error=Cupcake não encontrado")
		return
	}

	imagePath := cupcake.ImagemURL

	if err := database.DB.Delete(&cupcake).Error; err != nil {
		log.Printf("Erro ao deletar cupcake do DB: %v", err)
		c.Redirect(http.StatusSeeOther, "/lojista/cupcakes?error=Erro ao deletar do banco")
		return
	}

	log.Printf("Cupcake %d enviado para a lixeira (ou deletado) do DB.", id)

	if imagePath != defaultCupcakeImage && imagePath != "" {
		fsPath := filepath.Clean(imagePath[1:])
		if err := os.Remove(fsPath); err != nil {
			log.Printf("AVISO: Não foi possível remover o arquivo de imagem '%s': %v", fsPath, err)
		} else {
			log.Printf("Arquivo de imagem '%s' removido com sucesso.", fsPath)
		}
	}

	c.Redirect(http.StatusSeeOther, "/lojista/cupcakes")
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

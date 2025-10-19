package handler

import (
	"net/http"

	"github.com/ericoliveiras/meu-cupcake/internal/database"
	"github.com/ericoliveiras/meu-cupcake/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
)

type HomeHandler struct {
	Store *sessions.CookieStore
}

// getUserFromSession é uma função auxiliar para buscar os dados do usuário logado.
func (h *HomeHandler) getUserFromSession(c *gin.Context) (model.Usuario, bool) {
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

// ShowHomePage verifica se o usuário está logado e o redireciona para seu dashboard.
// Se não estiver logado, exibe a página inicial.
func (h *HomeHandler) ShowHomePage(c *gin.Context) {
	user, isLoggedIn := h.getUserFromSession(c)

	// --- LÓGICA DE REDIRECIONAMENTO AQUI ---
	if isLoggedIn {
		switch user.Tipo {
		case model.RoleCliente:
			c.Redirect(http.StatusFound, "/cliente/dashboard")
			return // Interrompe a execução para garantir que o redirect aconteça
		case model.RoleLojista:
			c.Redirect(http.StatusFound, "/lojista/dashboard")
			return // Interrompe a execução
		}
	}

	// Se não estiver logado, continua para renderizar a página inicial
	c.HTML(http.StatusOK, "index.html", gin.H{
		"IsLoggedIn": isLoggedIn,
		"User":       user,
	})
}

// ShowProfilePage renderiza a página de perfil apropriada para o usuário logado.
func (h *HomeHandler) ShowProfilePage(c *gin.Context) {
	userData, _ := c.Get("user")
	user := userData.(model.Usuario)

	switch user.Tipo {
	case model.RoleLojista:
		c.HTML(http.StatusOK, "lojista_profile.html", gin.H{
			"IsLoggedIn": true,
			"User":       user,
		})
	case model.RoleCliente:
		c.HTML(http.StatusOK, "cliente_profile.html", gin.H{
			"IsLoggedIn": true,
			"User":       user,
		})
	default:
		c.String(http.StatusInternalServerError, "Tipo de usuário desconhecido.")
	}
}

// ShowVitrinePage renderiza a página da vitrine de cupcakes.
func (h *HomeHandler) ShowVitrinePage(c *gin.Context) {
	var cupcakes []model.Cupcake
	if err := database.DB.Where("disponivel = ?", true).Order("created_at desc").Find(&cupcakes).Error; err != nil {
		c.String(http.StatusInternalServerError, "Não foi possível carregar a vitrine. Tente novamente mais tarde.")
		return
	}

	user, isLoggedIn := h.getUserFromSession(c)

	c.HTML(http.StatusOK, "vitrine.html", gin.H{
		"Cupcakes":   cupcakes,
		"IsLoggedIn": isLoggedIn,
		"User":       user,
		"ActivePage": "vitrine",
	})
}

// ShowClienteDashboard renderiza o painel principal do cliente.
func (h *HomeHandler) ShowClienteDashboard(c *gin.Context) {
	userData, _ := c.Get("user")
	user := userData.(model.Usuario)

	c.HTML(http.StatusOK, "cliente_dashboard.html", gin.H{
		"IsLoggedIn": true,
		"User":       user,
		"ActivePage": "dashboard",
	})
}
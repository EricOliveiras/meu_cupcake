// /internal/handler/home_handler.go
package handler

import (
	"fmt"
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

// getTotalCartQuantity é uma função auxiliar para somar as quantidades no carrinho.
func getTotalCartQuantity(session *sessions.Session) int {
	cartData := session.Values[CartSessionKey]
	cart, ok := cartData.(map[uint]int)
	if !ok {
		return 0
	}
	totalQuantity := 0
	for _, quantity := range cart {
		totalQuantity += quantity
	}
	return totalQuantity
}

// ShowHomePage renderiza a página inicial ou redireciona se logado.
func (h *HomeHandler) ShowHomePage(c *gin.Context) {
	user, isLoggedIn := h.getUserFromSession(c)

	if isLoggedIn {
		switch user.Tipo {
		case model.RoleCliente:
			c.Redirect(http.StatusFound, "/cliente/dashboard")
			return
		case model.RoleLojista:
			c.Redirect(http.StatusFound, "/lojista/dashboard")
			return
		}
	}

	session, _ := h.Store.Get(c.Request, "meu-cupcake-session")
	cartCount := getTotalCartQuantity(session)

	c.HTML(http.StatusOK, "index.html", gin.H{
		"IsLoggedIn":    isLoggedIn,
		"User":          user,
		"CartItemCount": cartCount,
	})
}

// ShowProfilePage renderiza a página de perfil apropriada.
func (h *HomeHandler) ShowProfilePage(c *gin.Context) {
	userData, _ := c.Get("user")
	user := userData.(model.Usuario)
	session, _ := h.Store.Get(c.Request, "meu-cupcake-session")
	cartCount := getTotalCartQuantity(session)

	switch user.Tipo {
	case model.RoleLojista:
		c.HTML(http.StatusOK, "lojista_profile.html", gin.H{
			"IsLoggedIn":    true,
			"User":          user,
			"CartItemCount": cartCount,
		})
	case model.RoleCliente:
		c.HTML(http.StatusOK, "cliente_profile.html", gin.H{
			"IsLoggedIn":    true,
			"User":          user,
			"CartItemCount": cartCount,
		})
	default:
		c.String(http.StatusInternalServerError, "Tipo de usuário desconhecido.")
	}
}

func (h *HomeHandler) ShowVitrinePage(c *gin.Context) {
	var cupcakes []model.Cupcake
	if err := database.DB.Where("disponivel = ?", true).Order("created_at desc").Find(&cupcakes).Error; err != nil {
		c.String(http.StatusInternalServerError, "Não foi possível carregar a vitrine.")
		return
	}

	user, isLoggedIn := h.getUserFromSession(c)
	session, _ := h.Store.Get(c.Request, "meu-cupcake-session")
	cartCount := getTotalCartQuantity(session)

	flashesSuccess := session.Flashes("success")

	err := session.Save(c.Request, c.Writer)
	if err != nil {
		fmt.Printf("AVISO: Erro ao salvar sessão após ler flashes em ShowVitrinePage: %v\n", err)
	}
	// ---------------------------------------------

	c.HTML(http.StatusOK, "vitrine.html", gin.H{
		"Cupcakes":      cupcakes,
		"IsLoggedIn":    isLoggedIn,
		"User":          user,
		"ActivePage":    "vitrine",
		"CartItemCount": cartCount,
		"Flashes":       flashesSuccess,
	})
}

// ShowClienteDashboard renderiza o painel principal do cliente.
func (h *HomeHandler) ShowClienteDashboard(c *gin.Context) {
	// Pega o usuário do contexto (já validado pelo middleware)
	userData, _ := c.Get("user")
	user := userData.(model.Usuario)

	// Pega a sessão para calcular a quantidade no carrinho
	session, _ := h.Store.Get(c.Request, "meu-cupcake-session")
	cartCount := getTotalCartQuantity(session)

	c.HTML(http.StatusOK, "cliente_dashboard.html", gin.H{
		"IsLoggedIn":    true,
		"User":          user,
		"ActivePage":    "dashboard",
		"CartItemCount": cartCount,
	})
}

func (h *HomeHandler) ShowPagamentoSucessoPage(c *gin.Context) {
	user, isLoggedIn := h.getUserFromSession(c)
	session, _ := h.Store.Get(c.Request, "meu-cupcake-session")
	cartCount := getTotalCartQuantity(session)

	c.HTML(http.StatusOK, "pagamento_sucesso.html", gin.H{
		"IsLoggedIn":    isLoggedIn,
		"User":          user,
		"CartItemCount": cartCount,
	})
}

func (h *HomeHandler) ShowClientePedidosPage(c *gin.Context) {
	userData, _ := c.Get("user")
	user := userData.(model.Usuario)

	session, _ := h.Store.Get(c.Request, "meu-cupcake-session")
	cartCount := getTotalCartQuantity(session)

	var pedidos []model.Order
	err := database.DB.Preload("Items.Cupcake").
		Where("usuario_id = ?", user.ID).
		Order("created_at desc").
		Find(&pedidos).Error

	if err != nil {
		fmt.Printf("Erro ao buscar pedidos do cliente %d: %v\n", user.ID, err)
		c.HTML(http.StatusOK, "cliente_pedidos.html", gin.H{
			"IsLoggedIn":    true,
			"User":          user,
			"CartItemCount": cartCount,
			"Pedidos":       []model.Order{}, // Lista vazia
			"ErrorMsg":      "Erro ao carregar histórico de pedidos.",
		})
		return
	}

	c.HTML(http.StatusOK, "cliente_pedidos.html", gin.H{
		"IsLoggedIn":    true,
		"User":          user,
		"CartItemCount": cartCount,
		"Pedidos":       pedidos,
	})
}

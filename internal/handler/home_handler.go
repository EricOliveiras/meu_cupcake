// /internal/handler/home_handler.go
package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/ericoliveiras/meu-cupcake/internal/database"
	"github.com/ericoliveiras/meu-cupcake/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/mercadopago/sdk-go/pkg/config"
	"github.com/mercadopago/sdk-go/pkg/payment"
	"gorm.io/gorm"
)

type HomeHandler struct {
	Store *sessions.CookieStore
	MPCfg *config.Config
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
	cartData := session.Values[CartSessionKey]
	cart, _ := cartData.(map[uint]int)
	cartCount := getTotalCartQuantityHelper(cart)

	flashesSuccess := session.Flashes("success")
	flashesError := session.Flashes("error")
	err := session.Save(c.Request, c.Writer)
	if err != nil {
		fmt.Printf("AVISO: Erro ao salvar sessão em ShowProfilePage: %v\n", err)
	}

	data := gin.H{
		"IsLoggedIn":     true,
		"User":           user,
		"CartItemCount":  cartCount,
		"FlashesSuccess": flashesSuccess,
		"FlashesError":   flashesError,
	}

	switch user.Tipo {
	case model.RoleLojista:
		c.HTML(http.StatusOK, "lojista_profile.html", data)
	case model.RoleCliente:
		c.HTML(http.StatusOK, "cliente_profile.html", data)
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

func (h *HomeHandler) ShowPedidoPagamentoPage(c *gin.Context) {
	user, _ := c.Get("user")
	usuario := user.(model.Usuario)

	// Pega o ID do pedido da URL
	pedidoIDStr := c.Param("id")
	pedidoID, err := strconv.ParseUint(pedidoIDStr, 10, 32)
	if err != nil {
		c.String(http.StatusNotFound, "Página não encontrada.")
		return
	}

	// Busca o pedido no DB, garantindo que ele pertence ao usuário logado
	var pedido model.Order
	err = database.DB.Where("id = ? AND usuario_id = ?", pedidoID, usuario.ID).First(&pedido).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.String(http.StatusNotFound, "Pedido não encontrado.")
			return
		}
		c.String(http.StatusInternalServerError, "Erro ao buscar pedido.")
		return
	}

	// Se o pedido não for PIX ou não estiver pendente, não há o que mostrar
	if pedido.Status != model.StatusPendente || pedido.MetodoPagamento != "pix" || pedido.PagamentoMPID == nil {
		c.String(http.StatusBadRequest, "Este pagamento não está pendente ou não é PIX.")
		return
	}

	// --- BUSCA OS DADOS DO PIX NO MERCADO PAGO ---
	fmt.Printf("Buscando dados do pagamento MP ID: %d\n", *pedido.PagamentoMPID)
	client := payment.NewClient(h.MPCfg)
	resource, err := client.Get(context.Background(), int(*pedido.PagamentoMPID))

	if err != nil {
		fmt.Printf("Erro ao buscar pagamento no MP: %v\n", err)
		c.String(http.StatusInternalServerError, "Erro ao buscar dados do pagamento.")
		return
	}

	// Verifica se o pagamento ainda está pendente no MP
	if resource.Status == "pending" {
		session, _ := h.Store.Get(c.Request, "meu-cupcake-session")
		cartData := session.Values[CartSessionKey]
		cart, _ := cartData.(map[uint]int)
		cartCount := getTotalCartQuantityHelper(cart)

		c.HTML(http.StatusOK, "pagamento_pix.html", gin.H{
			"IsLoggedIn":       true,
			"User":             usuario,
			"Pedido":           pedido,
			"QrCodeBase64":     resource.PointOfInteraction.TransactionData.QRCodeBase64,
			"QrCodeCopiaECola": resource.PointOfInteraction.TransactionData.QRCode,
			"Total":            pedido.Total,
			"CartItemCount":    cartCount,
		})
	} else {
		// O pagamento não está mais pendente (foi pago ou expirou)
		// Atualiza nosso banco (caso o webhook tenha falhado)
		switch resource.Status {
		case "approved":
			database.DB.Model(&pedido).Update("status", model.StatusPago)
		case "cancelled", "expired":
			database.DB.Model(&pedido).Update("status", model.StatusFalhou) // Ou "expirado"
		}
		// Redireciona de volta para o histórico de pedidos
		session, _ := h.Store.Get(c.Request, "meu-cupcake-session")
		session.AddFlash("O status deste pagamento mudou. Verifique seu histórico.", "success")
		session.Save(c.Request, c.Writer)
		c.Redirect(http.StatusFound, "/cliente/pedidos")
	}
}

// ShowEditProfilePage exibe o formulário de edição de perfil.
func (h *HomeHandler) ShowEditProfilePage(c *gin.Context) {
	userData, _ := c.Get("user")
	user := userData.(model.Usuario)

	session, _ := h.Store.Get(c.Request, "meu-cupcake-session")
	cartData := session.Values[CartSessionKey]
	cart, _ := cartData.(map[uint]int)
	cartCount := getTotalCartQuantityHelper(cart)

	c.HTML(http.StatusOK, "perfil_editar.html", gin.H{
		"IsLoggedIn":     true,
		"User":           user,
		"CartItemCount":  cartCount,
		"FlashesSuccess": session.Flashes("success"),
		"FlashesError":   session.Flashes("error"),
	})
	session.Save(c.Request, c.Writer)
}

// ProcessEditProfileForm processa a atualização do perfil.
func (h *HomeHandler) ProcessEditProfileForm(c *gin.Context) {
	userData, _ := c.Get("user")
	user := userData.(model.Usuario)
	session, _ := h.Store.Get(c.Request, "meu-cupcake-session")

	// Pega TODOS os dados do formulário
	novoNome := c.PostForm("nome")
	novoEmail := c.PostForm("email")
	novoTelefone := c.PostForm("telefone")
	novoCEP := c.PostForm("cep")
	novoRua := c.PostForm("rua")
	novoNumero := c.PostForm("numero")
	novoComplemento := c.PostForm("complemento")
	novoBairro := c.PostForm("bairro")
	novoCidade := c.PostForm("cidade")
	novoEstado := c.PostForm("estado")

	// Validação básica
	if novoNome == "" || novoEmail == "" {
		session.AddFlash("Nome e E-mail são obrigatórios.", "error")
		session.Save(c.Request, c.Writer)
		c.Redirect(http.StatusFound, "/perfil/editar")
		return
	}

	// Validação de E-mail (se foi alterado)
	if novoEmail != user.Email {
		var existingUser model.Usuario
		if err := database.DB.Where("email = ?", novoEmail).First(&existingUser).Error; err == nil {
			session.AddFlash("O e-mail informado já está em uso por outra conta.", "error")
			session.Save(c.Request, c.Writer)
			c.Redirect(http.StatusFound, "/perfil/editar")
			return
		}
	}

	// Atualiza os dados no banco
	// Usamos .Updates() que só atualiza campos não-nulos/não-zero
	// Para garantir que campos em branco (ex: complemento) sejam salvos, usamos Select()
	// Ou podemos usar um map[string]interface{}

	updateData := map[string]interface{}{
		"Nome":        novoNome,
		"Email":       novoEmail,
		"Telefone":    novoTelefone,
		"CEP":         novoCEP,
		"Rua":         novoRua,
		"Numero":      novoNumero,
		"Complemento": novoComplemento,
		"Bairro":      novoBairro,
		"Cidade":      novoCidade,
		"Estado":      novoEstado,
	}

	result := database.DB.Model(&user).Updates(updateData)

	if result.Error != nil {
		log.Printf("Erro ao atualizar perfil do usuário %d: %v\n", user.ID, result.Error)
		session.AddFlash("Erro ao salvar as alterações. Tente novamente.", "error")
		session.Save(c.Request, c.Writer)
		c.Redirect(http.StatusFound, "/perfil/editar")
		return
	}

	log.Printf("Perfil do usuário %d atualizado.\n", user.ID)
	session.AddFlash("Perfil atualizado com sucesso!", "success")
	session.Save(c.Request, c.Writer)

	c.Redirect(http.StatusFound, "/perfil")
}

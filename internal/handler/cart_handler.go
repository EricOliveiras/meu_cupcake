package handler

import (
	"context"
	"errors"
	"fmt" // Import log
	"net/http"
	"os"
	"sort"
	"strconv" // Import strings
	"time"

	"github.com/ericoliveiras/meu-cupcake/internal/database"
	"github.com/ericoliveiras/meu-cupcake/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/mercadopago/sdk-go/pkg/config"
	"github.com/mercadopago/sdk-go/pkg/payment"
	"gorm.io/gorm"
)

// PaymentRequestData espelha a estrutura do JSON enviado pelo frontend (CARTÃO).
type PaymentRequestData struct {
	Token             string  `json:"token"`
	IssuerID          string  `json:"issuer_id"`
	PaymentMethodID   string  `json:"payment_method_id"`
	TransactionAmount float64 `json:"transaction_amount"`
	Installments      int     `json:"installments"`
	Description       string  `json:"description"`
	Payer             struct {
		Email          string `json:"email"`
		Identification struct {
			Type   string `json:"type"`
			Number string `json:"number"`
		} `json:"identification"`
	} `json:"payer"`
}

// PixRequestData espelha a estrutura do JSON enviado pelo frontend (PIX).
type PixRequestData struct {
	TransactionAmount float64 `json:"transaction_amount"`
	Description       string  `json:"description"`
	Payer             struct {
		Email string `json:"email"`
		// (Campos de identificação removidos para o teste do PIX)
	} `json:"payer"`
}

// Estrutura auxiliar para passar dados do item do carrinho para o template
type CartItemView struct {
	Cupcake  model.Cupcake
	Quantity int
	Subtotal float64
}

// CartHandler agrupa os handlers do carrinho.
type CartHandler struct {
	Store *sessions.CookieStore
	MPCfg *config.Config
}

const CartSessionKey = "shopping_cart"

// AddToCart adiciona um item ao carrinho e retorna JSON (sem recarregar a página)
func (h *CartHandler) AddToCart(c *gin.Context) {
	idStr := c.Param("id")
	id64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "ID do cupcake inválido."})
		return
	}
	cupcakeID := uint(id64)

	var cupcake model.Cupcake
	if err := database.DB.Where("id = ? AND disponivel = ?", cupcakeID, true).First(&cupcake).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "Cupcake não encontrado ou indisponível."})
		return
	}

	session, _ := h.Store.Get(c.Request, "meu-cupcake-session")
	cartData := session.Values[CartSessionKey]

	cart, ok := cartData.(map[uint]int)
	if !ok {
		cart = make(map[uint]int)
	}

	cart[cupcakeID]++

	session.Values[CartSessionKey] = cart
	if err := session.Save(c.Request, c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Erro ao salvar o carrinho."})
		return
	}

	newTotalQuantity := getTotalCartQuantityHelper(cart)

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"message":      "Item adicionado com sucesso!",
		"newCartCount": newTotalQuantity,
	})
}

// ShowCartPage exibe o conteúdo do carrinho de compras.
func (h *CartHandler) ShowCartPage(c *gin.Context) {
	session, _ := h.Store.Get(c.Request, "meu-cupcake-session")
	cartData := session.Values[CartSessionKey]
	user, isLoggedIn := h.getUserFromSession(c)

	cart, ok := cartData.(map[uint]int)
	if !ok || len(cart) == 0 {
		flashesSuccess := session.Flashes("success")
		flashesError := session.Flashes("error")
		session.Save(c.Request, c.Writer)
		c.HTML(http.StatusOK, "carrinho.html", gin.H{
			"Items":          []CartItemView{},
			"Total":          0.0,
			"IsLoggedIn":     isLoggedIn,
			"User":           user,
			"CartItemCount":  0,
			"FlashesSuccess": flashesSuccess,
			"FlashesError":   flashesError,
		})
		return
	}

	cupcakeIDs := make([]uint, 0, len(cart))
	for id := range cart {
		cupcakeIDs = append(cupcakeIDs, id)
	}

	var cupcakes []model.Cupcake
	database.DB.Where("id IN ? AND disponivel = ?", cupcakeIDs, true).Find(&cupcakes)

	var total float64
	cartItemsView := make([]CartItemView, 0, len(cupcakes))
	cupcakeMap := make(map[uint]model.Cupcake)
	for _, cp := range cupcakes {
		cupcakeMap[cp.ID] = cp
	}

	finalCart := make(map[uint]int)
	for id, quantity := range cart {
		if cupcake, found := cupcakeMap[id]; found {
			subtotal := cupcake.Preco * float64(quantity)
			cartItemsView = append(cartItemsView, CartItemView{
				Cupcake: cupcake, Quantity: quantity, Subtotal: subtotal,
			})
			total += subtotal
			finalCart[id] = quantity
		}
	}

	sort.Slice(cartItemsView, func(i, j int) bool {
		return cartItemsView[i].Cupcake.Nome < cartItemsView[j].Cupcake.Nome
	})

	cartCount := getTotalCartQuantityHelper(finalCart)

	flashesSuccess := session.Flashes("success")
	flashesError := session.Flashes("error")
	session.Save(c.Request, c.Writer)

	c.HTML(http.StatusOK, "carrinho.html", gin.H{
		"Items":          cartItemsView,
		"Total":          total,
		"IsLoggedIn":     isLoggedIn,
		"User":           user,
		"CartItemCount":  cartCount,
		"FlashesSuccess": flashesSuccess,
		"FlashesError":   flashesError,
	})
}

func (h *CartHandler) RemoveFromCart(c *gin.Context) {
	idStr := c.Param("id")
	id64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "ID inválido."})
		return
	}
	cupcakeID := uint(id64)
	session, _ := h.Store.Get(c.Request, "meu-cupcake-session")
	cartData := session.Values[CartSessionKey]
	cart, ok := cartData.(map[uint]int)
	if !ok || len(cart) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Carrinho já vazio.", "newCartCount": 0})
		return
	}

	delete(cart, cupcakeID) // Remove o item
	session.Values[CartSessionKey] = cart
	if err := session.Save(c.Request, c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Erro ao atualizar o carrinho."})
		return
	}

	// --- 1. CALCULA E RETORNA A NOVA CONTAGEM ---
	newTotalQuantity := getTotalCartQuantityHelper(cart)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Item removido.", "newCartCount": newTotalQuantity})
}

// DecreaseQuantity diminui a quantidade de um item no carrinho.
func (h *CartHandler) DecreaseQuantity(c *gin.Context) {
	idStr := c.Param("id")
	id64, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "ID inválido."})
		return
	}
	cupcakeID := uint(id64)
	session, _ := h.Store.Get(c.Request, "meu-cupcake-session")
	cartData := session.Values[CartSessionKey]
	cart, ok := cartData.(map[uint]int)
	if !ok || len(cart) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Carrinho já vazio.", "newCartCount": 0})
		return
	}

	if quantity, exists := cart[cupcakeID]; exists {
		if quantity > 1 {
			cart[cupcakeID]--
		} else {
			delete(cart, cupcakeID)
		}
		session.Values[CartSessionKey] = cart
		if err := session.Save(c.Request, c.Writer); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Erro ao atualizar o carrinho."})
			return
		}
	}

	// --- 2. CALCULA E RETORNA A NOVA CONTAGEM ---
	newTotalQuantity := getTotalCartQuantityHelper(cart)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Quantidade atualizada.", "newCartCount": newTotalQuantity})
}

// ClearCart remove todos os itens do carrinho.
func (h *CartHandler) ClearCart(c *gin.Context) {
	session, _ := h.Store.Get(c.Request, "meu-cupcake-session")
	session.Values[CartSessionKey] = make(map[uint]int) // Define como vazio
	if err := session.Save(c.Request, c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Erro ao limpar o carrinho."})
		return
	}

	// --- 3. RETORNA A NOVA CONTAGEM (SEMPRE 0) ---
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Carrinho esvaziado.", "newCartCount": 0})
}

// ShowCheckoutPage exibe a página de resumo do pedido antes do pagamento.
func (h *CartHandler) ShowCheckoutPage(c *gin.Context) {
	session, _ := h.Store.Get(c.Request, "meu-cupcake-session")
	cartData := session.Values[CartSessionKey]
	userData, _ := c.Get("user")
	user := userData.(model.Usuario)

	cart, ok := cartData.(map[uint]int)
	if !ok || len(cart) == 0 {
		c.Redirect(http.StatusFound, "/carrinho")
		return
	}

	cupcakeIDs := make([]uint, 0, len(cart))
	for id := range cart {
		cupcakeIDs = append(cupcakeIDs, id)
	}

	var cupcakes []model.Cupcake
	result := database.DB.Where("id IN ? AND disponivel = ?", cupcakeIDs, true).Find(&cupcakes)
	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		c.String(http.StatusInternalServerError, "Erro ao buscar detalhes dos produtos.")
		return
	}

	if len(cupcakes) != len(cart) {
		fmt.Printf("Checkout inválido: Itens na sessão (%d) != Itens válidos no DB (%d)\n", len(cart), len(cupcakes))
		session.AddFlash("Alguns itens no seu carrinho não estão mais disponíveis. Verifique seu carrinho.", "error")
		session.Save(c.Request, c.Writer)
		c.Redirect(http.StatusFound, "/carrinho")
		return
	}

	var total float64
	cartItemsView := make([]CartItemView, 0, len(cupcakes))
	cupcakeMap := make(map[uint]model.Cupcake)
	for _, cp := range cupcakes {
		cupcakeMap[cp.ID] = cp
	}
	finalCart := make(map[uint]int)
	for id, quantity := range cart {
		if cupcake, found := cupcakeMap[id]; found {
			subtotal := cupcake.Preco * float64(quantity)
			cartItemsView = append(cartItemsView, CartItemView{
				Cupcake: cupcake, Quantity: quantity, Subtotal: subtotal,
			})
			total += subtotal
			finalCart[id] = quantity
		}
	}
	cartCount := getTotalCartQuantityHelper(finalCart)

	mpPublicKey := os.Getenv("MP_PUBLIC_KEY")
	if mpPublicKey == "" {
		fmt.Println("AVISO: MP_PUBLIC_KEY não encontrada no .env")
	}

	c.HTML(http.StatusOK, "checkout.html", gin.H{
		"Items":                cartItemsView,
		"Total":                total,
		"IsLoggedIn":           true,
		"User":                 user,
		"CartItemCount":        cartCount,
		"MercadoPagoPublicKey": mpPublicKey,
	})
}

// ProcessPayment (Pagamento com Cartão)
func (h *CartHandler) ProcessPayment(c *gin.Context) {
	if h.MPCfg == nil {
		fmt.Println("FATAL: Configuração do Mercado Pago é nula (nil) no handler.")
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Erro interno: config pagamento."})
		return
	}

	var reqData PaymentRequestData
	if err := c.ShouldBindJSON(&reqData); err != nil {
		fmt.Printf("Erro Bind JSON: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos.", "details": err.Error()})
		return
	}
	fmt.Printf("Dados Pagamento Recebidos: %+v\n", reqData)

	userData, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuário não autenticado."})
		return
	}
	user := userData.(model.Usuario)

	session, _ := h.Store.Get(c.Request, "meu-cupcake-session")
	cartData := session.Values[CartSessionKey]
	cart, ok := cartData.(map[uint]int)
	if !ok || len(cart) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Carrinho vazio ou inválido."})
		return
	}

	// --- LÓGICA DE VALIDAÇÃO DO CARRINHO ---
	var currentTotal float64
	cupcakeIDs := make([]uint, 0, len(cart))
	for id := range cart {
		cupcakeIDs = append(cupcakeIDs, id)
	}

	var cupcakesFromDB []model.Cupcake
	result := database.DB.Where("id IN ? AND disponivel = ?", cupcakeIDs, true).Find(&cupcakesFromDB)
	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		fmt.Printf("Erro DB ao buscar cupcakes: %v\n", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao verificar produtos."})
		return
	}

	cupcakeMap := make(map[uint]model.Cupcake)
	for _, cp := range cupcakesFromDB {
		cupcakeMap[cp.ID] = cp
	}

	validItems := make([]CartItemView, 0, len(cart))
	validItemCount := 0
	for id, quantity := range cart {
		cupcake, found := cupcakeMap[id]
		if found {
			subtotal := cupcake.Preco * float64(quantity)
			validItems = append(validItems, CartItemView{
				Cupcake: cupcake, Quantity: quantity, Subtotal: subtotal,
			})
			currentTotal += subtotal
			validItemCount++
		}
	}

	if validItemCount != len(cart) || len(validItems) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Um ou mais itens no seu carrinho não estão mais disponíveis."})
		return
	}
	// --------------------------------------------------------

	// --- Validação de Segurança do Total ---
	tolerance := 0.01
	if (currentTotal-reqData.TransactionAmount) > tolerance || (reqData.TransactionAmount-currentTotal) > tolerance {
		fmt.Printf("ALERTA SEGURANÇA: Total Backend (%.2f) != Total Frontend (%.2f)\n", currentTotal, reqData.TransactionAmount)
		c.JSON(http.StatusBadRequest, gin.H{"error": "O valor total do pedido foi modificado."})
		return
	}
	fmt.Printf("Validação Total OK: Backend=%.2f, Frontend=%.2f\n", currentTotal, reqData.TransactionAmount)

	// --- Criação do Pedido no DB (Transação) ---
	var pedidoCriado model.Order // Corrigido para model.Pedido
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		externalRef := fmt.Sprintf("pedido_%d_%d", user.ID, time.Now().UnixNano())
		pedido := model.Order{ // Corrigido para model.Pedido
			UsuarioID: user.ID, Status: model.StatusPendente, Total: currentTotal,
			MetodoPagamento: reqData.PaymentMethodID, Parcelas: reqData.Installments, ExternalReference: externalRef,
		}
		if err := tx.Create(&pedido).Error; err != nil {
			return errors.New("erro ao criar o cabeçalho do pedido")
		}
		pedidoCriado = pedido
		for _, item := range validItems {
			itemPedido := model.ItemOrder{ // Corrigido para model.ItemPedido
				PedidoID: pedido.ID, CupcakeID: item.Cupcake.ID, Quantidade: item.Quantity,
				PrecoUnitario: item.Cupcake.Preco, Subtotal: item.Subtotal,
			}
			if err := tx.Create(&itemPedido).Error; err != nil {
				return errors.New("erro ao salvar os itens do pedido")
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Não foi possível registrar seu pedido.", "details": err.Error()})
		return
	}
	fmt.Printf("Pedido %d criado no DB (Ref: %s)\n", pedidoCriado.ID, pedidoCriado.ExternalReference)

	// --- Chamada à API do Mercado Pago ---
	fmt.Println("Tentando criar pagamento MP...")
	client := payment.NewClient(h.MPCfg)
	request := payment.Request{
		TransactionAmount: currentTotal, // USA O VALOR DO BACKEND
		Token:             reqData.Token,
		Description:       reqData.Description,
		Installments:      reqData.Installments,
		PaymentMethodID:   reqData.PaymentMethodID,
		IssuerID:          reqData.IssuerID,
		ExternalReference: pedidoCriado.ExternalReference,
		Payer: &payment.PayerRequest{
			Email: reqData.Payer.Email,
			Identification: &payment.IdentificationRequest{
				Type:   reqData.Payer.Identification.Type,
				Number: reqData.Payer.Identification.Number,
			},
		},
		// notification_url: "SUA_URL_WEBHOOK"
	}

	// --- CORREÇÃO AQUI: REMOVIDO 'var resource ...' ---
	resource, err := client.Create(context.Background(), request) // Usa := aqui

	// --- Tratamento da Resposta e Atualização do Pedido no DB ---
	var finalPedidoStatus model.StatusOrder = model.StatusFalhou // Corrigido
	var responseStatus string = "rejected"
	var message string = "Pagamento recusado ou pendente."
	var mpPaymentID *int64 = nil

	if err != nil {
		fmt.Printf("Erro MP: %v\n", err)
		message = "Erro ao processar pagamento com o provedor."
	} else {
		fmt.Printf("Resposta MP: Status=%s, Detail=%s, ID=%d\n", resource.Status, resource.StatusDetail, resource.ID)
		tempID := int64(resource.ID)
		mpPaymentID = &tempID
		switch resource.Status {
		case "approved":
			finalPedidoStatus = model.StatusPago // Corrigido
			responseStatus = "approved"
			message = "Pagamento aprovado!"
			session.Values[CartSessionKey] = make(map[uint]int)
			session.Save(c.Request, c.Writer)
		case "in_process", "pending":
			finalPedidoStatus = model.StatusPendente // Corrigido
			responseStatus = "pending"
			message = "Pagamento pendente."
		default:
			finalPedidoStatus = model.StatusFalhou // Corrigido
			responseStatus = "rejected"
			message = fmt.Sprintf("Pagamento não aprovado (%s).", resource.StatusDetail)
		}
	}

	updateResult := database.DB.Model(&pedidoCriado).Updates(model.Order{ // Corrigido
		Status: finalPedidoStatus, PagamentoMPID: mpPaymentID,
	})
	if updateResult.Error != nil {
		fmt.Printf("ERRO CRÍTICO DB UPDATE Pedido %d: %v\n", pedidoCriado.ID, updateResult.Error)
	}

	c.JSON(http.StatusOK, gin.H{"status": responseStatus, "message": message, "paymentId": mpPaymentID})
}

// ProcessPixPayment recebe os dados do pagador, cria o pedido e gera um pagamento PIX.
func (h *CartHandler) ProcessPixPayment(c *gin.Context) {
	if h.MPCfg == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Configuração de pagamento indisponível."})
		return
	}

	// 1. Usa a nova struct PixRequestData
	var pixReqData PixRequestData
	if err := c.ShouldBindJSON(&pixReqData); err != nil {
		fmt.Printf("Erro Bind JSON PIX: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados do pagador inválidos."})
		return
	}
	fmt.Printf("Dados Pagamento PIX Recebidos: %+v\n", pixReqData)

	userData, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuário não autenticado."})
		return
	}
	user := userData.(model.Usuario)

	session, _ := h.Store.Get(c.Request, "meu-cupcake-session")
	cartData := session.Values[CartSessionKey]
	cart, ok := cartData.(map[uint]int)
	if !ok || len(cart) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Carrinho vazio."})
		return
	}

	// --- LÓGICA DE RECÁLCULO (COPIADA DO ProcessPayment) ---
	var currentTotal float64
	cupcakeIDs := make([]uint, 0, len(cart))
	for id := range cart {
		cupcakeIDs = append(cupcakeIDs, id)
	}

	var cupcakesFromDB []model.Cupcake
	result := database.DB.Where("id IN ? AND disponivel = ?", cupcakeIDs, true).Find(&cupcakesFromDB)
	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		fmt.Printf("Erro DB ao buscar cupcakes: %v\n", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao verificar produtos."})
		return
	}

	cupcakeMap := make(map[uint]model.Cupcake)
	for _, cp := range cupcakesFromDB {
		cupcakeMap[cp.ID] = cp
	}

	validItems := make([]CartItemView, 0, len(cart)) // 'validItems' DECLARADA E PREENCHIDA
	validItemCount := 0
	for id, quantity := range cart {
		cupcake, found := cupcakeMap[id]
		if found {
			subtotal := cupcake.Preco * float64(quantity)
			validItems = append(validItems, CartItemView{
				Cupcake: cupcake, Quantity: quantity, Subtotal: subtotal,
			})
			currentTotal += subtotal
			validItemCount++
		}
	}

	if validItemCount != len(cart) || len(validItems) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Um ou mais itens no seu carrinho não estão mais disponíveis."})
		return
	}

	tolerance := 0.01
	if (currentTotal-pixReqData.TransactionAmount) > tolerance || (pixReqData.TransactionAmount-currentTotal) > tolerance {
		fmt.Printf("ALERTA SEGURANÇA (PIX): Total Backend (%.2f) != Total Frontend (%.2f)\n", currentTotal, pixReqData.TransactionAmount)
		c.JSON(http.StatusBadRequest, gin.H{"error": "O valor total do pedido foi modificado."})
		return
	}
	fmt.Printf("Validação Total PIX OK: Backend=%.2f, Frontend=%.2f\n", currentTotal, pixReqData.TransactionAmount)
	// --- Fim da lógica de validação ---

	// 4. CRIAR PEDIDO E ITENS NO BANCO DE DADOS (Status Pendente)
	var pedidoCriado model.Order // Corrigido
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		externalRef := fmt.Sprintf("pedido_%d_%d", user.ID, time.Now().UnixNano())
		pedido := model.Order{ // Corrigido
			UsuarioID:         user.ID,
			Status:            model.StatusPendente,
			Total:             currentTotal,
			MetodoPagamento:   "pix",
			Parcelas:          1,
			ExternalReference: externalRef,
		}
		if err := tx.Create(&pedido).Error; err != nil {
			return err
		}
		pedidoCriado = pedido

		for _, item := range validItems {
			itemPedido := model.ItemOrder{ // Corrigido
				PedidoID:      pedido.ID,
				CupcakeID:     item.Cupcake.ID,
				Quantidade:    item.Quantity,
				PrecoUnitario: item.Cupcake.Preco,
				Subtotal:      item.Subtotal,
			}
			if err := tx.Create(&itemPedido).Error; err != nil {
				fmt.Printf("Erro ao criar item %d do pedido %d no DB: %v\n", item.Cupcake.ID, pedido.ID, err)
				return errors.New("erro ao salvar os itens do pedido")
			}
		}

		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao registrar o pedido no banco."})
		return
	}
	fmt.Printf("Pedido PIX %d criado no DB (Ref: %s)\n", pedidoCriado.ID, pedidoCriado.ExternalReference)

	// 5. CHAMAR A API DO MERCADO PAGO PARA GERAR O PIX
	fmt.Println("Tentando criar pagamento PIX via API Mercado Pago...")
	client := payment.NewClient(h.MPCfg)

	request := payment.Request{
		TransactionAmount: currentTotal,
		Description:       pixReqData.Description,
		PaymentMethodID:   "pix",
		ExternalReference: pedidoCriado.ExternalReference,
		Payer: &payment.PayerRequest{
			Email: pixReqData.Payer.Email, // Envia SÓ o email
			// Bloco de identificação removido
			FirstName: user.Nome,
		},
		// notification_url: "SUA_URL_WEBHOOK", // IMPORTANTE!
	}

	// --- CORREÇÃO AQUI: REMOVIDO 'var resource ...' ---
	resource, err := client.Create(context.Background(), request) // Usa :=

	// 6. TRATAR RESPOSTA E ENVIAR QR CODE PARA O FRONTEND
	if err != nil {
		fmt.Printf("Erro ao criar PIX no MP: %v\n", err)
		database.DB.Model(&pedidoCriado).Update("status", model.StatusFalhou)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao gerar PIX com o provedor."})
		return
	}

	if resource.Status == "pending" {
		fmt.Println("Pagamento PIX gerado com sucesso, aguardando pagamento.")

		mpPaymentID := int64(resource.ID)
		database.DB.Model(&pedidoCriado).Update("PagamentoMPID", mpPaymentID)

		c.JSON(http.StatusOK, gin.H{
			"status":         "pending",
			"payment_id":     resource.ID,
			"qr_code_base64": resource.PointOfInteraction.TransactionData.QRCodeBase64,
			"qr_code":        resource.PointOfInteraction.TransactionData.QRCode,
		})
	} else {
		fmt.Printf("Status inesperado ao gerar PIX: %s\n", resource.Status)
		database.DB.Model(&pedidoCriado).Update("status", model.StatusFalhou)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Status inesperado do provedor de pagamento."})
	}
}

// --- Funções Auxiliares ---

func (h *CartHandler) getUserFromSession(c *gin.Context) (model.Usuario, bool) {
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

func getTotalCartQuantityHelper(cart map[uint]int) int {
	if cart == nil {
		return 0
	}
	totalQuantity := 0
	for _, quantity := range cart {
		totalQuantity += quantity
	}
	return totalQuantity
}

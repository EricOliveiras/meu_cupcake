package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/ericoliveiras/meu-cupcake/internal/database"
	"github.com/ericoliveiras/meu-cupcake/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthHandler struct {
	Store *sessions.CookieStore
}

// ShowCadastroPage renderiza a página de cadastro e exibe flash messages.
func (h *AuthHandler) ShowCadastroPage(c *gin.Context) {
	session, _ := h.Store.Get(c.Request, "meu-cupcake-session")
	flashesSuccess := session.Flashes("success")
	flashesError := session.Flashes("error")
	err := session.Save(c.Request, c.Writer)
	if err != nil {
		fmt.Printf("AVISO: Erro ao salvar sessão em ShowCadastroPage: %v\n", err)
	}

	c.HTML(http.StatusOK, "cadastro.html", gin.H{
		"IsLoggedIn":     false,
		"FlashesSuccess": flashesSuccess,
		"FlashesError":   flashesError,
	})
}

// ProcessCadastroForm processa os dados submetidos pelo formulário de cadastro.
func (h *AuthHandler) ProcessCadastroForm(c *gin.Context) {
	session, _ := h.Store.Get(c.Request, "meu-cupcake-session")
	nome := c.PostForm("nome")
	email := c.PostForm("email")
	senha := c.PostForm("senha")
	confirmarSenha := c.PostForm("confirmar_senha")

	if senha != confirmarSenha {
		session.AddFlash("As senhas não conferem!", "error")
		session.Save(c.Request, c.Writer)
		c.Redirect(http.StatusFound, "/cadastro")
		return
	}

	senhaHash, err := bcrypt.GenerateFromPassword([]byte(senha), bcrypt.DefaultCost)
	if err != nil {
		session.AddFlash("Erro ao processar a senha. Tente novamente.", "error")
		session.Save(c.Request, c.Writer)
		c.Redirect(http.StatusFound, "/cadastro")
		return
	}

	novoUsuario := model.Usuario{
		Nome:      nome,
		Email:     email,
		SenhaHash: string(senhaHash),
		Tipo:      model.RoleCliente,
	}

	result := database.DB.Create(&novoUsuario)
	if result.Error != nil {
		if strings.Contains(result.Error.Error(), "unique constraint") || strings.Contains(result.Error.Error(), "duplicate key") {
			session.AddFlash("Este e-mail já está cadastrado.", "error")
		} else {
			session.AddFlash("Erro ao criar usuário. Tente novamente.", "error")
		}
		session.Save(c.Request, c.Writer)
		c.Redirect(http.StatusFound, "/cadastro")
		return
	}

	session.AddFlash("Cadastro realizado com sucesso! Faça o login.", "success")
	session.Save(c.Request, c.Writer)
	c.Redirect(http.StatusFound, "/login")
}

// ShowLoginPage renderiza a página de login e exibe flash messages.
func (h *AuthHandler) ShowLoginPage(c *gin.Context) {
	session, _ := h.Store.Get(c.Request, "meu-cupcake-session")
	flashesSuccess := session.Flashes("success")
	flashesError := session.Flashes("error")
	err := session.Save(c.Request, c.Writer)
	if err != nil {
		fmt.Printf("AVISO: Erro ao salvar sessão em ShowLoginPage: %v\n", err)
	}

	c.HTML(http.StatusOK, "login.html", gin.H{
		"IsLoggedIn":     false,
		"FlashesSuccess": flashesSuccess,
		"FlashesError":   flashesError,
	})
}

// ProcessLoginForm processa os dados do formulário de login.
func (h *AuthHandler) ProcessLoginForm(c *gin.Context) {
	session, _ := h.Store.Get(c.Request, "meu-cupcake-session")
	email := c.PostForm("email")
	senha := c.PostForm("senha")

	var usuario model.Usuario
	result := database.DB.Where("email = ?", email).First(&usuario)

	if result.Error != nil && result.Error == gorm.ErrRecordNotFound {
		session.AddFlash("E-mail ou senha inválidos.", "error")
		session.Save(c.Request, c.Writer)
		c.Redirect(http.StatusFound, "/login")
		return
	}

	if result.Error != nil {
		session.AddFlash("Ocorreu um erro interno. Tente novamente.", "error")
		session.Save(c.Request, c.Writer)
		c.Redirect(http.StatusFound, "/login")
		return
	}

	err := bcrypt.CompareHashAndPassword([]byte(usuario.SenhaHash), []byte(senha))
	if err != nil {
		session.AddFlash("E-mail ou senha inválidos.", "error")
		session.Save(c.Request, c.Writer)
		c.Redirect(http.StatusFound, "/login")
		return
	}

	session.Values["userID"] = usuario.ID
	session.Values["userName"] = usuario.Nome

	err = session.Save(c.Request, c.Writer)
	if err != nil {
		fmt.Printf("ERRO ao salvar sessão de login: %v\n", err)
		session.AddFlash("Erro ao iniciar a sessão. Tente novamente.", "error")
		_ = session.Save(c.Request, c.Writer)
		c.Redirect(http.StatusFound, "/login")
		return
	}

	if usuario.Tipo == model.RoleLojista {
		c.Redirect(http.StatusFound, "/lojista/dashboard")
	} else { // Assume cliente
		c.Redirect(http.StatusFound, "/cliente/dashboard")
	}
}

func (h *AuthHandler) Logout(c *gin.Context) {
	session, _ := h.Store.Get(c.Request, "meu-cupcake-session")
	session.Values["userID"] = nil
	session.Values["userName"] = nil

	session.Options.MaxAge = -1
	err := session.Save(c.Request, c.Writer)
	if err != nil {
		fmt.Printf("Erro ao salvar sessão de logout: %v\n", err)
		c.String(http.StatusInternalServerError, "Erro ao fazer logout.")
		return
	}
	fmt.Println("Logout realizado com sucesso.")
	c.Redirect(http.StatusFound, "/login")
}

func (h *AuthHandler) AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session, _ := h.Store.Get(c.Request, "meu-cupcake-session")
		userID, ok := session.Values["userID"].(uint)
		if !ok {
			fmt.Println("AuthRequired: UserID não encontrado ou inválido na sessão. Redirecionando para /login.") // Log
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		var user model.Usuario
		if err := database.DB.First(&user, userID).Error; err != nil {
			fmt.Printf("AuthRequired: Usuário ID %d não encontrado no DB. Forçando logout.\n", userID)
			session.Values["userID"] = nil
			session.Values["userName"] = nil
			session.Options.MaxAge = -1
			session.Save(c.Request, c.Writer)
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		c.Set("user", user)
		fmt.Printf("AuthRequired: Usuário ID %d autenticado (%s).\n", userID, user.Email)
		c.Next()
	}
}

// RoleRequired é um middleware para verificar se o usuário logado tem o papel necessário.
func (h *AuthHandler) RoleRequired(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userData, exists := c.Get("user")
		if !exists {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		user := userData.(model.Usuario)

		if user.Tipo != requiredRole {
			fmt.Printf("RoleRequired: Acesso negado para usuário ID %d (%s). Role requerido: %s, Role do usuário: %s\n", user.ID, user.Email, requiredRole, user.Tipo) // Log
			c.String(http.StatusForbidden, "Acesso negado.")
			c.Abort()
			return
		}
		c.Next()
	}
}

package handler

import (
	"net/http"

	"github.com/ericoliveiras/meu-cupcake/internal/database"
	"github.com/ericoliveiras/meu-cupcake/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	Store *sessions.CookieStore
}

func ShowCadastroPage(c *gin.Context) {
	c.HTML(http.StatusOK, "cadastro.html", nil)
}

func ProcessCadastroForm(c *gin.Context) {
	nome := c.PostForm("nome")
	email := c.PostForm("email")
	senha := c.PostForm("senha")
	confirmarSenha := c.PostForm("confirmar_senha")

	if senha != confirmarSenha {
		c.String(http.StatusBadRequest, "Erro: As senhas não conferem!")
		return
	}

	senhaHash, err := bcrypt.GenerateFromPassword([]byte(senha), bcrypt.DefaultCost)
	if err != nil {
		c.String(http.StatusInternalServerError, "Erro ao processar a senha.")
		return
	}

	novoUsuario := model.Usuario{
		Nome:      nome,
		Email:     email,
		SenhaHash: string(senhaHash),
	}

	result := database.DB.Create(&novoUsuario)
	if result.Error != nil {
		c.String(http.StatusInternalServerError, "Erro ao criar usuário. O e-mail já pode estar em uso.")
		return
	}

	c.Redirect(http.StatusFound, "/login")
}

func (h *AuthHandler) ShowLoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", nil)
}

func (h *AuthHandler) ProcessLoginForm(c *gin.Context) {
	email := c.PostForm("email")
	senha := c.PostForm("senha")

	var usuario model.Usuario
	result := database.DB.Where("email = ?", email).First(&usuario)
	if result.Error != nil {
		c.String(http.StatusUnauthorized, "E-mail ou senha inválidos.")
		return
	}

	err := bcrypt.CompareHashAndPassword([]byte(usuario.SenhaHash), []byte(senha))
	if err != nil {
		c.String(http.StatusUnauthorized, "E-mail ou senha inválidos.")
		return
	}

	session, _ := h.Store.Get(c.Request, "meu-cupcake-session")
	session.Values["userID"] = usuario.ID
	session.Values["userName"] = usuario.Nome
	err = session.Save(c.Request, c.Writer)
	if err != nil {
		c.String(http.StatusInternalServerError, "Erro ao salvar a sessão.")
		return
	}

	switch usuario.Tipo {
	case model.RoleLojista:
		c.Redirect(http.StatusFound, "/lojista/dashboard")
	case model.RoleCliente:
		c.Redirect(http.StatusFound, "/cliente/dashboard")
	default:
		c.Redirect(http.StatusFound, "/")
	}
}

func (h *AuthHandler) Logout(c *gin.Context) {
	session, _ := h.Store.Get(c.Request, "meu-cupcake-session")

	session.Options.MaxAge = -1
	err := session.Save(c.Request, c.Writer)
	if err != nil {
		c.String(http.StatusInternalServerError, "Erro ao fazer logout.")
		return
	}

	c.Redirect(http.StatusFound, "/login")
}

func (h *AuthHandler) AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session, _ := h.Store.Get(c.Request, "meu-cupcake-session")
		userID, ok := session.Values["userID"].(uint)
		if !ok {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		// --- CORREÇÃO PRINCIPAL AQUI ---
		// Buscamos o usuário completo no banco de dados.
		var user model.Usuario
		if err := database.DB.First(&user, userID).Error; err != nil {
			// Se o usuário não existe mais no banco, força o logout.
			c.Redirect(http.StatusFound, "/logout")
			c.Abort()
			return
		}

		// Colocamos o objeto User inteiro no contexto.
		c.Set("user", user)

		c.Next()
	}
}

func (h *AuthHandler) RoleRequired(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		session, _ := h.Store.Get(c.Request, "meu-cupcake-session")
		userID, ok := session.Values["userID"].(uint) 

		if !ok {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		var user model.Usuario
		if err := database.DB.First(&user, userID).Error; err != nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		if user.Tipo != requiredRole {
			c.String(http.StatusForbidden, "Acesso negado.")
			c.Abort()
			return
		}

		c.Next()
	}
}
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

func (h *HomeHandler) ShowHomePage(c *gin.Context) {
	session, _ := h.Store.Get(c.Request, "meu-cupcake-session")

	userID := session.Values["userID"]
	userName := session.Values["userName"]

	c.HTML(http.StatusOK, "index.html", gin.H{
		"IsLoggedIn": userID != nil, 
		"UserName":   userName,
	})
}

func (h *HomeHandler) ShowProfilePage(c *gin.Context) {
	c.String(http.StatusOK, "Esta é a sua página de perfil!")
}

func (h *HomeHandler) ShowVitrinePage(c *gin.Context) {
	var cupcakes []model.Cupcake

	if err := database.DB.Where("disponivel = ?", true).Order("created_at desc").Find(&cupcakes).Error; err != nil {
		c.String(http.StatusInternalServerError, "Não foi possível carregar a vitrine. Tente novamente mais tarde.")
		return
	}
	
	session, _ := h.Store.Get(c.Request, "meu-cupcake-session")
	userID := session.Values["userID"]
	userName := session.Values["userName"]

	c.HTML(http.StatusOK, "vitrine.html", gin.H{
		"Cupcakes":   cupcakes,
		"IsLoggedIn": userID != nil,
		"UserName":   userName,
	})
}
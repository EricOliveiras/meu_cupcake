// /internal/database/seed.go
package database

import (
	"log"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/ericoliveiras/meu-cupcake/internal/model"
)

func SeedLojista() {
	var user model.Usuario
	result := DB.Where("email = ?", "lojista@meucupcake.com").First(&user)

	if result.Error != nil && result.Error == gorm.ErrRecordNotFound {
		log.Println("Usuário lojista não encontrado, criando um novo...")

		senhaHash, err := bcrypt.GenerateFromPassword([]byte("senhaforte123"), bcrypt.DefaultCost)
		if err != nil {
			log.Fatalf("Falha ao criar hash da senha do lojista: %v", err)
		}

		lojista := model.Usuario{
			Nome:      "Lojista Principal",
			Email:     "lojista@meucupcake.com",
			SenhaHash: string(senhaHash),
			Tipo:      model.RoleLojista, 
		}

		if err := DB.Create(&lojista).Error; err != nil {
			log.Fatalf("Falha ao criar o usuário lojista: %v", err)
		}
		log.Println("Usuário lojista criado com sucesso.")
	} else {
		log.Println("Usuário lojista já existe.")
	}
}
# Meu Cupcake 🧁

Uma aplicação web de e-commerce desenvolvida em Go para a venda de cupcakes personalizados. Este projeto permite que clientes visualizem produtos, montem pedidos, gerenciem seus carrinhos e realizem pagamentos (integrado com o Mercado Pago em ambiente de teste). Lojistas possuem um painel administrativo para gerenciar o catálogo de produtos e visualizar o histórico de vendas.

## ✨ Funcionalidades Principais

- **Autenticação de Usuários:** Cadastro (cliente), Login e Logout.
- **Gerenciamento de Sessão:** Mantém o usuário conectado.
- **Controle de Acesso Baseado em Papel:**
  - **Cliente:** Pode ver vitrine, gerenciar carrinho, finalizar compra, ver histórico de pedidos, gerenciar perfil.
  - **Lojista:** Pode gerenciar produtos (CRUD com upload de imagem), ver histórico de vendas, gerenciar perfil.
- **Gerenciamento de Produtos (Lojista):**
  - Listar cupcakes cadastrados.
  - Adicionar novo cupcake (com nome, descrição, preço, imagem e status de disponibilidade) via modal.
  - Editar cupcake existente via modal.
  - Excluir cupcake (soft delete no banco e exclusão do arquivo de imagem).
- **Vitrine de Produtos (Cliente/Visitante):**
  - Exibe cupcakes disponíveis em formato de card.
  - Modal para visualização de detalhes do produto.
- **Carrinho de Compras (Cliente/Visitante):**
  - Adicionar itens ao carrinho (via vitrine ou modal).
  - Visualizar carrinho detalhado.
  - Aumentar/Diminuir quantidade de itens.
  - Remover item do carrinho.
  - Limpar carrinho.
  - Armazenamento via sessão.
- **Checkout:**
  - Página de resumo do pedido.
  - Integração com Mercado Pago (CardForm/Bricks) para coleta segura de dados de cartão (ambiente de teste).
- **Processamento de Pagamento (Backend):**
  - Recebe token do Mercado Pago.
  - Recalcula e valida o total do pedido.
  - Cria o registro do pedido no banco de dados local (status inicial: pendente).
  - Chama a API do Mercado Pago para criar o pagamento (ambiente de teste).
  - Atualiza o status do pedido no banco de dados local (pago, falhou, pendente).
  - Limpa o carrinho em caso de sucesso.
- **Histórico:**
  - Página de histórico de pedidos para o cliente.
  - Página de histórico de vendas para o lojista.
- **Interface Responsiva:** Cabeçalho com menu hamburger, tabelas com rolagem horizontal, layouts adaptáveis.
- **Flash Messages:** Feedback visual para o usuário após ações (ex: item adicionado, erro no login).

## 🚀 Tecnologias Utilizadas

- **Backend:** Go (Golang)
- **Framework Web:** Gin
- **ORM:** GORM
- **Banco de Dados:** PostgreSQL (configurado para usar Neon DB na nuvem)
- **Gerenciamento de Sessão:** Gorilla Sessions
- **Hashing de Senha:** Bcrypt
- **Gerenciamento de Configuração:** Arquivo `.env` com Godotenv
- **Frontend:** HTML (Templates Go), CSS, JavaScript
- **Gateway de Pagamento:** Mercado Pago SDK Go V2 (Ambiente de Teste)
- **Containerização (Opcional para App):** Docker, Docker Compose
- **Compartilhamento (Opcional):** Cloudflare Tunnel


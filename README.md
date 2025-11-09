# Meu Cupcake 🧁

Uma aplicação web de e-commerce desenvolvida em Go para a venda de cupcakes personalizados. Este projeto permite que clientes visualizem produtos, montem pedidos, gerenciem seus carrinhos e realizem pagamentos (integrado com o Mercado Pago em ambiente de teste). Lojistas possuem um painel administrativo para gerenciar o catálogo de produtos e visualizar o histórico de vendas.

**Este projeto foi desenvolvido como parte do Projeto Integrador Transdisciplinar em Engenharia de Software II.**

## ✨ Funcionalidades Principais Implementadas

- **Autenticação de Usuários:** Cadastro (cliente), Login e Logout.
- **Gerenciamento de Sessão:** Mantém o usuário conectado.
- **Controle de Acesso Baseado em Papel:** Diferenciação entre Cliente e Lojista.
  - **Cliente:** Pode ver vitrine, gerenciar carrinho, finalizar compra, ver histórico de pedidos, gerenciar perfil.
  - **Lojista:** Pode gerenciar produtos (CRUD com upload de imagem), ver histórico de vendas, gerenciar perfil. (Acesso via credenciais específicas).
- **Gerenciamento de Produtos (Lojista):** Listar, Adicionar (via modal), Editar (via modal), Excluir (soft delete + exclusão de arquivo).
- **Vitrine de Produtos:** Exibe cupcakes disponíveis em formato de card, com modal para detalhes.
- **Carrinho de Compras:** Adicionar, visualizar, aumentar/diminuir quantidade, remover item, limpar carrinho (armazenado em sessão).
- **Checkout:** Página de resumo do pedido e integração com Mercado Pago (CardForm/Bricks) para coleta segura de dados de cartão (ambiente de teste).
- **Processamento de Pagamento (Backend):** Validação de carrinho/total, criação de pedido no DB, chamada à API do Mercado Pago (teste), atualização de status do pedido.
- **Histórico:** Página de histórico de pedidos para o cliente e vendas para o lojista.
- **Interface Responsiva:** Cabeçalho com menu hamburger, tabelas com rolagem horizontal, layouts adaptáveis.
- **Flash Messages:** Feedback visual para o usuário.

## 🚀 Tecnologias Utilizadas

- **Backend:** Go (Golang)
- **Framework Web:** Gin
- **ORM:** GORM
- **Banco de Dados:** PostgreSQL (hospedado na Neon DB)
- **Gerenciamento de Sessão:** Gorilla Sessions
- **Hashing de Senha:** Bcrypt
- **Frontend:** HTML (Templates Go), CSS, JavaScript
- **Gateway de Pagamento:** Mercado Pago SDK Go V2 (Ambiente de Teste)
- **Containerização:** Docker, Dockerfile
- **Hospedagem:** Fly.io

## 🌐 Acesso à Aplicação para Teste

A aplicação está hospedada e pode ser acessada através do seguinte link:

**[https://meu-cupcake-winter-frog-3330.fly.dev/](https://meu-cupcake-winter-frog-3330.fly.dev/)**

## 🧪 Instruções para Teste e Feedback

[cite_start]**O objetivo é coletar feedback sobre a usabilidade, funcionalidade e possíveis bugs da aplicação, conforme solicitado na Situação-Problema 3 do Projeto Integrador[cite: 471]. Por favor, siga os passos abaixo:**

1.  **Acesse a Aplicação:** Utilize o link fornecido acima.
2.  **Crie uma Conta:** Clique em "Cadastrar" e crie uma conta de **cliente** com seu e-mail (ou um e-mail fictício válido) e uma senha.
3.  **Explore as Funcionalidades do Cliente:**
    - Navegue pela Vitrine.
    - Adicione diferentes cupcakes ao carrinho (variando as quantidades).
    - Abra o modal de detalhes.
    - Acesse a página do Carrinho.
    - Teste aumentar/diminuir quantidades e remover itens.
    - Prossiga para o Checkout.
4.  **Simule um Pagamento (Ambiente de Teste):**
    - Na página de Checkout, você verá os campos para cartão de crédito. Utilize **obrigatoriamente** um dos **cartões de teste** fornecidos pelo Mercado Pago abaixo:
      - **Cartão Aprovado:**
        - Número: `5031 4332 1540 6351`
        - Validade: Qualquer data futura (ex: `11/30`)
        - CVV: `123`
        - Nome: "APRO"
        - CPF: "12345678909"
      - **Cartão Recusado (Outros Erros):**
        - Número: `5031 4332 1540 6351`
        - Validade: Qualquer data futura (ex: `11/30`)
        - CVV: `123`
        - Nome: "FUND", "CONT", ou "FORM"
        - CPF: "12345678909"
    - Preencha os demais campos (Nome no Cartão, CPF, etc.) com dados fictícios válidos.
    - Clique em "Pagar Agora". Você deverá ser redirecionado para a página de sucesso (se usar o cartão aprovado).
5.  **Verifique o Histórico:** Após um pagamento aprovado, acesse "Minha Conta" (no menu do cabeçalho) > "Meus Pedidos" para ver se o pedido aparece com o status correto.
6.  **Acesse o Painel do Lojista (Opcional):**
    - Faça logout da sua conta de cliente.
    - Faça login com as credenciais:
      - Email: `lojista@meucupcake.com`
      - Senha: `senhaforte123`
    - Explore as opções: "Gerenciar Cupcakes" (adicione/edite/exclua) e "Histórico de Vendas".
7.  **Responda ao Formulário de Feedback:** Por favor, acesse o link abaixo e responda ao questionário com suas impressões, bugs encontrados e sugestões. Seu feedback é muito importante!
    - **Link do Formulário Google:** [https://docs.google.com/forms/d/e/1FAIpQLSdpEJKlOypCjiigvD56hUZFRlh3SiHu5GVGFEtTjLveyJKksA/viewform?usp=dialog](https://docs.google.com/forms/d/e/1FAIpQLSdpEJKlOypCjiigvD56hUZFRlh3SiHu5GVGFEtTjLveyJKksA/viewform?usp=dialog)

**Formato do Feedback (Conforme Projeto Integrador):**
Ao preencher o formulário, por favor, detalhe:

- O que você testou e funcionou bem.
- O que você testou e não funcionou (descreva o problema e, se possível, os passos para reproduzi-lo).
- Quaisquer funcionalidades que você esperava e não encontrou, ou sugestões de melhoria.

Muito obrigado pela sua colaboração!

## 🏗️ Estrutura do Projeto

```bash
meu-cupcake/
├── cmd/
│   └── web/
│       └── main.go           # Ponto de entrada: bootstrap, rotas e inicialização do servidor
├── internal/
│   ├── config/               # Carregamento de .env, validação e structs de config (IMPLEMENTAÇÃO FUTURA SUGERIDA)
│   ├── database/             # Conexão com Postgres, migrations e seeders
│   ├── handler/              # Controllers (Gin handlers) — endpoints HTTP
│   ├── middleware/           # Autenticação, autorização, sessões (IMPLEMENTAÇÃO FUTURA SUGERIDA)
│   ├── model/                # Models GORM (User, Product, Order, Cart, etc.)
│   ├── service/              # Regras de negócio (pagamento, pedidos, catálogo) (IMPLEMENTAÇÃO FUTURA SUGERIDA)
│   └── view/
│       └── templates/        # Templates Go (HTML) e partials (_header.html)
├── static/
│   ├── css/                  # Arquivos CSS
│   ├── js/                   # Arquivos JavaScript (se houver mais complexidade)
│   └── img/                  # Assets públicos (layout, ícones)
├── uploads/                  # Imagens de produtos (armazenamento local/depósito)
├── scripts/                  # Scripts auxiliares (migrations, seed, deploy helpers) (IMPLEMENTAÇÃO FUTURA SUGERIDA)
├── .github/                  # Workflows CI/CD (opcional)
├── go.mod                    # Dependências Go
├── go.sum                    # Checksums das dependências
├── .env.example              # Exemplo de variáveis de ambiente (sem segredos)
├── Dockerfile                # Build da imagem da aplicação
├── fly.toml                  # Configuração para deploy no Fly.io
└── README.md                 # Documentação do projeto
```

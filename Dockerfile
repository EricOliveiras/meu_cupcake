# Dockerfile

# --- Estágio 1: Build ---
FROM golang:1.25-alpine AS builder

# Define o diretório de trabalho
WORKDIR /app

# --- CORREÇÃO AQUI: Instala o Git ---
# Alpine usa 'apk' como gerenciador de pacotes
RUN apk add --no-cache git
# ------------------------------------

# Copia os arquivos de gerenciamento de dependências
COPY go.mod go.sum ./

# Baixa as dependências (agora com Git disponível)
RUN go mod download
RUN go mod verify

# Copia todo o código fonte
COPY . .

# Compila a aplicação Go
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o app ./cmd/web/main.go

# --- Estágio 2: Execução ---
FROM alpine:latest

# Instala certificados CA
RUN apk update && apk add --no-cache ca-certificates tzdata && update-ca-certificates

# Define o diretório de trabalho
WORKDIR /app

# Copia o binário compilado
COPY --from=builder /app/app .

# Copia os assets
COPY internal/view/templates ./internal/view/templates
COPY static ./static
RUN mkdir -p uploads
COPY uploads ./uploads

# Expõe a porta
EXPOSE 8080

# Comando para executar
CMD ["./app"]
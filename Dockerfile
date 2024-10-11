# Use a imagem oficial do Golang no Debian como base
FROM golang:1.22.5-bullseye as builder

# Instalação das dependências
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    git \
    && rm -rf /var/lib/apt/lists/*

# Configuração do ambiente Go
ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH

# Copia o código fonte para o diretório de trabalho
WORKDIR /app
COPY app .

# Compilação do aplicativo
RUN go build -o /app/app .

# Imagem mínima do Debian para executar o aplicativo compilado
FROM debian:bullseye-slim

# Copia o binário compilado do estágio anterior
COPY --from=builder /app/app /app/app

# Comando padrão para iniciar o aplicativo
CMD ["/app/app"]

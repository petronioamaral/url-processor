package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"golang.org/x/net/context"
)

var rdb *redis.Client

const maxRetries = 5
const retryInterval = 2 * time.Second

func main() {
	// Obtendo configurações do Redis via variáveis de ambiente
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379" // Valor padrão
	}
	redisDB, err := strconv.Atoi(os.Getenv("REDIS_DB"))
	if err != nil {
		redisDB = 0 // Valor padrão
	}

	// Configuração do cliente Redis
	rdb = redis.NewClient(&redis.Options{
		Addr: redisAddr,
		DB:   redisDB,
	})

	// Configuração do servidor HTTP com Gin
	router := gin.Default()
	router.POST("/urls", saveURLHandler)
	router.GET("/urls", listURLsHandler)

	// Iniciar o processamento das URLs em uma goroutine separada
	go processURLs()

	// Iniciar o servidor HTTP
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Falha ao iniciar o servidor HTTP: %v", err)
	}
}

func saveURLHandler(c *gin.Context) {
	url := c.PostForm("url")

	// Verificar se a URL foi fornecida
	if url == "" {
		log.Printf("URL não fornecida")
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL não fornecida"})
		return
	}

	// Salva a URL no Redis
	err := rdb.LPush(context.Background(), "urls", url).Err()
	if err != nil {
		log.Printf("Erro ao salvar URL no Redis: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar URL"})
		return
	}

	log.Printf("URL salva no Redis: %s", url)
	c.JSON(http.StatusOK, gin.H{"message": "URL salva com sucesso"})
}

func listURLsHandler(c *gin.Context) {
	// Obter as 10 últimas URLs da lista 'urls' no Redis
	urls, err := rdb.LRange(context.Background(), "urls", 0, 9).Result()
	if err != nil {
		log.Printf("Erro ao obter URLs do Redis: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao obter URLs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"urls": urls})
}

func processURLs() {
	for {
		// LPop para pegar a URL mais antiga
		result, err := rdb.LPop(context.Background(), "urls").Result()
		if err != nil {
			if err != redis.Nil {
				log.Printf("Erro ao obter URL do Redis: %v", err)
			}
			time.Sleep(time.Second) // Espera um segundo antes de tentar novamente
			continue
		}

		// Processar a URL em uma goroutine separada
		go processURL(result)
	}
}

func processURL(url string) {
	for i := 0; i < maxRetries; i++ {
		// Tentar chamar a URL
		resp, err := http.Get(url)
		if err != nil {
			log.Printf("Erro ao chamar a URL %s na tentativa %d: %v", url, i+1, err)
			time.Sleep(retryInterval) // Espera antes de tentar novamente
			continue
		}
		defer resp.Body.Close()

		// Se a chamada for bem-sucedida, registra o sucesso e retorna
		log.Printf("URL %s chamada com sucesso, status: %s", url, resp.Status)
		return
	}

	// Se todas as tentativas falharem, registra o erro
	log.Printf("Falha ao chamar a URL %s após %d tentativas", url, maxRetries)

	// Gravar um log adicional em um arquivo no caso de falha em todas as tentativas
	f, err := os.OpenFile("failed_urls.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Erro ao abrir arquivo de log: %v", err)
		return
	}
	defer f.Close()

	logger := log.New(f, "", log.LstdFlags)
	logger.Printf("Falha ao chamar a URL %s após %d tentativas", url, maxRetries)
}

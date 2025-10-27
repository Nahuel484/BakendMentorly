package main

import (
	"context"
	"fmt"
	"log"
	"mentorly-backend/handlers"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	// Cargar variables de entorno
	err := godotenv.Load()
	if err != nil {
		log.Println("Advertencia: No se pudo cargar el archivo .env")
	}

	// Conectar a base de datos
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("Error: DATABASE_URL no está configurada")
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		log.Fatalf("Error al crear pool de conexiones: %v", err)
	}
	defer pool.Close()

	// Verificar conexión
	err = pool.Ping(context.Background())
	if err != nil {
		log.Fatalf("Error al conectar a base de datos: %v", err)
	}

	fmt.Println("✓ Conexión a base de datos exitosa")

	// Inicializar Handler
	handler := handlers.NewHandler(pool)

	// Inicializar Gin
	router := gin.Default()

	// Configurar CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:3000", "https://mentorly-web.vercel.app/", "https://mentorly-web.vercel.app", "http://localhost:5174"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Rutas públicas
	router.POST("/auth/register", handler.RegisterHandler)
	router.POST("/auth/login", handler.LoginHandler)

	// Rutas protegidas
	protected := router.Group("/")
	protected.Use(handlers.AuthMiddleware())
	{
		protected.POST("/auth/select-role", handler.SelectRoleHandler)
		protected.GET("/user/profile", handler.GetProfileHandler)
	}

	fmt.Println("✓ Servidor iniciado en http://localhost:8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Error al iniciar servidor: %v", err)
	}
}

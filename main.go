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

	// Verificar que el secreto del JWT esté configurado
	if os.Getenv("JWT_SECRET") == "" {
		log.Fatal("Error: JWT_SECRET no está configurada")
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

	// --- BLOQUE TEMPORAL PARA GENERAR HASH ---
	// Descomenta las siguientes líneas para generar un nuevo hash de contraseña
	//tempAuthService := services.NewAuthService(pool)
	//newHash, _ := tempAuthService.HashPassword("password123")
	//fmt.Println("========================================")
	//fmt.Println("NUEVO HASH PARA 'password123':")
	//fmt.Println(newHash)
	//fmt.Println("========================================")
	// --- FIN DEL BLOQUE TEMPORAL ---

	// Inicializar Handlers
	authHandler := handlers.NewHandler(pool)
	oauthHandler := handlers.NewOAuthHandler(pool)

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

	api := router.Group("/api")

	// Rutas públicas - Autenticación tradicional
	api.POST("/auth/register", authHandler.RegisterHandler)
	api.POST("/auth/login", authHandler.LoginHandler)

	// Rutas de OAuth - URLs de autenticación
	api.GET("/auth/google/url", oauthHandler.GetGoogleAuthURL)
	api.GET("/auth/github/url", oauthHandler.GetGitHubAuthURL)
	api.GET("/auth/linkedin/url", oauthHandler.GetLinkedInAuthURL)

	// Callbacks de OAuth
	api.GET("/auth/google/callback", oauthHandler.GoogleCallbackHandler)
	api.GET("/auth/github/callback", oauthHandler.GitHubCallbackHandler)
	api.GET("/auth/linkedin/callback", oauthHandler.LinkedInCallbackHandler)

	// Rutas protegidas
	userRoutes := api.Group("/user")
	userRoutes.Use(handlers.AuthMiddleware())
	{
		userRoutes.POST("/select-role", authHandler.SelectRoleHandler)
		userRoutes.GET("/profile", authHandler.GetProfileHandler)
		userRoutes.PUT("/profile", authHandler.UpdateProfileHandler)
		userRoutes.POST("/subscribe/:plan_id", authHandler.SubscribeToPlanHandler)
	}

	// Rutas de administración (protegidas por rol de admin)
	admin := api.Group("/admin")
	admin.Use(handlers.AuthMiddleware(), authHandler.AdminMiddleware())
	{
		admin.POST("/plans", authHandler.CreatePlanHandler)
		admin.GET("/plans", authHandler.GetAllPlansHandler)
		admin.GET("/plans/:id", authHandler.GetPlanByIDHandler)
		admin.PUT("/plans/:id", authHandler.UpdatePlanHandler)
		admin.DELETE("/plans/:id", authHandler.DeletePlanHandler)
	}

	fmt.Println("✓ Servidor iniciado en http://localhost:8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Error al iniciar servidor: %v", err)
	}
}

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

	// Inicializar Handlers
	authHandler := handlers.NewAuthHandler(pool)
	oauthHandler := handlers.NewOAuthHandler(pool)
	sessionHandler := handlers.NewSessionHandler(pool)
	skillHandler := handlers.NewSkillHandler(pool)
	especialidadHandler := handlers.NewEspecialidadHandler(pool)
	notificationHandler := handlers.NewNotificationHandler(pool)
	profileHandler := handlers.NewProfileHandler(pool)

	// Inicializar Gin
	router := gin.Default()

	// Configurar CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:3000", "https://mentorly-web.vercel.app", "http://localhost:5174"},
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

	// ============================================================
	// RUTAS PÚBLICAS - AUTENTICACIÓN TRADICIONAL
	// ============================================================
	api.POST("/auth/register", authHandler.RegisterHandler)
	api.POST("/auth/login", authHandler.LoginHandler)

	// ============================================================
	// RUTAS DE OAUTH
	// ============================================================
	// URLs de autenticación
	api.GET("/auth/google/url", oauthHandler.GetGoogleAuthURL)
	api.GET("/auth/github/url", oauthHandler.GetGitHubAuthURL)
	api.GET("/auth/linkedin/url", oauthHandler.GetLinkedInAuthURL)

	// Callbacks de OAuth
	api.GET("/auth/google/callback", oauthHandler.GoogleCallbackHandler)
	api.GET("/auth/github/callback", oauthHandler.GitHubCallbackHandler)
	api.GET("/auth/linkedin/callback", oauthHandler.LinkedInCallbackHandler)

	// ============================================================
	// RUTAS PROTEGIDAS - USUARIO
	// ============================================================
	userRoutes := api.Group("/user")
	userRoutes.Use(handlers.AuthMiddleware())
	{
		// Perfil
		userRoutes.GET("/profile", profileHandler.GetProfileHandler)
		userRoutes.PUT("/profile", profileHandler.UpdateProfileHandler)

		// Rol
		userRoutes.POST("/select-role", authHandler.SelectRoleHandler)

		// Suscripción
		userRoutes.POST("/subscribe/:plan_id", authHandler.SubscribeToPlanHandler)

		// Sesiones
		userRoutes.POST("/logout", sessionHandler.LogoutHandler)
		userRoutes.POST("/logout-all", sessionHandler.LogoutAllHandler)

		// Habilidades del usuario
		userRoutes.GET("/skills", skillHandler.GetUserSkillsHandler)
		userRoutes.POST("/skills", skillHandler.AddSkillToUserHandler)
		userRoutes.PUT("/skills/:skill_id/level", skillHandler.UpdateUserSkillLevelHandler)
		userRoutes.DELETE("/skills/:skill_id", skillHandler.RemoveUserSkillHandler)

		// Especialidades del usuario
		userRoutes.GET("/especialidades", especialidadHandler.GetUserEspecialidadesHandler)
		userRoutes.POST("/especialidades", especialidadHandler.AddEspecialidadToUserHandler)
		userRoutes.DELETE("/especialidades/:especialidad_id", especialidadHandler.RemoveUserEspecialidadHandler)

		// Notificaciones
		userRoutes.GET("/notifications", notificationHandler.GetUserNotificationsHandler)
		userRoutes.GET("/notifications/unread-count", notificationHandler.GetUnreadCountHandler)
		userRoutes.POST("/notifications", notificationHandler.CreateNotificationHandler)
		userRoutes.PUT("/notifications/:id/read", notificationHandler.MarkAsReadHandler)
		userRoutes.PUT("/notifications/read-all", notificationHandler.MarkAllAsReadHandler)
		userRoutes.DELETE("/notifications/:id", notificationHandler.DeleteNotificationHandler)
	}

	// ============================================================
	// RUTAS PÚBLICAS - HABILIDADES (INFORMACIÓN)
	// ============================================================
	skillsPublic := api.Group("/skills")
	{
		skillsPublic.GET("", skillHandler.GetAllSkillsHandler)
		skillsPublic.GET("/:id", skillHandler.GetSkillByIDHandler)
	}

	// ============================================================
	// RUTAS PÚBLICAS - ESPECIALIDADES (INFORMACIÓN)
	// ============================================================
	especialidadesPublic := api.Group("/especialidades")
	{
		especialidadesPublic.GET("", especialidadHandler.GetAllEspecialidadesHandler)
		especialidadesPublic.GET("/:id", especialidadHandler.GetEspecialidadByIDHandler)
	}

	// ============================================================
	// RUTAS DE ADMINISTRACIÓN (PROTEGIDAS POR ROL)
	// ============================================================
	admin := api.Group("/admin")
	admin.Use(handlers.AuthMiddleware(), authHandler.AdminMiddleware())
	{
		// Planes
		admin.POST("/plans", authHandler.CreatePlanHandler)
		admin.GET("/plans", authHandler.GetAllPlansHandler)
		admin.GET("/plans/:id", authHandler.GetPlanByIDHandler)
		admin.PUT("/plans/:id", authHandler.UpdatePlanHandler)
		admin.DELETE("/plans/:id", authHandler.DeletePlanHandler)

		// Habilidades (crear y gestionar)
		admin.POST("/skills", skillHandler.CreateSkillHandler)

		// Especialidades (crear y gestionar)
		admin.POST("/especialidades", especialidadHandler.CreateEspecialidadHandler)
		admin.DELETE("/especialidades/:id", especialidadHandler.DeleteEspecialidadHandler)
	}

	fmt.Println("✓ Servidor iniciado en http://localhost:8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Error al iniciar servidor: %v", err)
	}
}

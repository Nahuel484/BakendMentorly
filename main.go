package main

import (
	"context"
	"fmt"
	"log"
	"mentorly-backend/handlers"
	"mentorly-backend/services"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	// Cargar variables de entorno
	err := godotenv.Load()
	fmt.Println("MP ACCESS:", os.Getenv("MP_ACCESS_TOKEN"))
	fmt.Println("FRONT:", os.Getenv("FRONTEND_URL"))
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

	// Crear configuración del pool
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		log.Fatalf("Error al parsear DATABASE_URL: %v", err)
	}

	// Usar siempre Simple Protocol (sin prepared statements)
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
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
	suscripcionService := services.NewSuscripcionService(pool)
	authHandler := handlers.NewAuthHandler(pool, suscripcionService)
	oauthHandler := handlers.NewOAuthHandler(pool)
	sessionHandler := handlers.NewSessionHandler(pool)
	skillHandler := handlers.NewSkillHandler(pool)
	especialidadHandler := handlers.NewEspecialidadHandler(pool)
	notificationService := services.NewNotificationService(pool)
	notificationHandler := handlers.NewNotificationHandler(pool)
	profileHandler := handlers.NewProfileHandler(pool)
	solicitudService := services.NewSolicitudService(pool)
	solicitudHandler := handlers.NewSolicitudHandler(solicitudService, suscripcionService)

	postulacionService := services.NewPostulacionService(pool, notificationService)
	postulacionHandler := handlers.NewPostulacionHandler(postulacionService, suscripcionService)

	contratacionService := services.NewContratacionService(pool, notificationService)
	contratacionHandler := handlers.NewContratacionHandler(contratacionService)

	messageService := services.NewMessageService(pool, notificationService)
	messageHandler := handlers.NewMessageHandler(messageService)

	// 🔹 Planes (para frontend + pagos)
	planService := services.NewPlanService(pool)
	planHandler := handlers.NewPlanHandler(planService)

	mpService := services.NewMercadoPagoService()

	mpWebhook := handlers.NewMPWebhookHandler(mpService, suscripcionService)
	suscripcionHandler := handlers.NewSuscripcionHandler(suscripcionService, planService)
	paymentHandler := handlers.NewPaymentHandler(planService, mpService, suscripcionService)
	// Inicializar Gin
	router := gin.Default()

	// Configurar CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:3000", "https://mentorly-web.vercel.app", "http://localhost:5174, http://127.0.0.1:11463"},
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

	api.GET("/planes", planHandler.GetActivePlans)
	api.GET("/planes/:id", planHandler.GetPlanByID)
	api.POST("/payments/webhook", mpWebhook.HandleWebhook)

	// ============================================================
	// RUTAS PROTEGIDAS - USUARIO
	// ============================================================
	userRoutes := api.Group("/user")
	userRoutes.Use(handlers.AuthMiddleware())
	{
		// Perfil
		userRoutes.GET("/profile", profileHandler.GetProfileHandler)
		userRoutes.PUT("/profile", profileHandler.UpdateProfileHandler)
		userRoutes.GET("/public/:id", profileHandler.GetPublicProfileHandler)
		//Explore
		userRoutes.GET("/explore", profileHandler.ListUsersByRoleHandler)
		// Rol
		userRoutes.POST("/select-role", authHandler.SelectRoleHandler)

		// Suscripción
		userRoutes.POST("/subscribe/:plan_id", authHandler.SubscribeToPlanHandler)
		userRoutes.GET("/subscriptions/me", suscripcionHandler.GetMySubscription)
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

	secured := api.Group("/")
	secured.Use(handlers.AuthMiddleware())
	{
		// Solicitudes
		secured.POST("/solicitudes", solicitudHandler.CreateSolicitud)
		secured.GET("/solicitudes/explore", solicitudHandler.ListSolicitudesAbiertas)
		secured.GET("/solicitudes/mias", solicitudHandler.ListMisSolicitudes)
		secured.DELETE("/solicitudes/:id", solicitudHandler.DeleteSolicitud)

		// Postulaciones
		secured.POST("/postulaciones", postulacionHandler.CreatePostulacion)
		secured.POST("/postulaciones/rechazar", postulacionHandler.RejectPostulacion)

		// Contrataciones
		secured.POST("/contrataciones", contratacionHandler.CreateContratacion)
		// Mp
		secured.POST("/payments/mercadopago/preference", paymentHandler.CreateMercadoPagoPreference)

		// Mensajes
		secured.POST("/messages", messageHandler.SendMessage)
		secured.GET("/conversaciones/mias", messageHandler.ListMisConversaciones)            // listar chats del usuario
		secured.GET("/conversaciones/:id/mensajes", messageHandler.ListMensajesConversacion) // mensajes de un chat
		secured.POST("/conversaciones/:id/cerrar", messageHandler.CerrarConversacion)        // cerrar chat
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

		// Habilidades del usuario
		admin.GET("/skills", skillHandler.GetUserSkillsHandler)
		admin.POST("/skills", skillHandler.AddSkillToUserHandler)
		admin.PUT("/skills/:skill_id", skillHandler.UpdateUserSkillHandler) // NUEVA RUTA
		admin.PUT("/skills/:skill_id/level", skillHandler.UpdateUserSkillLevelHandler)
		admin.DELETE("/skills/:skill_id", skillHandler.RemoveUserSkillHandler)

		// Especialidades (crear y gestionar)
		admin.POST("/especialidades", especialidadHandler.CreateEspecialidadHandler)
		admin.DELETE("/especialidades/:id", especialidadHandler.DeleteEspecialidadHandler)
	}

	fmt.Println("✓ Servidor iniciado en http://localhost:8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Error al iniciar servidor: %v", err)
	}
}

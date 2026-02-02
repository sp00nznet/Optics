package router

import (
	"github.com/gin-gonic/gin"
	"github.com/optics/backend/internal/config"
	"github.com/optics/backend/internal/handlers"
	"github.com/optics/backend/internal/middleware"
	"github.com/optics/backend/internal/services"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Setup(
	pool *pgxpool.Pool,
	cfg *config.Config,
	scanner *services.Scanner,
) *gin.Engine {
	r := gin.Default()

	// CORS middleware
	r.Use(corsMiddleware())

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(cfg.JWT.Secret)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(pool, cfg.JWT.Secret, cfg.JWT.Expiry)
	modelHandler := handlers.NewModelHandler(pool, &cfg.Storage)
	categoryHandler := handlers.NewCategoryHandler(pool)
	tagHandler := handlers.NewTagHandler(pool)
	collectionHandler := handlers.NewCollectionHandler(pool)
	adminHandler := handlers.NewAdminHandler(pool, cfg, scanner)

	// API routes
	api := r.Group("/api")
	{
		// Auth routes (public)
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authMiddleware.RequireAuth(), authHandler.Refresh)
			auth.GET("/me", authMiddleware.RequireAuth(), authHandler.Me)
		}

		// Models routes
		models := api.Group("/models")
		{
			models.GET("", authMiddleware.OptionalAuth(), modelHandler.List)
			models.GET("/:id", authMiddleware.OptionalAuth(), modelHandler.Get)
			models.GET("/:id/file", modelHandler.ServeFile)
			models.GET("/:id/thumbnail", modelHandler.ServeThumbnail)
			models.PUT("/:id", authMiddleware.RequireAuth(), modelHandler.Update)
			models.DELETE("/:id", authMiddleware.RequireAuth(), authMiddleware.RequireAdmin(), modelHandler.Delete)
			models.PUT("/:id/category", authMiddleware.RequireAuth(), modelHandler.SetCategory)
			models.POST("/:id/tags", authMiddleware.RequireAuth(), modelHandler.AddTags)
			models.DELETE("/:id/tags/:tagId", authMiddleware.RequireAuth(), modelHandler.RemoveTag)
			models.POST("/:id/favorite", authMiddleware.RequireAuth(), modelHandler.ToggleFavorite)
		}

		// Search
		api.GET("/search", authMiddleware.OptionalAuth(), modelHandler.Search)

		// Categories routes
		categories := api.Group("/categories")
		{
			categories.GET("", categoryHandler.List)
			categories.GET("/:id", categoryHandler.Get)
			categories.POST("", authMiddleware.RequireAuth(), authMiddleware.RequireAdmin(), categoryHandler.Create)
			categories.PUT("/:id", authMiddleware.RequireAuth(), authMiddleware.RequireAdmin(), categoryHandler.Update)
			categories.DELETE("/:id", authMiddleware.RequireAuth(), authMiddleware.RequireAdmin(), categoryHandler.Delete)
		}

		// Tags routes
		tags := api.Group("/tags")
		{
			tags.GET("", tagHandler.List)
			tags.POST("", authMiddleware.RequireAuth(), authMiddleware.RequireAdmin(), tagHandler.Create)
			tags.DELETE("/:id", authMiddleware.RequireAuth(), authMiddleware.RequireAdmin(), tagHandler.Delete)
		}

		// Collections routes (authenticated)
		collections := api.Group("/collections", authMiddleware.RequireAuth())
		{
			collections.GET("", collectionHandler.List)
			collections.POST("", collectionHandler.Create)
			collections.POST("/:id/models", collectionHandler.AddModel)
			collections.DELETE("/:id/models/:modelId", collectionHandler.RemoveModel)
			collections.DELETE("/:id", collectionHandler.Delete)
		}

		// Admin routes
		admin := api.Group("/admin", authMiddleware.RequireAuth(), authMiddleware.RequireAdmin())
		{
			// Settings
			admin.GET("/settings", adminHandler.GetSettings)
			admin.PUT("/settings", adminHandler.UpdateSetting)
			admin.GET("/settings/model-path", adminHandler.GetModelPath)
			admin.PUT("/settings/model-path", adminHandler.SetModelPath)

			// User management
			admin.GET("/users", adminHandler.ListUsers)
			admin.POST("/users", adminHandler.CreateUser)
			admin.PUT("/users/:id", adminHandler.UpdateUser)
			admin.DELETE("/users/:id", adminHandler.DeleteUser)

			// Database management
			admin.POST("/clear-database", adminHandler.ClearDatabase)
			admin.POST("/clear-models", adminHandler.ClearModels)

			// Scanner
			admin.POST("/scan", adminHandler.StartScan)
			admin.GET("/scan/status", adminHandler.GetScanStatus)
			admin.POST("/scan/stop", adminHandler.StopScan)

			// Duplicates
			admin.POST("/duplicates/detect", adminHandler.DetectDuplicates)
			admin.GET("/duplicates", adminHandler.ListDuplicates)
			admin.POST("/duplicates/:id/resolve", adminHandler.ResolveDuplicate)

			// Stats
			admin.GET("/stats", adminHandler.GetStats)
		}
	}

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	return r
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

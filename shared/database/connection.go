package database

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"forgecrud-backend/shared/config"
	"forgecrud-backend/shared/database/models"
	"forgecrud-backend/shared/database/models/auth"
	"forgecrud-backend/shared/database/models/document"
	"forgecrud-backend/shared/database/models/notification"
)

var DB *gorm.DB

// getLogLevel returns appropriate log level based on environment
func getLogLevel(cfg *config.Config) logger.LogLevel {
	if cfg.DBHost == "localhost" || cfg.DBHost == "127.0.0.1" {
		return logger.Warn
	}
	return logger.Error
}

// InitDatabase initializes the database connection and runs migrations
func InitDatabase() error {
	cfg := config.GetConfig()

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
		cfg.DBHost,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
		cfg.DBPort,
		cfg.DBSSLMode,
	)

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(getLogLevel(cfg)),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	// Connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("✅ Database connection established successfully")

	// Run migrations
	if err := runMigrations(); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	return nil
}

// runMigrations runs all database migrations
func runMigrations() error {
	log.Println("🔄 Checking database schema...")

	modelsToMigrate := []interface{}{
		&models.Organization{},
		&models.User{},
		&models.Role{},
		&models.Resource{},
		&models.Action{},
		&models.Permission{},
		&models.PermissionAction{},
		&auth.UserSession{},
		&auth.PasswordResetToken{},
		&auth.PasswordResetAttempt{},
		&auth.EmailVerificationToken{},
		&auth.LoginAttempt{},
		&auth.BlacklistedToken{},
		&notification.AuditLog{},
		&notification.Notification{},
		&document.Folder{},
		&document.Document{},
		&document.DocumentVersion{},
	}

	// Check if all tables exist
	migrator := DB.Migrator()
	allTablesExist := true

	for _, model := range modelsToMigrate {
		if !migrator.HasTable(model) {
			allTablesExist = false
			break
		}
	}

	// If all tables exist, skip migration
	if allTablesExist {
		log.Println("✅ Database schema is up to date - skipping migration")
		return nil
	}

	// Auto migrate all models
	migratedCount := 0
	for _, model := range modelsToMigrate {
		tableName := DB.NamingStrategy.TableName(fmt.Sprintf("%T", model)[1:])

		if !migrator.HasTable(model) {
			log.Printf("📦 Creating table: %s", tableName)
			migratedCount++
		}

		if err := DB.AutoMigrate(model); err != nil {
			return fmt.Errorf("failed to migrate %T: %w", model, err)
		}
	}

	if migratedCount > 0 {
		log.Printf("✅ Database migrations completed (%d tables created/updated)", migratedCount)
	} else {
		log.Println("✅ Database schema is up to date")
	}

	return nil
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	return DB
}

// CloseDatabase closes the database connection
func CloseDatabase() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

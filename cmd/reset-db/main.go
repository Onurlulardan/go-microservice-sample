package main

import (
	"log"

	"forgecrud-backend/shared/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	log.Println("🗑️ Starting database reset...")

	config.LoadConfig()
	cfg := config.GetConfig()

	dsn := "host=" + cfg.DBHost +
		" user=" + cfg.DBUser +
		" password=" + cfg.DBPassword +
		" dbname=" + cfg.DBName +
		" port=" + cfg.DBPort +
		" sslmode=" + cfg.DBSSLMode +
		" TimeZone=UTC"

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatal("❌ Database connection failed:", err)
	}

	// Tüm tabloları listele ve sil
	tables := []string{
		"user_sessions",
		"login_attempts",
		"password_reset_tokens",
		"email_verification_tokens",
		"permission_actions",
		"permissions",
		"users",
		"roles",
		"organizations",
		"actions",
		"resources",
		"documents",
		"document_versions",
		"folders",
		"notifications",
		"audit_logs",
		"blacklisted_tokens",
		"password_reset_attempts",
	}

	log.Println("🗑️ Dropping all tables...")

	for _, table := range tables {
		log.Printf("   Dropping table: %s", table)
		db.Exec("DROP TABLE IF EXISTS " + table + " CASCADE;")
	}

	log.Println("✅ Database reset completed - all tables dropped!")
	log.Println("💡 Run 'make seed' to recreate tables and seed data")
}

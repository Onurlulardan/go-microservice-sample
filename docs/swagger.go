// Package docs ForgeCRUD API documentation
package docs

// Swagger documentation info
// @title ForgeCRUD API
// @version 1.0
// @description Central API documentation - For all ForgeCRUD microservices
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.forgecrud.com/support
// @contact.email support@forgecrud.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8000
// @BasePath /api
// @schemes http https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and the JWT token.

// Auth Service Endpoints
// @tag.name auth
// @tag.description Authentication and user session management

// Core Service Endpoints
// @tag.name users
// @tag.description User management
// @tag.name roles
// @tag.description Role management
// @tag.name organizations
// @tag.description Organization management

// Permission Service Endpoints
// @tag.name permissions
// @tag.description Permission management
// @tag.name resources
// @tag.description Resource management
// @tag.name actions
// @tag.description Action management

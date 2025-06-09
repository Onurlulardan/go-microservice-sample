# ForgeCRUD Backend - Microservices Architecture

```
  _____                      _____  _____   _    _  _____
 |  ___|                    /  __ \|  _  \ | |  | ||  _  |
 | |_  ___  _ __  __ _  ___ | /  \/| |_| | | |  | || | | |
 | _|/ _ \| '__|/ _` |/ _ \| |    |    /  | |  | || | | |
 | | | (_) | |  | (_| |  __/| \__/\| |\ \  | |__| || |/ /
 \_|  \___/|_|   \__, |\___|  \____/\_| \_|  \____/ |___/
                  __/ |
                 |___/
```

**Modern, scalable backend API system built with microservices architecture.**

## 🏗️ Architecture Overview

```
Frontend → API Gateway → [Auth | Permission | Core | Notification | Document] Services → PostgreSQL
          (8000)       (8001)  (8002)     (8003)    (8004)        (8005)                + MinIO
```


![screencapture-mermaid-ink-svg-pako-eNq9WOFu2zYQfhWCQIsNcBLZsmzLGwp4SeYGqJegalFg837QEm1zlUSVpJK4cd59J8mWHVWzSCFYAgSxfPfd3XfH0x2fsM8Disd4JUiyRp-u5vE8RvDjdvv2X3P8WVLx60K8u35UVMQkRBNfcTHHf5dSfZC6-g1NgojFyPMFS5TMNKa8lJLpooB3u10HxEswbyMVjWQpuIPsg](https://github.com/user-attachments/assets/603a13bc-f30d-4564-917e-f05eb69de6c6)


### **Docker Compose Architecture**

**All microservices run in Docker containers:**

- **PostgreSQL Database** - Main database (shared by all services)
- **MinIO Object Storage** - File storage (for Document Service)
- **API Gateway** - Port 8000 - Main entry point
- **Auth Service** - Port 8001 - Authentication
- **Permission Service** - Port 8002 - Authorization
- **Core Service** - Port 8003 - Business logic
- **Notification Service** - Port 8004 - Notifications
- **Document Service** - Port 8005 - File management

**Container Network:**

- All services run in the same Docker network
- Inter-service communication uses container names
- Only API Gateway is exposed to external world (port 8000)

## 🚀 Services

### 1. **API Gateway** _(Port: 8000)_

- **Central entry point** - All client requests pass through here
- **Route Management** - Routes requests to appropriate services
- **Authentication** - JWT token validation
- **Authorization** - Permission-based access control
- **Rate Limiting** - Global IP-based request throttling
- **CORS** - Frontend integration support
- **Unified Response** - Standardizes all API responses with metadata
- **Real-time Notifications** - WebSocket integration for live updates

**Endpoint Examples:**

```bash
POST /api/auth/login          # Proxy to Auth Service
GET  /api/users               # Proxy to Core Service (permission required)
GET  /api/permissions         # Proxy to Permission Service (admin only)
```

### 2. **Auth Service** _(Port: 8001)_

- **User authentication** system
- **JWT token** generation and validation
- **Session management** - Active session tracking
- **Login history** - Login attempt tracking
- **Password hashing** - bcrypt secure password storage
- **Built-in rate limiting** - Login/register attempt protection

**Main Endpoints:**

```bash
# Core Authentication
POST /api/auth/login          # User login
POST /api/auth/register       # New user registration
POST /api/auth/logout         # Session logout

# JWT Token Management
POST /api/auth/refresh        # Refresh JWT token
POST /api/auth/validate       # Validate JWT token
POST /api/auth/blacklist      # Blacklist JWT token

# Email Verification
POST /api/auth/send-verification      # Send email verification link
GET  /api/auth/verify-email/:token    # Verify email with token
POST /api/auth/resend-verification    # Resend verification email

# Password Management
POST /api/auth/change-password        # Change current password
POST /api/auth/forgot-password        # Send password reset email
POST /api/auth/reset-password         # Reset password with token

# Session & Security Management
GET  /api/auth/sessions               # List active sessions
DELETE /api/auth/sessions/:id         # Terminate specific session
DELETE /api/auth/sessions             # Terminate all other sessions
GET  /api/auth/login-history          # Get login history

# Health & Test
GET  /health                          # Service health check
GET  /api/auth/test                   # Test endpoint
```

### 3. **Permission Service** _(Port: 8002)_

- **RBAC (Role-Based Access Control)** system
- **Granular permissions** - Resource + Action based permissions
- **Dynamic authorization** - Runtime permission checks
- **Hierarchical permissions** - User > Role > Organization levels

**Permission Structure:**

```
User Permission → Role Permission → Organization Permission
```

**Main Endpoints:**

```bash
# Permission Management
GET  /api/permissions         # All permissions (with pagination)
POST /api/permissions         # Create new permission
GET  /api/permissions/:id     # Get specific permission
PUT  /api/permissions/:id     # Update permission
DELETE /api/permissions/:id   # Delete permission

# Resource Management
GET  /api/permissions/resources      # All resources (with pagination)
POST /api/permissions/resources      # Create new resource
GET  /api/permissions/resources/:id  # Get specific resource
PUT  /api/permissions/resources/:id  # Update resource
DELETE /api/permissions/resources/:id # Delete resource

# Action Management
GET  /api/permissions/actions        # All actions (with pagination)
POST /api/permissions/actions        # Create new action
GET  /api/permissions/actions/:id    # Get specific action
PUT  /api/permissions/actions/:id    # Update action
DELETE /api/permissions/actions/:id  # Delete action

# Permission Checks (for middleware)
POST /api/permissions/check          # Single permission check
POST /api/permissions/batch-check    # Multiple permissions check

# Cache Management
GET  /api/permissions/cache/stats                     # Cache statistics
POST /api/permissions/cache/invalidate/:user_id       # Clear user cache
POST /api/permissions/cache/invalidate/role/:role_id  # Clear role cache
POST /api/permissions/cache/invalidate/org/:org_id    # Clear org cache
POST /api/permissions/cache/invalidate/all            # Clear all cache
```

### 4. **Core Service** _(Port: 8003)_

- **Business logic** and **data management**
- **User/Role/Organization** CRUD operations
- **Database** interactions
- **Data validation** and **business rules**

**Main Endpoints:**

```bash
# User Management
GET    /api/users                  # User list (pagination + filtering)
GET    /api/users/:id              # Single User
POST   /api/users                  # Create user
PUT    /api/users/:id              # Update user
DELETE /api/users/:id              # Delete user
GET    /api/users/:id/permissions  # User permissions

# Role Management
GET    /api/roles                  # Role list
GET    /api/roles/:id              # Single Role
POST   /api/roles                  # Create role
PUT    /api/roles/:id              # Update role
DELETE /api/roles/:id              # Delete role
GET    /api/roles/:id/permissions  # role permissions


# Organization Management
GET    /api/organizations                  # Organization list
GET    /api/organization/:id               # Single Organization
POST   /api/organizations                  # Create organization
PUT    /api/organizations/:id              # Update organization
DELETE /api/organizations/:id              # Delete organization
GET    /api/organizations/:id/permissions  # organizations permissions
```

### 5. **Notification Service** _(Port: 8004)_

- **Email notifications** - SMTP email sending with templates
- **Real-time notifications** - WebSocket connections for live updates
- **Unified response transformation** - Standardizes all API responses
- **Audit logging** - Request/response tracking and performance metrics
- **Template management** - HTML email templates with localization

**Main Endpoints:**

```bash
# Email Management
POST /api/notifications/email/send                # Send generic email
POST /api/notifications/email/welcome             # Send welcome/verification email
POST /api/notifications/email/password-reset      # Send password reset email
POST /api/notifications/email/verification        # Send email verification
POST /api/notifications/email/resend-verification # Resend verification email

# Notification Management
GET    /api/notifications                 # Get user notifications (with pagination)
GET    /api/notifications/:id             # Get specific notification
POST   /api/notifications                 # Create new notification
PUT   /api/notifications/:id/read        # Mark notification as read
DELETE /api/notifications/:id             # Delete notification

# WebSocket Real-time
WS  /ws/notifications/:user_id            # WebSocket connection for real-time updates
POST /ws/send                             # Send WebSocket message (internal API)

# Health Check
GET /health                               # Service health status
```

### 6. **Document Service** _(Port: 8005)_

- **File management** system with MinIO integration
- **Folder hierarchy** - Nested folder structure with path management
- **Document versioning** - Multiple versions of same document
- **File operations** - Upload, download, move, copy, delete
- **ZIP archiving** - Download folders as compressed archives
- **Storage integration** - MinIO object storage backend

**Main Endpoints:**

```bash
# Folder Management
GET    /api/folders                    # List folders (pagination + filtering)
GET    /api/folders/:id                # Get specific folder
GET    /api/folders/:id/contents       # Get folder contents (subfolders + documents)
POST   /api/folders                    # Create new folder
PUT    /api/folders/:id                # Update folder name
POST   /api/folders/:id/move           # Move folder to different parent
DELETE /api/folders/:id                # Delete empty folder
GET    /api/folders/:id/download       # Download folder as ZIP archive

# Document Management
POST   /api/documents                  # Upload new document
GET    /api/documents                  # List documents in folder
GET    /api/documents/:id              # Get document details
GET    /api/documents/:id/download     # Download document file
PUT    /api/documents/:id              # Update document metadata
POST   /api/documents/:id/move         # Move document to different folder
DELETE /api/documents/:id              # Delete document
POST   /api/documents/:id/copy         # Copy document to another folder

# Document Versions
GET    /api/documents/:id/versions            # Get all document versions
GET    /api/documents/:id/versions/latest     # Get latest version
POST   /api/documents/:id/versions            # Upload new version

# Health Check
GET    /health                         # Service health status
```

## 🛡️ Security & Rate Limiting

### **Authentication Flow:**

1. **Login** → Auth Service generates JWT token
2. **Request** → API Gateway validates token
3. **Authorization** → Permission Service checks permissions
4. **Proxy** → Routes request to appropriate service

### **Rate Limiting:**

**Global Rate Limiting (API Gateway):**

- **IP-based** request throttling for all endpoints
- **Configurable limits** via environment variables
- **Automatic cleanup** of old rate limit records
- **Block duration** for exceeding limits

**Auth-specific Rate Limiting (Auth Service):**

- **Login attempts:** Separate rate limiting for authentication
- **Registration:** Protection against spam registrations
- **Password reset:** Prevents abuse of reset functionality

**Environment Configuration:**

```env
# Global Rate Limiting
RATE_LIMIT_MAX_REQUESTS=100          # 100 requests per time window
RATE_LIMIT_TIME_WINDOW_SECONDS=60    # 60 second time window
RATE_LIMIT_BLOCK_DURATION_MINUTES=15 # 15 minute block duration
```

### **Permission Levels:**

```
1. USER LEVEL     → Direct user permissions
2. ROLE LEVEL     → Role-inherited permissions
3. ORG LEVEL      → Organization-inherited permissions
```

## 🗃️ Database

**PostgreSQL** with shared database model:

- **UUID** primary keys
- **GORM** ORM usage
- **Automatic migrations**

### Main Tables:

- `users` - User information
- `roles` - Role definitions
- `organizations` - Organization structure
- `permissions` - Permission records
- `resources` - System resources
- `actions` - Available actions
- `user_sessions` - Active sessions
- `notifications` - Real-time notifications
- `audit_logs` - Request/response audit trail

## 🚀 Quick Start

### 1. **Environment Setup**

```bash
cp .env.example .env
# Configure environment variables required to run all services with Docker Compose
```

### 2. **Start with Docker Compose**

```bash
# Start all services with Docker Compose (PostgreSQL + All Microservices)
make docker-up

# Or build and start
make docker-rebuild
```

### 3. **Database Setup & Seeding**

```bash
# Database migration and seed data (Docker environment)
make docker-fresh

# Or for local development
make fresh          # Database reset + seed data
make seed          # Seed data only
```

### 4. **Test**

```bash
# Login
curl -X POST http://localhost:8000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@forgecrud.com","password":"admin123"}'

# List users (token required)
curl -H "Authorization: Bearer YOUR_TOKEN" \
  http://localhost:8000/api/users
```

## 📄 Query Parameter Standardization

All list endpoints across the system implement standardized query parameters for **filtering**, **sorting**, **searching**, and **pagination** that are compatible with Ant Design Table components on the frontend.

### **Query Parameters**

| Feature        | Format                                         | Example                                   | Description                                 |
| -------------- | ---------------------------------------------- | ----------------------------------------- | ------------------------------------------- |
| **Filtering**  | `filters[field_name]=value`                    | `filters[status]=ACTIVE`                  | Filter records by field values              |
| **Sorting**    | `sort[field]=field_name&sort[order]=asc\|desc` | `sort[field]=created_at&sort[order]=desc` | Sort by field in ascending/descending order |
| **Searching**  | `search=term`                                  | `search=john`                             | Search across predefined fields             |
| **Pagination** | `page=n&limit=m`                               | `page=1&limit=10`                         | Control page number and items per page      |

### **Response Structure**

All API responses follow a unified format with consistent structure and metadata:

```json
{
  "success": true,
  "message": "Operation completed successfully",
  "data": {
    "items": [
      // Array of items for current page
    ],
    "pagination": {
      "page": 1, // Current page number
      "limit": 10, // Items per page
      "total": 45, // Total items across all pages
      "total_pages": 5, // Total number of pages
      "has_next": true, // Whether next page exists
      "has_prev": false // Whether previous page exists
    }
  },
  "meta": {
    "request_id": "req_123",
    "timestamp": "2025-06-03T10:30:00Z",
    "execution_time": "120ms"
  }
}
```

**Error Response Example:**

```json
{
  "success": false,
  "message": "Validation failed",
  "error": {
    "code": "VALIDATION_ERROR",
    "details": "Email field is required"
  },
  "meta": {
    "request_id": "req_124",
    "timestamp": "2025-06-03T10:30:05Z",
    "execution_time": "15ms"
  }
}
```

### **Implementation with the Shared Query Utility**

The `shared/utils/query` package provides:

- `ParseQueryParams()` - Extracts filter, sort, search, and pagination parameters
- `ApplyFilters()` - Applies field filters to GORM query
- `ApplySearch()` - Performs ILIKE search across fields
- `ApplySort()` - Applies sorting to query
- `ApplyPagination()` - Applies limit and offset
- `BuildPaginationResponse()` - Builds consistent pagination metadata

### **Example API Calls**

**Multiple Filters:**

```
GET /api/users?filters[status]=ACTIVE&filters[organization_id]=12345&page=1&limit=10
```

**Combined Search and Sort:**

```
GET /api/users?search=john&sort[field]=created_at&sort[order]=desc&page=2&limit=5
```

## 🐳 Docker Development Environment

### **Container Management**

```bash
# Start all containers
make docker-up

# Restart specific container
make docker-restart-service SERVICE=api-gateway

# Check container status
make docker-status

# Connect to running containers
docker-compose exec api-gateway sh
docker-compose exec postgres psql -U postgres -d forgecrud
```

### **Volume Management**

```bash
# Delete PostgreSQL data (for fresh start)
make docker-clean

# Only stop containers, keep volumes
make docker-down
```

### **Environment Variables**

**Important environment variables for Docker Compose:**

```env
# Database
POSTGRES_HOST=postgres          # Container name
POSTGRES_PORT=5432
POSTGRES_DB=forgecrud
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres123

# MinIO
MINIO_ENDPOINT=minio:9000      # Container name
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin

# Service URLs (container network)
AUTH_SERVICE_URL=http://auth-service:8001
PERMISSION_SERVICE_URL=http://permission-service:8002
CORE_SERVICE_URL=http://core-service:8003
```

### **Troubleshooting**

```bash
# Clean all containers and restart
make docker-clean
make docker-up

# For database connection issues
make docker-logs-service SERVICE=postgres

# Detailed service log inspection
make docker-logs-service SERVICE=api-gateway
```

**Date Range Filter (Auth Service):**

```
GET /api/auth/login-history?filters[from_date]=2025-05-01&filters[to_date]=2025-06-01
```

### **Service Endpoints with Standardized Query Parameters**

- **Core Service:** `/api/users`, `/api/roles`, `/api/organizations`
- **Permission Service:** `/api/permissions`, `/api/permissions/resources`, `/api/permissions/actions`
- **Auth Service:** `/api/auth/sessions`, `/api/auth/login-history`
- **Notification Service:** `/api/notifications`

## 🔧 Development Commands

### **Local Development:**

```bash
make dev        # Start all services locally
make stop       # Stop all local services
make status     # Check health of all services
```

### **Database:**

```bash
make fresh      # Reset DB + seed data
make seed       # Add seed data only
make reset-db   # Reset database structure only
```

### **Docker:**

```bash
make docker-build              # Build Docker images
make docker-up                 # Start all containers
make docker-down               # Stop all containers
make docker-rebuild            # Rebuild and restart
make docker-logs               # Follow all logs
make docker-logs-service       # Follow specific service logs
make docker-status             # Check container status
make docker-restart            # Restart all containers
make docker-restart-service    # Restart specific container
make docker-clean              # Clean volumes and restart fresh
make docker-fresh              # Fresh database seed in Docker
```

### **Utility:**

```bash
make swagger    # Generate Swagger documentation
make clean      # Clean temporary files
```

## 🏢 Service Ports

| Service              | Port | Description                     |
| -------------------- | ---- | ------------------------------- |
| API Gateway          | 8000 | Main entry point                |
| Auth Service         | 8001 | Authentication                  |
| Permission Service   | 8002 | Permission management           |
| Core Service         | 8003 | Business logic                  |
| Notification Service | 8004 | Real-time & Email notifications |
| Document Service     | 8005 | File & folder management        |

## 🔗 Technology Stack

- **Go** (Golang) - Backend language
- **Gin** - HTTP web framework
- **GORM** - ORM
- **PostgreSQL** - Database
- **MinIO** - Object storage
- **JWT** - Authentication
- **bcrypt** - Password hashing
- **UUID** - Primary keys
- **Redis** - Session storage

## 📝 Features

✅ **Microservices Architecture**  
✅ **JWT Authentication**  
✅ **Role-Based Access Control (RBAC)**  
✅ **Granular Permissions**  
✅ **Global Rate Limiting**  
✅ **Auth-specific Rate Limiting**  
✅ **Standardized Pagination**  
✅ **Session Management**  
✅ **Login History Tracking**  
✅ **CORS Support**  
✅ **Standardized API Responses**  
✅ **Database Migrations**  
✅ **Seed Data System**  
✅ **Environment-based Configuration**  
✅ **Email Notifications** - SMTP email system with templates  
✅ **Real-time Notifications** - WebSocket connections  
✅ **Unified Response Format** - Consistent API responses  
✅ **Audit Logging** - Request/response tracking
✅ **File Management System** - Upload, download, organize files  
✅ **Folder Hierarchy** - Nested folder structure with path tracking  
✅ **Document Versioning** - Multiple versions with history  
✅ **ZIP Archive Download** - Recursive folder compression  
✅ **MinIO Integration** - Scalable object storage backend

---

**This microservices architecture follows modern best practices for scalable, secure, and maintainable backend systems.**

package database

import (
	"fmt"
	"log"
	"time"

	"forgecrud-backend/shared/config"
	"forgecrud-backend/shared/database/models"
	utils "forgecrud-backend/shared/utils/auth"

	"github.com/google/uuid"
)

// SeedDatabase seeds the database with initial data
func SeedDatabase() error {
	log.Println("🌱 Checking database seed data...")

	// Seed Resources
	resourcesCreated, err := seedResources()
	if err != nil {
		return err
	}

	// Seed Actions
	actionsCreated, err := seedActions()
	if err != nil {
		return err
	}

	// Seed Default Roles
	rolesCreated, err := seedDefaultRoles()
	if err != nil {
		return err
	}

	if resourcesCreated > 0 || actionsCreated > 0 || rolesCreated > 0 {
		log.Printf("✅ Database seeding completed (%d resources, %d actions, %d roles created)", resourcesCreated, actionsCreated, rolesCreated)
	} else {
		log.Println("✅ Database seed data is up to date")
	}

	// Create super admin from config
	if err := CreateSuperAdminFromConfig(); err != nil {
		return err
	}

	// Seed super admin permissions (wildcard permissions)
	permissionsCreated, err := seedSuperAdminPermissions()
	if err != nil {
		return err
	}

	if permissionsCreated > 0 {
		log.Printf("✅ Super admin permissions created: %d wildcard permissions", permissionsCreated)
	}

	return nil
}

// seedResources creates default resources
func seedResources() (int, error) {
	resources := []models.Resource{
		{Name: "All Resources", Slug: "ALL", Description: "Wildcard access to all resources", IsSystem: true},
		{Name: "Users", Slug: "users", Description: "User management", IsSystem: true},
		{Name: "Organizations", Slug: "organizations", Description: "Organization management", IsSystem: true},
		{Name: "Roles", Slug: "roles", Description: "Role management", IsSystem: true},
		{Name: "Permissions", Slug: "permissions", Description: "Permission management", IsSystem: true},
		{Name: "Notifications", Slug: "notifications", Description: "Notification management", IsSystem: true},
		{Name: "Forms", Slug: "forms", Description: "Dynamic form management", IsSystem: true},
		{Name: "Dashboard", Slug: "dashboard", Description: "Dashboard access", IsSystem: true},
		{Name: "Security Logs", Slug: "security-logs", Description: "Security log access", IsSystem: true},
		{Name: "File management", Slug: "file-management", Description: "File management", IsSystem: true},
		{Name: "Documents", Slug: "documents", Description: "Document management", IsSystem: true},
		{Name: "Folders", Slug: "folders", Description: "Folder management", IsSystem: true},
	}

	created := 0
	for _, resource := range resources {
		var existing models.Resource
		result := DB.Where("slug = ?", resource.Slug).First(&existing)
		if result.Error != nil {
			if err := DB.Create(&resource).Error; err != nil {
				return created, err
			}
			created++
		}
	}

	return created, nil
}

// seedActions creates default actions
func seedActions() (int, error) {
	actions := []models.Action{
		{Name: "Create", Slug: "create", Description: "Create new records", IsSystem: true},
		{Name: "Read", Slug: "read", Description: "View/read records", IsSystem: true},
		{Name: "Update", Slug: "update", Description: "Update existing records", IsSystem: true},
		{Name: "Delete", Slug: "delete", Description: "Delete records", IsSystem: true},
		{Name: "Export", Slug: "export", Description: "Export data", IsSystem: false},
		{Name: "Import", Slug: "import", Description: "Import data", IsSystem: false},
		{Name: "Manage", Slug: "manage", Description: "Full management access", IsSystem: true},
	}

	created := 0
	for _, action := range actions {
		var existing models.Action
		result := DB.Where("slug = ?", action.Slug).First(&existing)
		if result.Error != nil {
			if err := DB.Create(&action).Error; err != nil {
				return created, err
			}
			created++
		}
	}

	return created, nil
}

// seedDefaultRoles creates default roles for organizations
func seedDefaultRoles() (int, error) {
	var superAdminOrg models.Organization
	if err := DB.Where("slug = ?", "super-admin").First(&superAdminOrg).Error; err != nil {
		return 0, nil
	}

	defaultRoles := []models.Role{
		{
			Name:           "Admin",
			Description:    "Organization administrator with full access",
			IsDefault:      true,
			OrganizationID: &superAdminOrg.ID,
		},
		{
			Name:           "User",
			Description:    "Standard user with limited access",
			IsDefault:      false,
			OrganizationID: &superAdminOrg.ID,
		},
		{
			Name:           "Manager",
			Description:    "Manager with moderate access",
			IsDefault:      false,
			OrganizationID: &superAdminOrg.ID,
		},
	}

	created := 0
	for _, role := range defaultRoles {
		var existing models.Role
		result := DB.Where("name = ? AND organization_id = ?", role.Name, role.OrganizationID).First(&existing)
		if result.Error != nil {
			if err := DB.Create(&role).Error; err != nil {
				return created, err
			}
			created++
		}
	}

	return created, nil
}

// CreateSuperAdminFromConfig creates super admin using config values
func CreateSuperAdminFromConfig() error {
	cfg := config.GetConfig()
	return CreateSuperAdmin(cfg.SuperAdminEmail, cfg.SuperAdminPassword, "Super", "Admin")
}

// CreateSuperAdmin creates a super admin organization and user
func CreateSuperAdmin(email, password, firstName, lastName string) error {
	var existingUser models.User
	if err := DB.Where("email = ?", email).First(&existingUser).Error; err == nil {
		log.Println("Super admin already exists")
		return nil
	}

	var superAdminOrg models.Organization
	err := DB.Where("slug = ?", "super-admin").First(&superAdminOrg).Error
	if err != nil {
		superAdminOrg = models.Organization{
			Name:      "Super Admin Organization",
			Slug:      "super-admin",
			Status:    "ACTIVE",
			OwnerID:   uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := DB.Create(&superAdminOrg).Error; err != nil {
			return err
		}
	}

	// Check if super admin role already exists
	var superAdminRole models.Role
	err = DB.Where("name = ? AND organization_id = ?", "Super Admin", superAdminOrg.ID).First(&superAdminRole).Error
	if err != nil {
		superAdminRole = models.Role{
			Name:           "Super Admin",
			Description:    "Full system access",
			IsDefault:      false,
			OrganizationID: &superAdminOrg.ID,
		}

		if err := DB.Create(&superAdminRole).Error; err != nil {
			return err
		}
	}

	// Hash password before creating user
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return err
	}

	// Create super admin user with proper references
	superAdminUser := models.User{
		Email:          email,
		Password:       hashedPassword,
		FirstName:      firstName,
		LastName:       lastName,
		Status:         "ACTIVE",
		EmailVerified:  true,
		OrganizationID: &superAdminOrg.ID,
		RoleID:         &superAdminRole.ID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := DB.Create(&superAdminUser).Error; err != nil {
		return err
	}

	// Update organization owner to actual user ID
	superAdminOrg.OwnerID = superAdminUser.ID
	DB.Save(&superAdminOrg)

	log.Printf("✅ Super admin created: %s", email)
	return nil
}

// seedSuperAdminPermissions creates wildcard permissions for super admin
func seedSuperAdminPermissions() (int, error) {
	// First check if super admin role exists
	var superAdminRole models.Role
	if err := DB.Where("name = ?", "Super Admin").First(&superAdminRole).Error; err != nil {
		log.Println("⚠️  Super admin role not found, skipping permission seeding")
		return 0, nil
	}

	// Get ALL resource (wildcard resource)
	var allResource models.Resource
	if err := DB.Where("slug = ?", "ALL").First(&allResource).Error; err != nil {
		return 0, fmt.Errorf("ALL resource not found: %v", err)
	}

	// Get all actions
	var actions []models.Action
	if err := DB.Find(&actions).Error; err != nil {
		return 0, fmt.Errorf("failed to fetch actions: %v", err)
	}

	// Check if permission already exists for ALL resource
	var existingPermission models.Permission
	result := DB.Where("resource_id = ? AND target = ? AND role_id = ?",
		allResource.ID, "ROLE", superAdminRole.ID).First(&existingPermission)

	var permission models.Permission
	permissionExists := (result.Error == nil)
	createdCount := 0

	if !permissionExists {
		// Create new permission for ALL resource
		permission = models.Permission{
			ResourceID: allResource.ID,
			Target:     "ROLE",
			RoleID:     &superAdminRole.ID,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		if err := DB.Create(&permission).Error; err != nil {
			return 0, fmt.Errorf("failed to create ALL permission: %v", err)
		}
		createdCount = 1
		log.Printf("✅ Created super admin ALL resource permission")
	} else {
		permission = existingPermission
		log.Println("✅ Super admin ALL resource permission already exists")
	}

	// Now create permission actions for all actions if they don't exist
	actionsCreated := 0
	for _, action := range actions {
		var existingPermissionAction models.PermissionAction
		actionResult := DB.Where("permission_id = ? AND action_id = ?",
			permission.ID, action.ID).First(&existingPermissionAction)

		if actionResult.Error == nil {
			// Permission action already exists, skip
			continue
		}

		// Create new permission action
		permissionAction := models.PermissionAction{
			PermissionID: permission.ID,
			ActionID:     action.ID,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		if err := DB.Create(&permissionAction).Error; err != nil {
			return createdCount, fmt.Errorf("failed to create permission action for ALL resource, action %s: %v",
				action.Name, err)
		}
		actionsCreated++
	}

	if createdCount > 0 {
		log.Printf("✅ Created super admin ALL permission with %d actions", len(actions))
	} else {
		log.Printf("✅ Super admin ALL permission is up to date with %d actions", actionsCreated)
	}

	return createdCount, nil
}

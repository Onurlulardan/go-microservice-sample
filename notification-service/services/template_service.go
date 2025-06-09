package services

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"sync"

	"forgecrud-backend/shared/config"
)

// TemplateService handles rendering of email templates
type TemplateService struct {
	config        *config.Config
	templateCache map[string]*template.Template
	templateDir   string
	templateMutex sync.RWMutex
}

// NewTemplateService creates a new template service
func NewTemplateService(cfg *config.Config) *TemplateService {
	return &TemplateService{
		config:        cfg,
		templateCache: make(map[string]*template.Template),
		templateDir:   "./shared/mail_templates", // Default template location
	}
}

// RenderTemplate renders an email template with provided data
func (ts *TemplateService) RenderTemplate(templateID string, data map[string]interface{}) (string, error) {
	// Check if template is in cache
	ts.templateMutex.RLock()
	tmpl, exists := ts.templateCache[templateID]
	ts.templateMutex.RUnlock()

	if !exists {
		// Load template from file
		filename := ts.getTemplateFilename(templateID)
		templatePath := filepath.Join(ts.templateDir, filename)

		// Check if file exists
		if _, err := os.Stat(templatePath); os.IsNotExist(err) {
			return "", fmt.Errorf("template file not found: %s", templatePath)
		}

		// Parse template
		var err error
		tmpl, err = template.ParseFiles(templatePath)
		if err != nil {
			return "", fmt.Errorf("failed to parse template %s: %v", templateID, err)
		}

		// Add to cache
		ts.templateMutex.Lock()
		ts.templateCache[templateID] = tmpl
		ts.templateMutex.Unlock()
	}

	// Render template
	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, data); err != nil {
		return "", fmt.Errorf("failed to render template %s: %v", templateID, err)
	}

	return rendered.String(), nil
}

// getTemplateFilename maps template ID to filename
func (ts *TemplateService) getTemplateFilename(templateID string) string {
	switch templateID {
	case "welcome_verification":
		return "welcome_verification.html"
	case "password_reset":
		return "password_reset.html"
	case "critical_error":
		return "critical_error.html"
	case "user_action":
		return "user_action.html"
	case "system_alert":
		return "system_alert.html"
	default:
		log.Printf("Unknown template ID: %s, using as filename", templateID)
		return templateID + ".html"
	}
}

// ReloadTemplate forces reload of a specific template
func (ts *TemplateService) ReloadTemplate(templateID string) error {
	filename := ts.getTemplateFilename(templateID)
	templatePath := filepath.Join(ts.templateDir, filename)

	// Check if file exists
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return fmt.Errorf("template file not found: %s", templatePath)
	}

	// Parse template
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %v", templateID, err)
	}

	// Update cache
	ts.templateMutex.Lock()
	ts.templateCache[templateID] = tmpl
	ts.templateMutex.Unlock()

	return nil
}

// ClearCache clears the template cache
func (ts *TemplateService) ClearCache() {
	ts.templateMutex.Lock()
	ts.templateCache = make(map[string]*template.Template)
	ts.templateMutex.Unlock()
}

package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

type Config struct {
	Port          string
	URLParam      string
	ExpectedValue string
	OverleafURL   string
	AdminEmail    string
	AdminPassword string
	ListenAddr    string
	TemplatesDir  string
	StaticDir     string
	NodePath      string
	ScriptPath    string
}

type PageData struct {
	Message     string
	MessageType string
}

type PuppeteerResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func registerUserWithPuppeteer(config Config, userEmail string) error {
	cmd := exec.Command(config.NodePath,
		config.ScriptPath,
		config.OverleafURL,
		config.AdminEmail,
		config.AdminPassword,
		userEmail,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("puppeteer script failed: %v, output: %s", err, string(output))
	}

	var response PuppeteerResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return fmt.Errorf("failed to parse puppeteer response: %v", err)
	}

	if !response.Success {
		return fmt.Errorf("registration failed: %s", response.Error)
	}

	return nil
}

func main() {
	config := Config{
		Port:          os.Getenv("PORT"),
		URLParam:      os.Getenv("URL_PARAM"),
		ExpectedValue: os.Getenv("EXPECTED_VALUE"),
		OverleafURL:   os.Getenv("OVERLEAF_URL"),
		AdminEmail:    os.Getenv("ADMIN_EMAIL"),
		AdminPassword: os.Getenv("ADMIN_PASSWORD"),
		ListenAddr:    os.Getenv("LISTEN_ADDR"),
		TemplatesDir:  os.Getenv("TEMPLATES_DIR"),
		StaticDir:     os.Getenv("STATIC_DIR"),
		NodePath:      os.Getenv("NODE_PATH"),
		ScriptPath:    os.Getenv("SCRIPT_PATH"),
	}

	// Set defaults
	if config.Port == "" {
		config.Port = "8080"
	}
	if config.ListenAddr == "" {
		config.ListenAddr = ":" + config.Port
	}
	if config.TemplatesDir == "" {
		config.TemplatesDir = "templates"
	}
	if config.StaticDir == "" {
		config.StaticDir = "static"
	}
	if config.NodePath == "" {
		config.NodePath = "node"
	}
	if config.ScriptPath == "" {
		config.ScriptPath = "register.js"
	}

	// Load template
	templatePath := filepath.Join(config.TemplatesDir, "register.html")
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}

	// Load forbidden template
	forbiddenTemplatePath := filepath.Join(config.TemplatesDir, "register-forbidden.html")
	forbiddenTmpl, err := template.ParseFiles(forbiddenTemplatePath)
	if err != nil {
		log.Fatalf("Failed to parse forbidden template: %v", err)
	}

	successTemplatePath := filepath.Join(config.TemplatesDir, "register-success.html")
	successTmpl, err := template.ParseFiles(successTemplatePath)
	if err != nil {
		log.Fatalf("Failed to parse success template: %v", err)
	}

	// Create static file server
	staticFileServer := http.FileServer(http.Dir(config.StaticDir))

	// Handle static files
	http.HandleFunc("/register/static/", func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = r.URL.Path[len("/register/static/"):]
		staticFileServer.ServeHTTP(w, r)
	})

	// Handle registration
	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		// Check URL parameter
		if param := r.URL.Query().Get(config.URLParam); param != config.ExpectedValue {
			forbiddenTmpl.Execute(w, &PageData{
				Message:     "Access forbidden. This server is invite only.",
				MessageType: "error",
			})
			return
		}

		if r.Method == "GET" {
			tmpl.Execute(w, &PageData{})
			return
		}

		if r.Method == "POST" {
			email := r.FormValue("email")
			if email == "" {
				tmpl.Execute(w, &PageData{
					Message:     "Email is required",
					MessageType: "error",
				})
				return
			}

			err := registerUserWithPuppeteer(config, email)
			if err != nil {
				log.Printf("Registration failed: %v", err)
				tmpl.Execute(w, &PageData{
					Message:     "Registration failed. The service might be under maintenance.",
					MessageType: "error",
				})
				return
			}

			// Redirect to success page
			successTmpl.Execute(w, &PageData{
				Message:     "Registration email sent. Please check your email inbox, including the spam folder.",
				MessageType: "success",
			})
			return
		}
	})

	// Handle all other routes
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login", http.StatusFound)
	})

	log.Printf("Server starting on %s", config.ListenAddr)
	log.Fatal(http.ListenAndServe(config.ListenAddr, nil))
}

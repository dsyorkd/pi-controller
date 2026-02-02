package ui

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed dist/*
var content embed.FS

// RegisterRoutes registers the UI routes on the gin engine
func RegisterRoutes(router *gin.Engine) error {
	// Get the dist directory from the embedded filesystem
	distFS, err := fs.Sub(content, "dist")
	if err != nil {
		return err
	}

	// Create a file server for the dist directory
	fileServer := http.FileServer(http.FS(distFS))

	// Serve static files
	router.NoRoute(func(c *gin.Context) {
		// If the request path starts with /api, return 404 because it's an API request
		if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
			c.JSON(http.StatusNotFound, gin.H{"error": "API endpoint not found"})
			return
		}

		// Create a file system that falls back to index.html for SPA routing
		// This checks if the file exists in the embedded FS
		path := c.Request.URL.Path
		if path == "/" {
			path = "index.html"
		} else {
			// Remove leading slash
			path = path[1:]
		}

		_, err := distFS.Open(path)
		if err != nil {
			// File not found, serve index.html for SPA routing
			c.Request.URL.Path = "/"
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}

		// File exists, serve it
		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	return nil
}

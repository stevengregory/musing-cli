package config

import (
	"os"
	"path/filepath"
)

const (
	MongoDevPort  = 27018
	MongoProdPort = 27019
	AngularPort   = 3000
)

// API service configurations
type ServiceConfig struct {
	Name string
	Port int
	Path string // Health check path
}

var APIServices = []ServiceConfig{
	{Name: "networks-api", Port: 8085, Path: "/health"},
	{Name: "random-facts-api", Port: 8082, Path: "/health"},
	{Name: "alcohol-free-api", Port: 8081, Path: "/health"},
	{Name: "random-quotes-api", Port: 8083, Path: "/health"},
	{Name: "news-api", Port: 8084, Path: "/health"},
	{Name: "about-me-api", Port: 8086, Path: "/health"},
	{Name: "featured-item-api", Port: 8087, Path: "/health"},
	{Name: "bitcoin-price-api", Port: 8088, Path: "/health"},
}

// GetAPIRepos returns paths to expected API repositories
func GetAPIRepos() []string {
	home := os.Getenv("HOME")
	return []string{
		filepath.Join(home, "repos", "networks-api"),
		filepath.Join(home, "repos", "random-facts-api"),
		filepath.Join(home, "repos", "alcohol-free-api"),
		filepath.Join(home, "repos", "random-quotes-api"),
		filepath.Join(home, "repos", "news-api"),
		filepath.Join(home, "repos", "about-me-api"),
		filepath.Join(home, "repos", "featured-item-api"),
		filepath.Join(home, "repos", "bitcoin-price-api"),
	}
}

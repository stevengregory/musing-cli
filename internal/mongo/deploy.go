package mongo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Collection represents a MongoDB collection configuration
type Collection struct {
	Name     string
	File     string
	IsArray  bool
}

// Collections maps collection names to their configurations
var Collections = map[string]Collection{
	"networks": {
		Name:    "social_networks",
		File:    "social_networks.json",
		IsArray: false,
	},
	"facts": {
		Name:    "random_facts",
		File:    "random_facts.json",
		IsArray: false,
	},
	"quotes": {
		Name:    "random_quotes",
		File:    "random_quotes.json",
		IsArray: true,
	},
	"news": {
		Name:    "news",
		File:    "news.json",
		IsArray: true,
	},
	"alcoholfree": {
		Name:    "alcohol_free_streak",
		File:    "alcohol-free-streak.json",
		IsArray: true,
	},
	"aboutme": {
		Name:    "about_me",
		File:    "about-me.json",
		IsArray: false,
	},
	"featureditem": {
		Name:    "featured_item",
		File:    "featured-item.json",
		IsArray: true,
	},
	"bitcoinprice": {
		Name:    "bitcoin_price",
		File:    "bitcoin-price.json",
		IsArray: false,
	},
}

// DeployCollection imports a single collection into MongoDB
func DeployCollection(uri, db, collectionKey, dataDir string) error {
	coll, exists := Collections[collectionKey]
	if !exists {
		return fmt.Errorf("unknown collection: %s", collectionKey)
	}

	filePath := filepath.Join(dataDir, coll.File)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("data file not found: %s", filePath)
	}

	args := []string{
		"--uri", uri,
		"--db", db,
		"--collection", coll.Name,
		"--file", filePath,
		"--drop",
	}

	if coll.IsArray {
		args = append(args, "--jsonArray")
	}

	cmd := exec.Command("mongoimport", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// DeployAll imports all collections
func DeployAll(uri, db, dataDir string) error {
	// Order matters - deploy in this sequence
	collectionsOrder := []string{
		"networks",
		"facts",
		"quotes",
		"news",
		"alcoholfree",
		"aboutme",
		"featureditem",
		"bitcoinprice",
	}

	for _, key := range collectionsOrder {
		if err := DeployCollection(uri, db, key, dataDir); err != nil {
			return fmt.Errorf("failed to deploy %s: %w", key, err)
		}
	}

	return nil
}

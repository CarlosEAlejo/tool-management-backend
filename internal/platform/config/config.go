package config

import "os"

type Config struct {
	Port            string
	MongoURI        string
	DatabaseName    string
	CollectionName  string
	FrontendOrigins []string
}

func Load() Config {
	primaryOrigin := getEnv("FRONTEND_ORIGIN", "http://localhost:3000")
	origins := []string{primaryOrigin}
	if primaryOrigin != "http://127.0.0.1:3000" {
		origins = append(origins, "http://127.0.0.1:3000")
	}

	return Config{
		Port:            getEnv("PORT", "8000"),
		MongoURI:        getEnv("MONGODB_URI", "mongodb://localhost:27017"),
		DatabaseName:    getEnv("MONGODB_DB", "herramientas"),
		CollectionName:  getEnv("MONGODB_COLLECTION", "herramientas"),
		FrontendOrigins: origins,
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

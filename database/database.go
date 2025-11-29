// path: database/database.go
package database

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client
var db *mongo.Database

// Connect establishes a singleton MongoDB connection.
func Connect(ctx context.Context) error {
	if client != nil && db != nil {
		return nil
	}

	cfg, reason := resolveConfig()
	if getenv("MONGO_DEBUG", "") != "" {
		log.Printf("mongo: env snapshot %s", envSnapshot())
	}

	start := time.Now()
	log.Printf("mongo: connecting mode=%s uri=%s db=%s (%s)", cfg.Mode, redactURI(cfg.URI), cfg.DBName, reason)

	dctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	c, err := mongo.Connect(dctx, options.Client().ApplyURI(cfg.URI))
	if err != nil {
		return fmt.Errorf("mongo connect: %w", err)
	}
	if err = c.Ping(dctx, nil); err != nil {
		_ = c.Disconnect(context.Background())
		return fmt.Errorf("mongo ping: %w", err)
	}

	client = c
	db = c.Database(cfg.DBName)

	if err := createIndexes(); err != nil {
		log.Printf("mongo: index creation warnings: %v", err)
	}

	log.Printf("mongo: connected ok in %s", time.Since(start).Round(time.Millisecond))
	return nil
}

func Disconnect(ctx context.Context) error {
	if client == nil {
		return nil
	}
	defer func() { client, db = nil, nil }()
	return client.Disconnect(ctx)
}

func Client() *mongo.Client { return client }

func Col(name string) *mongo.Collection {
	if db == nil {
		panic("database not connected: call database.Connect first")
	}
	return db.Collection(name)
}

// --- internal ---

type config struct {
	Mode   string
	URI    string
	DBName string
}

// resolveConfig returns the chosen config and a human-readable reason.
func resolveConfig() (config, string) {
	mode := strings.ToLower(getenv("MONGO_MODE", "auto"))
	dbname := getenv("MONGO_DB", "meniba")

	// Inputs (trimmed)
	explicit := strings.TrimSpace(os.Getenv("MONGO_URI"))
	local := getenv("MONGO_URI_LOCAL", "mongodb://localhost:27017")
	remote := strings.TrimSpace(os.Getenv("MONGO_URI_REMOTE"))
	println(remote)

	switch mode {
	case "local":
		return config{Mode: "local", URI: chooseFirstNonEmpty(explicit, local), DBName: dbname},
			reasonLocal(explicit, local)
	case "remote":
		if remote != "" {
			return config{Mode: "remote", URI: remote, DBName: dbname}, "MONGO_MODE=remote, using MONGO_URI_REMOTE"
		}
		// Fallback with warning
		log.Printf("mongo: WARNING MONGO_MODE=remote but MONGO_URI_REMOTE empty; falling back to local")
		return config{Mode: "local", URI: chooseFirstNonEmpty(explicit, local), DBName: dbname},
			"remote missing → fallback to explicit/local"
	default: // auto
		// NEW precedence: remote > explicit > local
		if remote != "" {
			return config{Mode: "remote", URI: remote, DBName: dbname}, "auto: MONGO_URI_REMOTE present"
		}
		if explicit != "" {
			return config{Mode: "auto", URI: explicit, DBName: dbname}, "auto: MONGO_URI present"
		}
		return config{Mode: "local", URI: local, DBName: dbname}, "auto: fallback to local"
	}
}

func createIndexes() error {
	if db == nil {
		return errors.New("db is nil")
	}
	ctxIdx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var errs []string
	col := Col("reports")

	if _, err := col.Indexes().CreateOne(ctxIdx, mongo.IndexModel{
		Keys: bson.D{{Key: "created_at", Value: -1}},
	}); err != nil {
		errs = append(errs, "created_at: "+err.Error())
	}
	if _, err := col.Indexes().CreateOne(ctxIdx, mongo.IndexModel{
		Keys: bson.D{{Key: "category", Value: 1}},
	}); err != nil {
		errs = append(errs, "category: "+err.Error())
	}
	if _, err := col.Indexes().CreateOne(ctxIdx, mongo.IndexModel{
		Keys: bson.D{{Key: "lat", Value: 1}, {Key: "lng", Value: 1}},
	}); err != nil {
		errs = append(errs, "lat,lng: "+err.Error())
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

// --- utils ---

func redactURI(raw string) string {
	if raw == "" || !strings.Contains(raw, "://") {
		return raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.User == nil {
		return raw
	}
	u.User = url.UserPassword("****", "****")
	return u.String()
}

func getenv(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

func chooseFirstNonEmpty(v1, v2 string) string {
	if strings.TrimSpace(v1) != "" {
		return v1
	}
	return v2
}

func reasonLocal(explicit, local string) string {
	if strings.TrimSpace(explicit) != "" {
		return "MONGO_MODE=local with explicit MONGO_URI"
	}
	if local != "" {
		return "MONGO_MODE=local using MONGO_URI_LOCAL/default"
	}
	return "MONGO_MODE=local (no URI provided)"
}

func envSnapshot() string {
	// WHY: helps debug precedence without leaking secrets.
	fields := []string{
		"MONGO_MODE=" + getenv("MONGO_MODE", "auto"),
		"MONGO_DB=" + getenv("MONGO_DB", "meniba"),
		"MONGO_URI=" + redactURI(os.Getenv("MONGO_URI")),
		"MONGO_URI_LOCAL=" + redactURI(getenv("MONGO_URI_LOCAL", "mongodb://localhost:27017")),
		"MONGO_URI_REMOTE=" + redactURI(os.Getenv("MONGO_URI_REMOTE")),
	}
	return strings.Join(fields, " ")
}

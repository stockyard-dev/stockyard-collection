// Stockyard Collection — Catalog and track your stuff.
// Flexible item catalog with categories, value tracking, notes, and search.
// Single binary, embedded SQLite, zero external dependencies.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/stockyard-dev/stockyard-collection/internal/server"
	"github.com/stockyard-dev/stockyard-collection/internal/store"
)

var version = "dev"

func main() {
	portFlag := flag.Int("port", 0, "HTTP port")
	dataFlag := flag.String("data", "", "Data directory")
	flag.Parse()
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("collection %s\n", version)
		os.Exit(0)
	}
	if len(os.Args) > 1 && (os.Args[1] == "--health" || os.Args[1] == "health") {
		fmt.Println("ok")
		os.Exit(0)
	}

	log.SetFlags(log.Ltime | log.Lshortfile)
	port := 8840
	if p := os.Getenv("PORT"); p != "" {
		if n, _ := strconv.Atoi(p); n > 0 {
			port = n
		}
	}
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}

	if *portFlag > 0 {
		port = *portFlag
	}
	if *dataFlag != "" {
		dataDir = *dataFlag
	}
	limits := server.DefaultLimits()
	db, err := store.Open(dataDir)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	log.Printf("  Questions? hello@stockyard.dev")
	log.Printf("")
	log.Printf("  Stockyard Collection %s", version)
	log.Printf("  Dashboard:  http://localhost:%d/ui", port)
	log.Printf("  API:        http://localhost:%d/api", port)
	log.Printf("  Tier:       %s", limits.Tier)
	log.Printf("")

	srv := server.New(db, port, limits, dataDir)
	if err := srv.Start(); err != nil {
		log.Fatalf("server: %v", err)
	}
}

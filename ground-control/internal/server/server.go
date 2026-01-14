package server

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"
	_ "github.com/lib/pq"

	"github.com/container-registry/harbor-satellite/ground-control/internal/database"
)

type Server struct {
	port      int
	db        *sql.DB
	dbQueries *database.Queries
}

// TLSConfig holds TLS settings for the server.
type TLSConfig struct {
	CertFile string
	KeyFile  string
	CAFile   string
	Enabled  bool
}

// ServerResult contains the http.Server and TLS configuration.
type ServerResult struct {
	Server    *http.Server
	TLSConfig *TLSConfig
}

var (
	dbName   = os.Getenv("DB_DATABASE")
	password = os.Getenv("DB_PASSWORD")
	username = os.Getenv("DB_USERNAME")
	PORT     = os.Getenv("DB_PORT")
	HOST     = os.Getenv("DB_HOST")
)

func NewServer() *ServerResult {
	port, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		log.Fatalf("PORT is not valid: %v", err)
	}

	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		username,
		password,
		HOST,
		PORT,
		dbName,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Error in sql: %v", err)
	}

	dbQueries := database.New(db)

	newServer := &Server{
		port:      port,
		db:        db,
		dbQueries: dbQueries,
	}

	tlsCfg := loadTLSConfig()

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", newServer.port),
		Handler:      newServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	if tlsCfg.Enabled {
		tlsConfig, err := buildServerTLSConfig(tlsCfg)
		if err != nil {
			log.Fatalf("Failed to load TLS config: %v", err)
		}
		httpServer.TLSConfig = tlsConfig
	}

	return &ServerResult{
		Server:    httpServer,
		TLSConfig: tlsCfg,
	}
}

func loadTLSConfig() *TLSConfig {
	certFile := os.Getenv("TLS_CERT_FILE")
	keyFile := os.Getenv("TLS_KEY_FILE")
	caFile := os.Getenv("TLS_CA_FILE")

	enabled := certFile != "" && keyFile != ""

	return &TLSConfig{
		CertFile: certFile,
		KeyFile:  keyFile,
		CAFile:   caFile,
		Enabled:  enabled,
	}
}

func buildServerTLSConfig(cfg *TLSConfig) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("load certificate: %w", err)
	}

	tlsConfig := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
	}

	if cfg.CAFile != "" {
		caData, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("read CA file: %w", err)
		}

		caPool := x509.NewCertPool()
		if !caPool.AppendCertsFromPEM(caData) {
			return nil, fmt.Errorf("invalid CA certificate")
		}

		tlsConfig.ClientCAs = caPool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}

	return tlsConfig, nil
}

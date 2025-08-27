package main

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var logger *slog.Logger

// Config holds the runtime configuration for the FTP server
type Config struct {
	Port            int
	LogLevel        slog.Level
	PaperlessURL    string
	PaperlessToken  string
	HTTPTimeout     time.Duration
}

// NewConfig creates a new Config with default values
func NewConfig() *Config {
	return &Config{
		Port:           2121,
		LogLevel:       slog.LevelInfo,
		PaperlessURL:   "http://localhost:8000",
		PaperlessToken: "",
		HTTPTimeout:    30 * time.Second,
	}
}

// LoadFromEnv updates config from environment variables
func (c *Config) LoadFromEnv() {
	// Load port from FTP_PORT environment variable
	if envPort := os.Getenv("FTP_PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			c.Port = p
		}
	}

	// Load Paperless-ngx URL from PAPERLESS_URL environment variable
	if envPaperlessURL := os.Getenv("PAPERLESS_URL"); envPaperlessURL != "" {
		c.PaperlessURL = envPaperlessURL
	}

	// Load Paperless-ngx API token from PAPERLESS_TOKEN environment variable
	if envPaperlessToken := os.Getenv("PAPERLESS_TOKEN"); envPaperlessToken != "" {
		c.PaperlessToken = envPaperlessToken
	}

	// Load HTTP timeout from HTTP_TIMEOUT environment variable
	if envTimeout := os.Getenv("HTTP_TIMEOUT"); envTimeout != "" {
		if t, err := time.ParseDuration(envTimeout); err == nil {
			c.HTTPTimeout = t
		}
	}

	// Load log level from LOG_LEVEL environment variable
	if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
		switch strings.ToUpper(envLogLevel) {
		case "DEBUG":
			c.LogLevel = slog.LevelDebug
		case "INFO":
			c.LogLevel = slog.LevelInfo
		case "WARN":
			c.LogLevel = slog.LevelWarn
		case "ERROR":
			c.LogLevel = slog.LevelError
		}
	}
}

type FTPServer struct {
	listener       net.Listener
	paperlessURL   string
	paperlessToken string
	httpClient     *http.Client
}

func NewFTPServer(config *Config) (*FTPServer, error) {
	listener, err := net.Listen("tcp", net.JoinHostPort("", strconv.Itoa(config.Port)))
	if err != nil {
		return nil, err
	}

	// Validate required Paperless configuration
	if config.PaperlessToken == "" {
		return nil, fmt.Errorf("PAPERLESS_TOKEN environment variable is required")
	}

	return &FTPServer{
		listener:       listener,
		paperlessURL:   config.PaperlessURL,
		paperlessToken: config.PaperlessToken,
		httpClient: &http.Client{
			Timeout: config.HTTPTimeout,
		},
	}, nil
}

// handleLIST processes the LIST command for directory listings
func (s *FTPServer) handleLIST(conn net.Conn, activeHost string, activePort int) {
	s.send(conn, "150 Opening data connection")

	// Establish active connection using PORT/EPRT
	if activeHost != "" && activePort > 0 {
		dataConn, err := net.Dial("tcp", net.JoinHostPort(activeHost, strconv.Itoa(activePort)))
		if err != nil {
			s.send(conn, "425 Can't open data connection")
			return
		}
		defer dataConn.Close()
		// Return empty listing since files go directly to Paperless-ngx
	}

	s.send(conn, "226 Transfer complete")
}

// handleSTOR processes the STOR command for file uploads
func (s *FTPServer) handleSTOR(conn net.Conn, filename string, activeHost string, activePort int) {
	s.send(conn, "150 Opening data connection")

	// Establish active connection using PORT/EPRT
	if activeHost != "" && activePort > 0 {
		dataConn, err := net.Dial("tcp", net.JoinHostPort(activeHost, strconv.Itoa(activePort)))
		if err != nil {
			s.send(conn, "425 Can't open data connection")
			return
		}
		defer dataConn.Close()

		// Read all file data into memory
		fileData, err := io.ReadAll(dataConn)
		if err != nil {
			s.send(conn, "426 Connection closed; transfer aborted")
			return
		}

		// Upload directly to Paperless-ngx
		if err := s.uploadToPaperless(filename, fileData); err != nil {
			logger.Error("Failed to upload to Paperless-ngx", "error", err, "filename", filename)
			s.send(conn, "550 Upload failed")
		} else {
			s.send(conn, "226 Transfer complete")
		}
	} else {
		s.send(conn, "425 Can't open data connection")
	}
}

// uploadToPaperless uploads a document to Paperless-ngx via REST API
func (s *FTPServer) uploadToPaperless(filename string, fileData []byte) error {

	// Create multipart form
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	// Add the document file
	fw, err := w.CreateFormFile("document", filepath.Base(filename))
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := fw.Write(fileData); err != nil {
		return fmt.Errorf("failed to write file data: %w", err)
	}

	// Close the multipart writer
	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create the HTTP request
	url := fmt.Sprintf("%s/api/documents/post_document/", strings.TrimRight(s.paperlessURL, "/"))
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", fmt.Sprintf("Token %s", s.paperlessToken))

	// Send the request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("paperless API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	logger.Info("Document uploaded to Paperless-ngx", "filename", filename, "status", resp.StatusCode)
	return nil
}

func (s *FTPServer) Start() {
	logger.Info("FTP Server started", "port", s.listener.Addr().String(), "paperless_url", s.paperlessURL)

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			logger.Warn("Failed to accept connection", "error", err)
			continue
		}
		go s.handleConnection(conn)
	}
}

func (s *FTPServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	s.send(conn, "220 Simple FTP Server Ready")

	var activeHost string
	var activePort int
	currentDir := "/"

	for {
		line, err := s.readLine(conn)
		if err != nil {
			break
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		cmd := strings.ToUpper(parts[0])

		switch cmd {
		case "USER":
			s.send(conn, "331 User name okay, need password")

		case "PASS":
			s.send(conn, "230 User logged in, proceed")

		case "SYST":
			s.send(conn, "215 UNIX Type: L8")

		case "TYPE":
			s.send(conn, "200 Type set")

		case "PWD":
			s.send(conn, fmt.Sprintf("257 \"%s\" is current directory", currentDir))

		case "CWD":
			if len(parts) > 1 {
				currentDir = parts[1]
			}
			s.send(conn, "250 Directory changed")

		case "PASV":
			s.send(conn, "502 Command not implemented")

		case "LIST":
			s.handleLIST(conn, activeHost, activePort)

		case "STOR":
			if len(parts) < 2 {
				s.send(conn, "501 Syntax error")
				continue
			}
			s.handleSTOR(conn, parts[1], activeHost, activePort)

		case "RETR":
			// File retrieval not supported since files go directly to Paperless-ngx
			s.send(conn, "550 File retrieval not supported")

		case "DELE":
			// File deletion not supported since files go directly to Paperless-ngx
			s.send(conn, "550 File deletion not supported")

		case "QUIT":
			s.send(conn, "221 Goodbye")
			return

		case "NOOP":
			s.send(conn, "200 OK")

		case "FEAT":
			s.send(conn, "211-Features:")
			s.send(conn, " UTF8")
			s.send(conn, "211 End")

		case "OPTS":
			if len(parts) > 1 && strings.ToUpper(parts[1]) == "UTF8" {
				s.send(conn, "200 UTF8 mode enabled")
			} else {
				s.send(conn, "200 OK")
			}

		case "PORT":
			if len(parts) < 2 {
				s.send(conn, "501 Syntax error")
				continue
			}

			// Parse PORT h1,h2,h3,h4,p1,p2
			addr := strings.Split(parts[1], ",")
			if len(addr) != 6 {
				s.send(conn, "501 Syntax error")
				continue
			}

			activeHost = fmt.Sprintf("%s.%s.%s.%s", addr[0], addr[1], addr[2], addr[3])
			p1, _ := strconv.Atoi(addr[4])
			p2, _ := strconv.Atoi(addr[5])
			activePort = p1*256 + p2

			s.send(conn, "200 PORT command successful")

		case "EPRT":
			if len(parts) < 2 {
				s.send(conn, "501 Syntax error")
				continue
			}

			// Parse EPRT |1|address|port| or |2|address|port|
			eprtData := strings.Join(parts[1:], " ")
			eprtParts := strings.Split(eprtData, "|")
			if len(eprtParts) < 4 {
				s.send(conn, "501 Syntax error")
				continue
			}

			protocol := eprtParts[1]
			if protocol != "1" && protocol != "2" {
				s.send(conn, "522 Network protocol not supported")
				continue
			}

			activeHost = eprtParts[2]
			activePort, _ = strconv.Atoi(eprtParts[3])

			s.send(conn, "200 EPRT command successful")

		default:
			s.send(conn, "502 Command not implemented")
		}
	}
}


func (s *FTPServer) send(conn net.Conn, msg string) {
	conn.Write([]byte(msg + "\r\n"))
	logger.Debug("FTP response", "message", msg)
}

func (s *FTPServer) readLine(conn net.Conn) (string, error) {
	buf := make([]byte, 1024)
	var line []byte

	for {
		n, err := conn.Read(buf)
		if err != nil {
			return "", err
		}

		line = append(line, buf[:n]...)

		if strings.Contains(string(line), "\r\n") {
			cmd := strings.TrimSpace(string(line))
			logger.Debug("FTP command", "command", cmd)
			return cmd, nil
		}
	}
}

func main() {
	// Create and load configuration from environment variables
	config := NewConfig()
	config.LoadFromEnv()

	// Create logger with text handler
	opts := &slog.HandlerOptions{
		Level: config.LogLevel,
	}
	logger = slog.New(slog.NewTextHandler(os.Stdout, opts))

	server, err := NewFTPServer(config)
	if err != nil {
		logger.Error("Failed to create FTP server", "error", err)
		os.Exit(1)
	}

	logger.Info("Starting FTP server",
		"port", config.Port,
		"paperless_url", config.PaperlessURL,
		"log_level", config.LogLevel.String(),
		"auth", "disabled")

	server.Start()
}

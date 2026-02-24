// generate-test-jwt generates an RSA key pair, a valid session JWT, and a
// minimal config.yaml that can be used to start rfd-server for load testing
// without needing a real OIDC provider.
//
// Usage:
//
//	go run ./cmd/generate-test-jwt
//	go run ./cmd/generate-test-jwt -out /tmp/loadtest
package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/geekgonecrazy/rfd-tool/models"
	"github.com/golang-jwt/jwt/v5"
)

func main() {
	outDir := flag.String("out", ".", "Directory to write generated files to")
	flag.Parse()

	if err := os.MkdirAll(*outDir, 0755); err != nil {
		log.Fatalf("mkdir: %v", err)
	}

	// Generate RSA-2048 key pair
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("generate key: %v", err)
	}

	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privKey),
	})

	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(&privKey.PublicKey),
	})

	// Create session token valid for 24 hours
	expiry := time.Now().Add(24 * time.Hour)
	session := models.SessionToken{
		User: models.SessionUser{
			Name:     "Load Test User",
			Email:    "loadtest@example.com",
			Staff:    true,
			LoggedIn: true,
		},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiry),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	tokenStr, err := jwt.NewWithClaims(jwt.SigningMethodRS256, session).SignedString(privKey)
	if err != nil {
		log.Fatalf("sign token: %v", err)
	}

	// Write files
	tokenPath := filepath.Join(*outDir, "test-token.txt")
	if err := os.WriteFile(tokenPath, []byte(tokenStr), 0600); err != nil {
		log.Fatalf("write token: %v", err)
	}

	configYAML := fmt.Sprintf(`site:
  name: "RFD Load Test"
  url: "http://localhost:8877"

dataPath: "%s/"

store: "sqlite"

apiSecret: "test-api-secret"

jwt:
  privateKey: |
%s
  publicKey: |
%s
`, filepath.Join(*outDir, "data"), indent(string(privPEM), 4), indent(string(pubPEM), 4))

	configPath := filepath.Join(*outDir, "config-loadtest.yaml")
	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		log.Fatalf("write config: %v", err)
	}

	fmt.Println("=== Generated Load Test Credentials ===")
	fmt.Println()
	fmt.Println("Config:     ", configPath)
	fmt.Println("JWT Token:  ", tokenPath)
	fmt.Println("API Secret: ", "test-api-secret")
	fmt.Println("Expires:    ", expiry.Format(time.RFC3339))
	fmt.Println()
	fmt.Println("Start the server:")
	fmt.Printf("  go run ./cmd/rfd-server -configFile %s\n", configPath)
	fmt.Println()
	fmt.Println("Use the JWT in K6 or curl:")
	fmt.Printf("  export TOKEN=$(cat %s)\n", tokenPath)
	fmt.Println(`  curl -H "Authorization: $TOKEN" http://localhost:8877/api/v1/rfds`)
}

// indent prepends each line with n spaces.
func indent(s string, n int) string {
	prefix := ""
	for i := 0; i < n; i++ {
		prefix += " "
	}
	out := ""
	for i, line := range splitLines(s) {
		if i > 0 {
			out += "\n"
		}
		if line != "" {
			out += prefix + line
		}
	}
	return out
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

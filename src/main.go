package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"log"
	"os"
	"os/exec"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func generateHMACSignature(secret string, payload []byte) string {
	h := hmac.New(sha1.New, []byte(secret))
	h.Write(payload)
	return "sha1=" + hex.EncodeToString(h.Sum(nil))
}

func runCommand(command string) error {
	workDir := os.Getenv("WORK_DIR")
	if workDir != "" {
		os.Chdir(workDir)
	}
	cmd := exec.Command("bash", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func main() {
	if _, err := os.Stat(".env"); err == nil {
		err := godotenv.Load()
		if err != nil {
			log.Fatal("Failed to load .env file: ", err)
		}
	} else if os.IsNotExist(err) {
		log.Println(".env file not found, continuing without loading it")
	} else {
		log.Fatal("Error checking for .env file: ", err)
	}

	secret := os.Getenv("GITHUB_SECRET")
	if secret == "" {
		log.Fatal("GITHUB_SECRET is not set")
		os.Exit(1)
	}
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.POST("/", func(c *gin.Context) {
		payload, err := io.ReadAll(c.Request.Body)
		if err != nil {
			log.Fatalf("Error reading body: %v", err)
			c.AbortWithStatus(400)
			return
		}

		if generateHMACSignature(secret, payload) != c.Request.Header.Get("X-Hub-Signature") {
			log.Fatalf("Invalid HMAC signature: %s", c.GetHeader("X-Hub-Signature"))
			c.AbortWithStatus(401)
			return
		}

		go func() {
			extraCommand := os.Getenv("EXTRA_COMMAND")
			if extraCommand == "" {
				return
			}
			err := runCommand(extraCommand) // TODO: push this notice.
			if err != nil {
				log.Fatalf("Error running command: %v", err)
				return
			}
		}()

		c.Status(200)
	})

	PORT := os.Getenv("PORT")
	if PORT == "" {
		PORT = "4567"
	}
	r.Run(":" + PORT)
	log.Printf("Listening on port %s", PORT)
}

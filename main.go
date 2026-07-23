package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
	"gopkg.in/mail.v2"
)

var attachments []string

type Config struct {
	TokenPath       string
	CredentialsPath string
	Recipients      []string
	Subject         string
	BodyMessage     string
	AttachmentPaths []string
	RedirectUrl     string
}

func config() Config {
	err := godotenv.Load(".env")
	if err != nil {
		panic(fmt.Errorf("Error loading .env file: %w", err))
	}
	return Config{
		TokenPath:       "token.json",
		CredentialsPath: "credentials.json",
		Recipients:      strings.Split(os.Getenv("RECIPIENTS"), ","),
		Subject:         os.Getenv("SUBJECT"),
		BodyMessage:     os.Getenv("MESSAGE"),
		AttachmentPaths: attachments,
		RedirectUrl:     "http://localhost:8080",
	}
}

// Retrieves a token, saves it, then returns the generated client.
// The tokenFile stores the user's access and refresh tokens.
// It is created when the authorization flow completes for the first time.
func getClient(oauthCfg *oauth2.Config, tokenFile string) *http.Client {
	tok, err := tokenFromFile(tokenFile)
	if err != nil {
		tok = getTokenFromWeb(oauthCfg)
		saveToken(tokenFile, tok)
	}
	return oauthCfg.Client(context.Background(), tok)
}

// Requests a token from the web, then returns the retrieved token.
func getTokenFromWeb(oauthCfg *oauth2.Config) *oauth2.Token {
	// Generate the URL
	authURL := oauthCfg.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("\nACTION REQUIRED\nVisit this link in your browser:\n%v\n\n", authURL)

	authCodeCh := make(chan string)

	// Create the server
	mux := http.NewServeMux()
	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code != "" {
			fmt.Fprint(w, "Authorization complete! You can close this window and return to your terminal.")
			authCodeCh <- code
		} else {
			http.Error(w, "No code in request", http.StatusBadRequest)
		}
	})

	// Start the server in the background
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for the browser to hit our local server and pass the code
	authCode := <-authCodeCh

	// Shut down the server so the port is freed up
	srv.Shutdown(context.Background())

	// Exchange the authorization code for a token
	tok, err := oauthCfg.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s...", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

// Returns an email as a base64url encoded string
func createEmailString(to, subject, body string, files []string) (string, error) {
	message := mail.NewMessage()
	message.SetHeader("To", to)
	message.SetHeader("Subject", subject)
	message.SetBody("text/plain", body)
	for _, file := range files {
		message.Attach(file)
	}

	var msgBuf bytes.Buffer
	_, err := message.WriteTo(&msgBuf)
	if err != nil {
		return "", fmt.Errorf("Error writing message to buffer: %w", err)
	}

	msgBytes := msgBuf.Bytes()
	encoder := base64.URLEncoding
	encBytes := make([]byte, encoder.EncodedLen(len(msgBytes)))
	encoder.Encode(encBytes, msgBytes)
	return string(encBytes), nil
}

func main() {
	// Parse paths to attachments
	fmt.Println("Starting setup... ")
	flag.Func("attachments", "Files to attach to emails", func(arg string) error {
		attachments = strings.Split(arg, " ")
		return nil
	})
	flag.Parse()

	cfg := config()
	credentials, err := os.ReadFile(cfg.CredentialsPath)
	if err != nil {
		panic(fmt.Errorf("Failed to read credentials from '%s': %w", cfg.CredentialsPath, err))
	}

	// Configure oauth with credentials, get client at redirect url
	oauthCfg, err := google.ConfigFromJSON(credentials, gmail.GmailSendScope)
	if err != nil {
		panic(fmt.Errorf("Failed to construct OAuth config with scope '%s' from credentials file at '%s': %w", gmail.GmailSendScope, cfg.CredentialsPath, err))
	}
	oauthCfg.RedirectURL = cfg.RedirectUrl
	client := getClient(oauthCfg, cfg.TokenPath)

	// Startup gmail service to send message
	ctx := context.Background()
	srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		panic(fmt.Errorf("Failed to create new Gmail service: %w", err))
	}
	fmt.Println("Done!\n")

	// Construct string and send message per recipient
	for _, r := range cfg.Recipients {
		fmt.Printf("About to create email to %s... ", r)
		emailStr, err := createEmailString(r, cfg.Subject, cfg.BodyMessage, cfg.AttachmentPaths)
		if err != nil {
			panic(fmt.Errorf("Error creating email string: %w", err))
		}
		fmt.Println("Done!")

		fmt.Print("About to send... ")
		_, err = srv.Users.Messages.Send(
			"me",
			&gmail.Message{Raw: emailStr},
		).Do()
		if err != nil {
			panic(fmt.Errorf("Error sending email: %w", err))
		}

		fmt.Println("Done!")
	}
}

package drive_auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path"

	"github.com/pkg/browser"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// GetService returns a Service object to interact with Google Drive API
//
// Current implementation uses OAuth2.0 to authenticate the user with an interactive browser flow, and saves
// a token file to the local filesystem to avoid the browser flow every time
// In the future, this could be replaced with a service account for non-interactive use
func GetService(ctx context.Context, clientSecretPath string, clientTokenPath string, forReading bool, forWriting bool) (*drive.Service, error) {
	b, err := os.ReadFile(clientSecretPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read client secret file: %v", err)
	}

	// if modifying these scopes, delete your previously saved token.json
	scopes := []string{}
	if forReading {
		// allows to read all files accessible by user
		scopes = append(scopes, drive.DriveReadonlyScope)
	}
	if forWriting {
		// allows to write to files exclusively created by this app
		scopes = append(scopes, drive.DriveFileScope)
	}
	config, err := google.ConfigFromJSON(b, scopes...)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %v", err)
	}

	// The file token.json stores the user's access and refresh tokens, and is
	// created when the authorization flow completes for the first time.
	tokFile := path.Join(clientTokenPath, "token-")
	if forReading {
		tokFile = tokFile + "r"
	}
	if forWriting {
		tokFile = tokFile + "w"
	}
	tokFile = tokFile + ".json"
	tok, err := getTokenFromFile(tokFile)
	if err != nil {
		tok, err = getTokenFromWeb(ctx, config)
		if err != nil {
			return nil, fmt.Errorf("unable to get token from web: %v", err)
		}
		err = saveTokenToFile(tokFile, tok)
		if err != nil {
			return nil, fmt.Errorf("unable to save token: %v", err)
		}
	}
	client := config.Client(ctx, tok)

	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Drive client: %v", err)
	}
	return srv, err
}

func getTokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	err2 := f.Close()

	if err != nil || err2 != nil {
		return nil, fmt.Errorf("error reading token from file: %v, %v", err, err2)
	}

	return tok, nil
}

func saveTokenToFile(path string, token *oauth2.Token) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	err = json.NewEncoder(f).Encode(token)
	err2 := f.Close()

	if err != nil || err2 != nil {
		return fmt.Errorf("error writing token to file: %v, %v", err, err2)
	}

	return nil
}

func getTokenFromWeb(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	// Start listener to receive the auth code from the browser. Use address ":0" to get a random free port
	var lc net.ListenConfig
	listener, err := lc.Listen(ctx, "tcp", ":0")
	if err != nil {
		return nil, fmt.Errorf("unable to start listener: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	// set up a handler that will return the received auth code in the
	// authCode variable and signal via channel that the server can be shut down
	var authCode string
	done := make(chan bool)
	srv := &http.Server{Addr: fmt.Sprintf(":%v", port)}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		queryParams := r.URL.Query()
		for key, values := range queryParams {
			if key == "code" {
				for _, value := range values {
					authCode = value
					_, err = fmt.Fprintln(w, "You can close this tab.")
					if err != nil {
						fmt.Println("Error writing response:", err)
					}
					done <- true
				}
			}
		}
	})

	// start a goroutine to shut down the server when the auth code is received or the context is done
	go func() {
		select {
		case <-done:
		case <-ctx.Done():
		}
		err := srv.Shutdown(ctx)
		if err != nil {
			fmt.Println("Error shutting down server:", err)
		}
	}()

	// open the browser to authenticate
	config.RedirectURL = fmt.Sprintf("http://localhost:%v", port)
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	err = browser.OpenURL(authURL)
	if err != nil {
		return nil, fmt.Errorf("unable to open browser: %v", err)
	}

	// start the server
	fmt.Printf("Starting server at :%v to collect authentication code...\n", port)
	if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return nil, fmt.Errorf("unable to start server: %v", err)
	}

	// with the auth code received, exchange it for a token
	tok, err := config.Exchange(ctx, authCode)
	if err != nil {
		return nil, fmt.Errorf("unable to exchange auth code for token: %v", err)
	}
	return tok, nil
}

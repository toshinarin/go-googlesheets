package googlesheets

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/sheets/v4"
)

// [Reference] This file is based on the Google's sample https://developers.google.com/sheets/api/quickstart/go .
// The sample does all in main.go. To be used from applications, this file returns *sheets.Service.

var logger *log.Logger

func New(config *oauth2.Config, cacheFileName string) (*sheets.Service, error) {
	client, err := getClient(context.Background(), config, cacheFileName)
	if err != nil {
		return nil, err
	}
	return sheets.New(client)
}

// getClient uses a Context and Config to retrieve a Token
// then generate a Client. It returns the generated Client.
func getClient(ctx context.Context, config *oauth2.Config, cacheFileName string) (*http.Client, error) {
	cacheFile, err := tokenCacheFile(cacheFileName)
	if err != nil {
		return nil, fmt.Errorf("Unable to get path to cached credential file. %v", err)
	}
	tok, err := tokenFromFile(cacheFile)
	if err == nil {
		return config.Client(ctx, tok), nil
	}
	tok, err = getTokenFromWeb(ctx, config)
	if err != nil {
		return nil, err
	}
	if err := saveToken(cacheFile, tok); err != nil {
		return nil, err
	}
	return config.Client(ctx, tok), nil
}

// getTokenFromWeb uses Config to request a Token.
// It returns the retrieved Token.
func getTokenFromWeb(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	ch := make(chan string)
	randState := fmt.Sprintf("st%d", time.Now().UnixNano())
	ts := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/favicon.ico" {
			http.Error(rw, "", 404)
			return
		}
		if req.FormValue("state") != randState {
			log.Printf("State doesn't match: req = %#v", req)
			http.Error(rw, "", 500)
			return
		}
		if code := req.FormValue("code"); code != "" {
			fmt.Fprint(rw, "<h1>Success</h1>Authorized.")
			rw.(http.Flusher).Flush()
			ch <- code
			return
		}
		logger.Print("no code from web")
		http.Error(rw, "", 500)
	}))
	defer ts.Close()

	config.RedirectURL = ts.URL
	authURL := config.AuthCodeURL(randState)
	go openURL(authURL)
	logger.Printf("Authorize this app at: %s", authURL)
	code := <-ch
	logger.Printf("Got code: %s", code)

	token, err := config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("Token exchange error: %v", err)
	}
	return token, nil
}

func openURL(url string) {
	try := []string{"xdg-open", "google-chrome", "open"}
	for _, bin := range try {
		err := exec.Command(bin, url).Run()
		if err == nil {
			return
		}
	}
	logger.Print("Failed to open URL in browser.")
}

// tokenCacheFile generates credential file path/filename.
// It returns the generated credential path/filename.
func tokenCacheFile(cacheFileName string) (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	tokenCacheDir := filepath.Join(usr.HomeDir, ".google_oauth_credentials")
	os.MkdirAll(tokenCacheDir, 0700)
	const suffix = ".json"
	if !strings.HasSuffix(cacheFileName, suffix) {
		cacheFileName = cacheFileName + suffix
	}
	return filepath.Join(tokenCacheDir,
		url.QueryEscape(cacheFileName)), err
}

// tokenFromFile retrieves a Token from a given file path.
// It returns the retrieved Token and any read error encountered.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	t := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)
	defer f.Close()
	return t, err
}

// saveToken uses a file path to create a file and store the
// token in it.
func saveToken(file string, token *oauth2.Token) error {
	fmt.Printf("Saving credential file to: %s\n", file)
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
	return nil
}

func init() {
	logger = log.New(os.Stderr, "googlespreadsheet", log.LstdFlags)
}

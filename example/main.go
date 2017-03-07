package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/toshinarin/go-googlesheets"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/sheets/v4"
)

func newService(configJsonPath, credentialsFileName string) (*sheets.Service, error) {
	b, err := ioutil.ReadFile(configJsonPath)
	if err != nil {
		return nil, err
	}

	// If modifying these scopes, delete your previously saved credentials
	// at ~/.google_oauth_credentials/{credentialsFileName}
	config, err := google.ConfigFromJSON([]byte(b), "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		return nil, fmt.Errorf("Unable to parse client secret file to config: %v", err)
	}

	srv, err := googlesheets.New(config, credentialsFileName)
	if err != nil {
		return nil, fmt.Errorf("Unable to retrieve Sheets Client %v", err)
	}
	return srv, nil
}

func importSpreadSheet(srv *sheets.Service, spreadsheetId, spreadsheetRange string) error {
	resp, err := srv.Spreadsheets.Values.Get(spreadsheetId, spreadsheetRange).Do()
	if err != nil {
		return fmt.Errorf("Unable to retrieve data from sheet. %v", err)
	}
	for i, row := range resp.Values {
		fmt.Printf("row[%d]; %s\n", i, row)
	}
	return nil
}

func exportToSpreadSheet(srv *sheets.Service, spreadsheetId, spreadsheetRange string, rows [][]interface{}) error {
	valueRange := sheets.ValueRange{}
	for _, r := range rows {
		valueRange.Values = append(valueRange.Values, r)
	}

	clearReq := sheets.ClearValuesRequest{}
	clearResp, err := srv.Spreadsheets.Values.Clear(spreadsheetId, spreadsheetRange, &clearReq).Do()
	if err != nil {
		return fmt.Errorf("failed to clear sheet. error: %v", err)
	}
	log.Printf("clear response: %v", clearResp)

	resp, err := srv.Spreadsheets.Values.Update(spreadsheetId, spreadsheetRange, &valueRange).ValueInputOption("RAW").Do()
	if err != nil {
		return fmt.Errorf("failed to update sheet. error: %v", err)
	}
	log.Printf("update response: %v", resp)
	return nil
}

func main() {
	mode := flag.String("mode", "import", "import or export")
	spreadSheetID := flag.String("id", "", "google spread sheet id")
	flag.Parse()

	if *spreadSheetID == "" {
		log.Fatal("option -id: please set spread sheet id")
	}

	srv, err := newService("client_secret.json", "googlesheets-example.json")
	if err != nil {
		log.Fatal(err)
	}

	if *mode == "import" {
		if err := importSpreadSheet(srv, *spreadSheetID, "A1:B"); err != nil {
			log.Fatal(err)
		}
	} else if *mode == "export" {
		rows := [][]interface{}{[]interface{}{"a1", "b1"}, []interface{}{"a2", "b2"}}
		if err := exportToSpreadSheet(srv, *spreadSheetID, "A1:B", rows); err != nil {
			log.Fatal(err)
		}
	} else {
		log.Fatal("option -mode: please set import or export")
	}
}

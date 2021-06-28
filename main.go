package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"path"
	"strings"
	"time"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

type results struct {
	Results []*result `json:"results"`
}

type result struct {
	PaintColor        []string `json:"PAINT"`
	MetroName         string   `json:"MetroName"`
	Hash              string   `json:"Hash"`
	AdditionalOptions []string `json:"ADL_OPTS"`
}

var emailPwd string

func run() error {
	pwd := os.Getenv("EMAIL_PWD")
	if pwd == "" {
		return fmt.Errorf("set EMAIL_PWD env var")
	}
	emailPwd = pwd

	for {
		fmt.Println("Starting iteration...")
		if err := doIter(); err != nil {
			log.Printf("Error running iter. err=%s", err)
		}
		time.Sleep(15 * time.Second)
	}
}

func doIter() error {
	rs, err := findResults()
	if err != nil {
		return err
	}

	if len(rs) == 0 {
		fmt.Println("No results.")
		return nil
	}

	for _, r := range rs {
		isSent, err := checkIfSent(r)
		if err != nil {
			return err
		} else if isSent {
			continue
		}

		fmt.Println("Sending email.")
		if err := sendEmail(r); err != nil {
			return err
		}

		fmt.Println("Marking email as sent.")
		if err := markSent(r); err != nil {
			return err
		}
	}

	return nil
}

func findResults() ([]*result, error) {
	zipCode := "92612"

	req, err := http.NewRequest(
		"GET",
		strings.ReplaceAll(
			"https://www.tesla.com/inventory/api/v1/inventory-results?query=%7B%22query%22%3A%7B%22model%22%3A%22m3%22%2C%22condition%22%3A%22new%22%2C%22options%22%3A%7B%7D%2C%22arrangeby%22%3A%22Price%22%2C%22order%22%3A%22asc%22%2C%22market%22%3A%22US%22%2C%22language%22%3A%22en%22%2C%22super_region%22%3A%22north%20america%22%2C%22lng%22%3A-117.8282121%2C%22lat%22%3A33.6588951%2C%22zip%22%3A%2292612%22%2C%22range%22%3A0%2C%22region%22%3A%22CA%22%7D%2C%22offset%22%3A0%2C%22count%22%3A50%2C%22outsideOffset%22%3A0%2C%22outsideSearch%22%3Atrue%7D",
			"92612",
			zipCode,
		),
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error not OK: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	rs := &results{}

	if err := json.Unmarshal(body, rs); err != nil {
		return nil, err
	}

	var filteredR []*result
	for _, r := range rs.Results {
		if arrContains(r.PaintColor, "RED") && !arrContains(r.AdditionalOptions, "PERFORMANCE_UPGRADE") {
			filteredR = append(filteredR, r)
		}
	}

	return filteredR, nil
}

func arrContains(haystack []string, needle string) bool {
	for _, maybe := range haystack {
		if maybe == needle {
			return true
		}
	}
	return false
}

func checkIfSent(r *result) (bool, error) {
	_, err := os.Stat(computeFname(r))
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func markSent(r *result) error {
	f, err := os.Create(computeFname(r))
	if err != nil {
		return err
	}
	return f.Close()
}

func computeFname(r *result) string {
	fname := strings.ReplaceAll(r.Hash, "/", "")
	fname = strings.ReplaceAll(fname, ".", "")
	fname = path.Join(".", fname)
	return fname
}

func sendEmail(r *result) error {
	msg := "From: jnewman2501@gmail.com\n" +
		"To: jnewman2501@gmail.com\n" +
		fmt.Sprintf("Subject: New Tesla Found In %s\n\n", r.MetroName) +
		"New Tesla Found!\n"
	return smtp.SendMail(
		"smtp.gmail.com:587",
		smtp.PlainAuth("", "jnewman2501@gmail.com", emailPwd, "smtp.gmail.com"),
		"jnewman2501@gmail.com",
		[]string{"jnewman2501@gmail.com"},
		[]byte(msg),
	)
}

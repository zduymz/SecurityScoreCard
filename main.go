package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Entities struct {
	Entity []Entity `json:"entries"`
}

type Entity struct {
	Name     string  `json:"name"`
	Score    int     `json:"score"`
	Grade    string  `json:"grade"`
	GradeURL string  `json:"grade_url"`
	Issues   []Issue `json:"issue_summary"`
}

type Issue struct {
	Severity string `json:"severity"`
	Type     string `json:"issue_type"`
	Count    int    `json:"count"`
}

// send request to api
func createRequest() {
	client := &http.Client{}
	url := "https://api.securityscorecard.com/api/v1/vendors/lifelock.com/factorsummary"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Authorization", "Token 644a701114d825f71115a2dadd1cf9be")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		handleResponse(bodyBytes)
	} else {
		log.Fatal("Response status code: ", resp.StatusCode)
	}
}

// handle response and check
func handleResponse(data []byte) {
	var jsonData Entities
	var result []Entity
	json.Unmarshal(data, &jsonData)

	if len(jsonData.Entity) == 0 {
		log.Println("Empty response")
	}

	for _, entity := range jsonData.Entity {
		// TODO: need to improve condition to get more accuarate
		if entity.Score < 99 {
			result = append(result, entity)
		}
	}

	// send email alert if got something wrong
	if len(result) > 0 {
		sendAlert(&result)
		// log.Println(result)
	}

}

func sendAlert(result *[]Entity) {
	// Prepare result
	var tmp []string
	for _, entity := range *result {
		tmp = append(tmp, fmt.Sprintf("%s : %s", entity.Name, entity.Grade))
	}
	data := strings.Join(tmp, "\n")

	// Connect to the remote SMTP server.
	c, err := smtp.Dial("mail.example.com:25")
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	// Set the sender and recipient.
	c.Mail("sender@example.org")
	c.Rcpt("recipient@example.net")

	// Send the email body.
	wc, err := c.Data()
	if err != nil {
		log.Fatal(err)
	}
	defer wc.Close()
	buf := bytes.NewBufferString(data)
	if _, err = buf.WriteTo(wc); err != nil {
		log.Fatal(err)
	}
}

// catch interupt from keyboard
func enforceGracefulShutdown(f func(wg *sync.WaitGroup, shutdown chan struct{})) {
	wg := &sync.WaitGroup{}
	shutdown := make(chan struct{})
	signals := make(chan os.Signal)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	go func() {
		<-signals
		close(shutdown)
	}()

	log.Println("Graceful shutdown enabled")
	f(wg, shutdown)

	<-shutdown
	wg.Wait()
}

// make scheduler
func startScheduler(wg *sync.WaitGroup, shutdown chan struct{}) {
	wg.Add(1)
	defer wg.Done()

	wait := time.After(0)
	for {
		select {
		case <-wait:
			createRequest()
		case <-shutdown:
			log.Println("Shutting down process")
			return
		}
		// interval is 5 minutes
		wait = time.After(5 * time.Minute)
	}
}

func main() {
	enforceGracefulShutdown(func(wg *sync.WaitGroup, shutdown chan struct{}) {
		startScheduler(wg, shutdown)
	})
}

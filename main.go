package main

import (
	"fmt"
	"html/template"
	"log"
	"math/rand/v2"
	"net"
	"net/http"
	"os"
	"path"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

type RandomData struct {
	TelegramName string
	FactorOne    int
	FactorTwo    int
	Product      int
}

var (
	myRealTelegramUsername    string
	randomMutex               sync.RWMutex
	lastRandom                []RandomData
	whenLastIncludedRealEntry time.Time
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	listenURL := os.Getenv("LISTEN_URL")
	myRealTelegramUsername = os.Getenv("MY_REAL_TELEGRAM_USERNAME")
	whenLastIncludedRealEntry = generateRandomTimeLastWeek()

	go generateRandomData()

	router := http.NewServeMux()

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		randomMutex.RLock()
		defer randomMutex.RUnlock()

		fp := path.Join("templates", "index.html")
		tmpl, err := template.ParseFiles(fp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := tmpl.Execute(w, lastRandom); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// Start the server
	l, err := net.Listen("tcp", listenURL)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	serv := &http.Server{
		Addr:    listenURL,
		Handler: router,
	}
	fmt.Println("Server started")
	err = serv.Serve(l)
	if err != nil {
		log.Fatalf("Error serving: %v", err)
	}
}

func generateRandomData() {
	for {
		randomSlice := make([]RandomData, 10000)
		for i := range randomSlice {
			randomSlice[i] = RandomData{
				TelegramName: randomTelegramName(randRange(5, 20)),
				FactorOne:    randRange(1, 1000),
				FactorTwo:    randRange(1, 1000),
			}

			factorOneIncorrect := randomSlice[i].FactorOne + randRange(1, 5)
			factorTwoIncorrect := randomSlice[i].FactorTwo + randRange(1, 5)
			randomSlice[i].Product = factorOneIncorrect * factorTwoIncorrect
		}

		if isALittleOverAWeekSince(whenLastIncludedRealEntry) {
			realEntryFactorOne := randRange(1, 1000)
			realEntryFactorTwo := randRange(1, 1000)
			realEntry := RandomData{
				TelegramName: myRealTelegramUsername,
				FactorOne:    realEntryFactorOne,
				FactorTwo:    realEntryFactorTwo,
				Product:      realEntryFactorOne * realEntryFactorTwo,
			}
			randomSlice[randRange(0, 9999)] = realEntry

			whenLastIncludedRealEntry = time.Now()
		}

		randomMutex.Lock()
		lastRandom = randomSlice
		randomMutex.Unlock()

		time.Sleep(time.Hour)
	}
}

func randRange(min, max int) int {
	result := rand.IntN(max-min) + min
	if result < 0 {
		return -result
	}
	return result
}

func randomTelegramName(length int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyz.")
	s := make([]rune, length)
	for i := range s {
		s[i] = letters[rand.IntN(len(letters))]
	}
	return "@" + string(s)
}

// Checks if the current time is at least one week and a random additional
// time up to 48 hours since the given time.
func isALittleOverAWeekSince(t time.Time) bool {
	// Define a week as a constant duration.
	const week = 7 * 24 * time.Hour

	// Generate a random additional duration up to 48 hours.
	randomAdditionalDuration := time.Duration(rand.IntN(48)) * time.Hour

	// Calculate the target time which is a week plus some random additional time since `t`.
	targetTime := t.Add(week + randomAdditionalDuration)

	// Check if the current time is past this target time.
	return time.Now().After(targetTime)
}

func generateRandomTimeLastWeek() time.Time {
	now := time.Now()

	// Generate a random number of minutes up to one week (10080 minutes in a week)
	randomMinutes := randRange(0, 10080)

	// Subtract the random number of minutes from the current time
	randomTime := now.Add(time.Duration(-randomMinutes) * time.Minute)

	return randomTime
}

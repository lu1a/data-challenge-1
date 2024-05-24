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
	if len(os.Args) != 4 {
		log.Fatal("Please supply the correct arguments")
	}

	listenURL := os.Args[1]
	if len(listenURL) == 0 {
		listenURL = os.Getenv("LISTEN_URL")
	}
	myRealTelegramUsername = os.Args[2]
	if len(myRealTelegramUsername) == 0 {
		myRealTelegramUsername = os.Getenv("MY_REAL_TELEGRAM_USERNAME")
	}
	rootPath := os.Args[3]

	whenLastIncludedRealEntry = generateRandomTimeLastDay()

	go generateRandomData()

	router := http.NewServeMux()

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		randomMutex.RLock()
		defer randomMutex.RUnlock()

		fp := path.Join(rootPath, "templates", "index.html")
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

		if isALittleOverADaySince(whenLastIncludedRealEntry) {
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

// Checks if the current time is a bit over a single day (give or take a couple of hours) since the given time.
func isALittleOverADaySince(t time.Time) bool {
	// Define a day as a constant duration.
	const day = 24 * time.Hour

	// Generate a random additional duration up to 4 hours.
	randomAdditionalDuration := time.Duration(rand.IntN(4*60*60)) * time.Second

	// Calculate the target time which is a day plus some random additional time since `t`.
	targetTime := t.Add(day + randomAdditionalDuration)

	// Check if the current time is past this target time.
	return time.Now().After(targetTime)
}

func generateRandomTimeLastDay() time.Time {
	now := time.Now()

	// Generate a random number of minutes up to one day (1440 minutes in a day)
	randomMinutes := randRange(0, 1440)

	// Subtract the random number of minutes from the current time
	randomTime := now.Add(time.Duration(-randomMinutes) * time.Minute)

	return randomTime
}

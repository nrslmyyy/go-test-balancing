package main

import (
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/robfig/cron/v3"
)

func main() {
	http.HandleFunc("/consume-memory/", consumeMemoryHandler)
	http.HandleFunc("/status/200", status200Handler)
	http.HandleFunc("/status/400", status400Handler)
	http.HandleFunc("/status/500", status500Handler)
	scheduler := cron.New(cron.WithLocation(time.FixedZone("Asia/Jakarta", 7*60*60)))
	Job(scheduler)
	ListJobs(scheduler)
	fmt.Println("Server listening on port 8080...")
	http.ListenAndServe(":8080", nil)

}

func status200Handler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	// Get Ip from Here http://checkip.amazonaws.com
	ip, err := http.Get("http://checkip.amazonaws.com")
	if err != nil {
		fmt.Println("Failed to get IP:", err)
		return
	}
	defer ip.Body.Close()

	ipAddress, err := ioutil.ReadAll(ip.Body)
	if err != nil {
		fmt.Println("Failed to read IP:", err)
		return
	}

	fmt.Println("IP Address:", string(ipAddress))
	fmt.Fprintf(w, "Status 200 - OK\n"+"IP Address: "+string(ipAddress)+"\n")
}

func status400Handler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
	fmt.Fprintf(w, "Status 400 - Bad Request\n")
}

func status500Handler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintf(w, "Status 500 - Internal Server Error\n")
}

func consumeMemoryHandler(w http.ResponseWriter, r *http.Request) {
	mbStr := r.URL.Path[len("/consume-memory/"):]
	mb, err := strconv.Atoi(mbStr)
	if err != nil {
		http.Error(w, "Invalid number of megabytes", http.StatusBadRequest)
		return
	}

	// Allocate memory
	data := make([]byte, mb*1024*1024)
	if data == nil {
		http.Error(w, "Failed to allocate memory", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Allocated %d MB of memory", mb)
}

func Job(scheduler *cron.Cron) {
	file, err := os.Open("data.csv")
	if err != nil {
		fmt.Println("Failed to open file:", err)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		fmt.Println("Failed to read file:", err)
		return
	}

	for i, record := range records {
		if i == 0 {
			continue
		}

		dateTimeStr := record[0] + " " + record[1]
		requests := record[2]

		dateTime, err := time.Parse("2/1/2006 15:04:05", dateTimeStr)
		if err != nil {
			fmt.Println("Failed to parse date time:", err)
			return
		}

		cronExpr := fmt.Sprintf("%d %d %d %d *", dateTime.Minute(), dateTime.Hour(), dateTime.Day(), int(dateTime.Month()))
		scheduler.AddFunc(cronExpr, func(req string) func() {
			return func() {
				RunApacheBenchmark(req, "1000")
			}
		}(requests))
	}

	scheduler.Start()
}

func RunApacheBenchmark(requests string, concurrency string) {
	cmd := exec.Command("ab", "-n", requests, "-c", concurrency, "http://simulationloadwebtest-473289163.ap-southeast-2.elb.amazonaws.com/status/200")
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("Failed to run Apache Benchmark: %s\nError: %s\n", err, output)
	} else {
		fmt.Printf("Apache Benchmark ran successfully: %s\n", output)
	}
}
func ListJobs(scheduler *cron.Cron) {
	entries := scheduler.Entries()
	fmt.Println("Scheduled Jobs:")
	for _, entry := range entries {
		fmt.Printf("ID: %d, Schedule: %s, Next: %s, Prev: %s\n",
			entry.ID, entry.Schedule, entry.Next, entry.Prev)
	}
}

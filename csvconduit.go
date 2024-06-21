package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type lcLead struct {
	Id string
}

type lcResponse struct {
	Outcome string
	Reason  string
	Lead    lcLead
	Price   float64
}

var csvlogfile *os.File
var csvloglinenumber = 1
var lcSubmissionUrlCheck = regexp.MustCompile("/flows/[a-z0-9]{24}/sources/[a-z0-9]{24}")
var flowIdColumn = -1
var sourceIdColumn = -1

func initLog() {
	// only init once
	if csvlogfile == nil {
		now := time.Now()
		var err error
		csvlogfile, err = os.Create(fmt.Sprintf("log_%s.csv", now.Format("0102_0304")))
		if err != nil {
			panic(err)
		}
		_, err = fmt.Fprintf(csvlogfile, "import_line_num,import_outcome,import_lead_id,import_reason\n")
		if err != nil {
			panic(err)
		}
	}
}

func csvlog(outcome, leadId, reason string) {
	_, err := fmt.Fprintf(csvlogfile, "%d,%s,%s,%s\n", csvloglinenumber, outcome, leadId, reason)
	if err != nil {
		panic(err)
	}
	csvloglinenumber++
}

func getFieldnames(rawRow []string) []string {
	fieldnames := make([]string, len(rawRow))

	for i, field := range rawRow {
		fieldnames[i] = strings.ToLower(strings.ReplaceAll(field, " ", "_"))

		// cache these column IDs, if found
		if fieldnames[i] == "flow_id" {
			flowIdColumn = i
		} else if fieldnames[i] == "source_id" {
			sourceIdColumn = i
		}
	}
	return fieldnames
}

func isFullLcUrl(url string) bool {
	return lcSubmissionUrlCheck.MatchString(url)
}

func getUrl(url string, record []string) string {
	if isFullLcUrl(url) {
		return url
	}
	if flowIdColumn >= 0 && sourceIdColumn >= 0 {
		return fmt.Sprintf("%s/flows/%s/sources/%s/submit", url, record[flowIdColumn], record[sourceIdColumn])
	} else {
		log.Fatal("error: bad URL and no flow or source ID columns set")
		return ""
	}
}

func showPreview(serverUrl string, fieldnames []string, record []string, rowNum int) (proceedFlag int) {
	fmt.Printf("posting URL: %s\n", getUrl(serverUrl, record))
	fmt.Printf("preview of row #%d (note: empty values will not be posted)\n", rowNum)

	// determine the longest field name, so we can center-align the preview
	longestNameLength := 0
	for _, name := range fieldnames {
		if len(name) > longestNameLength {
			longestNameLength = len(name)
		}
	}

	for i, field := range record {
		fmt.Printf("  %*s: %s\n", longestNameLength, fieldnames[i], field)
	}
	fmt.Print("\n")

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("proceed with posting 0, 1, or All remaining rows? (enter 0, 1, or A): ")
	text, _ := reader.ReadString('\n')
	text = strings.ToLower(strings.TrimSpace(text))

	if text == "1" {
		proceedFlag = 1
	} else if text == "a" {
		proceedFlag = 2 // 2 means "all"
	} else {
		// default to 0 (halt) on any other input
		proceedFlag = 0
	}

	return proceedFlag
}

func post(serverUrl string, fieldnames, fields []string, showResponse bool) (outcome string) {
	var leadId, reason string
	values := url.Values{}
	for i, field := range fields {
		if field != "" {
			values.Set(fieldnames[i], field)
		}
	}

	resp, err := http.PostForm(getUrl(serverUrl, fields), values)
	if err != nil {
		outcome = "error"
		reason = err.Error()
	} else {
		defer resp.Body.Close()
		var body []byte
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			outcome = "error"
			reason = err.Error()
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			var lcr lcResponse

			if showResponse {
				fmt.Printf("\n%s\n\n", body)
			}

			err = json.Unmarshal(body, &lcr)
			if err != nil {
				outcome = "error"
				reason = err.Error()
			} else {
				outcome = lcr.Outcome
				leadId = lcr.Lead.Id
				reason = lcr.Reason
			}
		} else {
			outcome = "error"
			reason = strconv.Itoa(resp.StatusCode)
		}
	}

	csvlog(outcome, leadId, reason)
	return
}

func main() {
	// initialize command-line flags
	var showHelp = flag.Bool("help", false, "show help & exit")
	flag.Parse()

	if *showHelp {
		ShowHelp()
	}

	filename := ""
	if len(flag.Args()) > 0 {
		filename = flag.Arg(0)
	} else {
		ShowHelp()
	}
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var serverUrl string
	if len(flag.Args()) > 1 {
		serverUrl = flag.Arg(1)

		// make sure URL is valid
		parsed, parseErr := url.Parse(serverUrl)
		if parseErr != nil || parsed.Scheme == "" || parsed.Host == "" {
			log.Fatalf("invalid URL: %q", serverUrl)
		}
	} else {
		log.Fatal("specify URL to post to")
	}

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		log.Fatal("error reading file: ", err)
	}
	fmt.Printf("read %d data rows\n", len(records)-1) //

	fieldnames := getFieldnames(records[0])

	// make sure either the URL has flow & source ids, or the fieldnames do
	if !isFullLcUrl(serverUrl) && (flowIdColumn < 0 || sourceIdColumn < 0) {
		log.Fatal("required submission info 'flow_id' and 'source_id' not found in URL or CSV")
	}

	proceedFlag := 1 // default to make showPreview run the first time, at least

	var successes, failures, errors int
	statusChar := "."
	for i, dataRow := range records[1:] {

		// show preview 1st time and when user has selected to proceed with 1 row
		if proceedFlag == 1 {
			proceedFlag = showPreview(serverUrl, fieldnames, dataRow, i+1)
			if proceedFlag == 0 {
				csvlogfile.Close()
				break
			}

			// this only needs to happen the first time through, but
			// we only want to create the log if something will be posted
			// (calling it more than once doesn't hurt anything)
			initLog()
		}

		outcome := post(serverUrl, fieldnames, dataRow, proceedFlag == 1)

		// keep score & show progress to stdout
		switch outcome {
		case "success":
			statusChar = "."
			successes++
		case "failure":
			statusChar = "f"
			failures++
		case "error":
			statusChar = "e"
			errors++
		}

		// only show statusChars when posting "all"
		if proceedFlag == 2 {
			fmt.Print(statusChar)
		}
	}

	logfileMsg := ""
	if csvlogfile != nil {
		logfileMsg = fmt.Sprintf("(see %s)", csvlogfile.Name())
	}
	fmt.Printf("\nfinished: %d successes, %d failures, %d errors %s\n",
		successes, failures, errors, logfileMsg)

	csvlogfile.Close()
}

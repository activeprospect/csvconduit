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
var csvloglinenumber int
var lcSubmissionUrlCheck *regexp.Regexp
var flowIdColumn, sourceIdColumn int

func initLog() {
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
	csvloglinenumber = 1
}

func csvlog(outcome, leadId, reason string) {
	_, err := fmt.Fprintf(csvlogfile, "%d,%s,%s,%s\n", csvloglinenumber, outcome, leadId, reason)
	if err != nil {
		panic(err)
	}
	csvloglinenumber++
}

func getFieldnames(rawRow []string) []string {
	flowIdColumn = -1
	sourceIdColumn = -1
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
	lcSubmissionUrlCheck := regexp.MustCompile("/flows/[a-z0-9]{24}/sources/[a-z0-9]{24}")
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

func showPreview(serverUrl string, rowNum int, fieldnames []string, records [][]string) (proceed bool) {
	if rowNum >= len(records) {
		fmt.Printf("cannot preview row number %d in a file with %d rows\n", rowNum, len(records)-1)
	} else {
		fmt.Printf("posting URL: %s\n", getUrl(serverUrl, records[rowNum+1]))
		fmt.Printf("preview using data row #%d (note: empty values will not be posted)\n", rowNum)

		longestNameLength := 0
		for _, name := range fieldnames {
			if len(name) > longestNameLength {
				longestNameLength = len(name)
			}
		}

		for i, field := range records[rowNum+1] {
			fmt.Printf("  %*s: %s\n", longestNameLength, fieldnames[i], field)
		}
		fmt.Print("\n")
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("proceed posting %d rows (Y/n)? ", len(records)-1)
	text, _ := reader.ReadString('\n')
	text = strings.ToLower(strings.TrimSpace(text))

	return text == "" || text == "y"
}

func post(serverUrl string, fieldnames, fields []string) (outcome string) {
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
	// initialize command-line flagsk
	var showHelp = flag.Bool("help", false, "show help & exit")
	var previewRowNum = flag.Int("preview-row", 1, "row-number to show in preview (0 to skip preview)")
	flag.Parse()

	if *showHelp {
		ShowHelp()
	}

	// set regexp global
	lcSubmissionUrlCheck = regexp.MustCompile("/flows/[a-z0-9]{24}/sources/[a-z0-9]{24}")

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
	fmt.Printf("read %d data rows\n", len(records)-1) //

	if err != nil {
		log.Fatal("error reading file: ", err)
	}

	fieldnames := getFieldnames(records[0])

	// make sure either the URL has flow & source ids, or the fieldnames do
	if !isFullLcUrl(serverUrl) && (flowIdColumn < 0 || sourceIdColumn < 0) {
		log.Fatal("required submission info 'flow_id' and 'source_id' not found in URL or CSV")
	}

	if *previewRowNum > 0 {
		proceed := showPreview(serverUrl, *previewRowNum, fieldnames, records)
		if !proceed {
			csvlogfile.Close()
			os.Exit(0)
		}
	}

	initLog()

	var successes, failures, errors int
	for _, dataRow := range records[1:] {
		outcome := post(serverUrl, fieldnames, dataRow)

		// keep score & show progress to stdout
		switch outcome {
		case "success":
			fmt.Print(".")
			successes++
		case "failure":
			fmt.Print("f")
			failures++
		case "error":
			fmt.Print("e")
			errors++
		}
	}
	fmt.Printf("\nfinished: %d successes, %d failures, %d errors (see %s)\n",
		successes, failures, errors, csvlogfile.Name())

	csvlogfile.Close()
}

package main

import (
	"flag"
	"fmt"
	"os"
)

func help() string {
	return `
This utility will read the lines of a CSV (comma-separated value) file, posting
each data line as a lead to ActiveProspect's LeadConduit.

Usage: csvconduit filename.csv leadconduit-url

The CSV file is expected to have a header row; the first line's values 
(normalized to lowercase, with spaces converted to underscores) will
be used as the form post field names. The first data row will be shown as a
preview so that you can ensure everything looks right before starting.

The leadconduit-url may have one of two forms. It can be a full, specific 
submission URL, including the flow and source IDs to post to, such as: 
'https://app.leadconduit.com/flows/FLOW-ID/sources/SOURCE-ID/submit'

Alternatively, if the CSV file includes 'flow_id' and 'source_id' columns, the
leadconduit-url can be the simple base URL of the app: 
'https://app.leadconduit.com'.

Once started, progress will be shown as the file is processed: a period (".") 
for each successful post, the letter "f" for each failure, and the letter "e"
for each error. A record of each result is also written to a CSV log file for
the run, with a line for each input row to help identify which records
from the import had trouble, and why.
`
}

func ShowHelp() {
	fmt.Println(help())
	fmt.Printf("Options:\n")
	flag.PrintDefaults()
	os.Exit(0)
}

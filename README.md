# csvconduit

## about

This utility will read the lines of a CSV (comma-separated value) file, posting
each data line as a lead to ActiveProspect's [LeadConduit](https://activeprospect.com/leadconduit/).

Usage: `csvconduit filename.csv leadconduit-url`

The CSV file is expected to have a header row; the first line's values
(normalized to lowercase, with spaces converted to underscores) will
be used as the form post field names. The first data row will be shown as a
preview so that you can ensure everything looks right before starting.

The `leadconduit-url` may have one of two forms. It can be a full, specific
submission URL, including the flow and source IDs to post to, such as:
'https://app.leadconduit.com/flows/FLOW-ID/sources/SOURCE-ID/submit'

Alternatively, if the CSV file includes `flow_id` and `source_id` columns, the
`leadconduit-url` can be the simple base URL of the app:
'https://app.leadconduit.com'.

Once started, progress will be shown as the file is processed: a period (".")
for each successful post, the letter "f" for each failure, and the letter "e"
for each error. 

A record of each result is also written to a CSV log file for the run, with 
a line for each input row to help identify which records from the import 
had trouble, and why.

## example run

Here's an example run for a small file, which also shows the CSV log format.

```
$ csvconduit import.csv https://app.leadconduit.com/flows/abc/sources/xyz/submit 
read 9 data rows
posting URL: https://app.leadconduit.com/flows/abc/sources/xyz/submit
preview of row #1 (note: empty values will not be posted)
                      address_1: 123 Cornelia Street
                      address_2: 
                          email: taylor@aol.com
                     first_name: Taylor
                      last_name: Swift
                        phone_1: 5125551212

proceed with posting 0, 1, or All remaining rows? (enter 0, 1, or A): a
.....f...
finished: 8 successes, 1 failures, 0 errors (see log_0621_0342.csv)

$ cat log_0621_0342.csv
import_line_num,import_outcome,import_lead_id,import_reason
1,success,6675e5a9265facd94db6a763,
2,success,6675e5a9265facd94db6a766,
3,success,6675e5a9265facd94db6a769,
4,success,6675e5aa265facd94db6a76c,
5,success,6675e5aa265facd94db6a76f,
6,failure,6675e5ab265facd94db6a772,lead.state must not be equal to FL
7,success,6675e5ab265facd94db6a774,
8,success,6675e5ab265facd94db6a777,
9,success,6675e5ab265facd94db6a77a,
```

## installation

Compiled versions of the utility can be found on the [Releases
page](https://github.com/activeprospect/csvconduit/releases).

### MacOS

If you use a newer "Apple Silicon" Mac, download the latest "darwin_arm64.tar.gz" 
version; older Intel Macs will need the "darwin_amd64.tar.gz" one. Open the 
"About This Mac" screen from the Apple menu to check: if it says the chip is 
"Apple M1" (or M2, M3, etc.) then the ARM64 version is the one you need.

Download it, and double-click the file to open it. The command-line utility
is called simply `csvconduit`. When you first try to run it, you'll see a
warning dialog saying **"csvconduit" cannot be opened because it is from an
unidentified developer."** 

Click the question-mark help button. That opens a help window titled "Protect
your Mac from malware", which tells you how to grant this app an exception.
The easiest way is to click the blue link, "Open Privacy & Security
settings for me".

Scroll down past all the different apps listed there, to the "Security" 
section. There will be a little section that says **"csvconduit" was blocked
from use because it is not from an identified developer**. Click the "Open
Anyway" button. This will require your fingerprint (or password), plus clicking
"Open" on one more _are you sure?_ dialog. After that, you should be able
to run it without going through this every time.

## development

With Go installed, build with `go build`.

package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	stashcp "github.com/htcondor/osdf-client/v6"
	"github.com/htcondor/osdf-client/v6/classads"
	"github.com/htcondor/osdf-client/v6/config"
	log "github.com/sirupsen/logrus"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "unknown"
)

type Transfer struct {
	url       string
	localFile string
}

func main() {
	err := config.Init()
	if err != nil {
		log.Errorln(err)
		os.Exit(1)
	}

	// Parse command line arguments
	var upload bool = false
	// Set the options
	stashcp.Options.Recursive = false
	stashcp.Options.ProgressBars = false
	stashcp.Options.Version = version
	if err := setLogging(log.PanicLevel); err != nil {
		log.Panicln("Failed to set log level")
	}
	methods := []string{"cvmfs", "http"}
	var infile, outfile, testCachePath string
	var useOutFile bool = false
	var getCaches bool = false

	// Pop the executable off the args list
	_, os.Args = os.Args[0], os.Args[1:]
	for len(os.Args) > 0 {

		if os.Args[0] == "-classad" {
			// Print classad and exit
			fmt.Println("MultipleFileSupport = true")
			fmt.Println("PluginVersion = \"" + version + "\"")
			fmt.Println("PluginType = \"FileTransfer\"")
			fmt.Println("SupportedMethods = \"stash, osdf\"")
			os.Exit(0)
		} else if os.Args[0] == "-version" || os.Args[0] == "-v" {
			fmt.Println("Version:", version)
			fmt.Println("Build Date:", date)
			fmt.Println("Build Commit:", commit)
			fmt.Println("Built By:", builtBy)
			os.Exit(0)
		} else if os.Args[0] == "-upload" {
			log.Debugln("Upload detected")
			upload = true
		} else if os.Args[0] == "-infile" {
			infile = os.Args[1]
			os.Args = os.Args[1:]
			log.Debugln("Infile:", infile)
		} else if os.Args[0] == "-outfile" {
			outfile = os.Args[1]
			os.Args = os.Args[1:]
			useOutFile = true
			log.Debugln("Outfile:", outfile)
		} else if os.Args[0] == "-d" {
			if err := setLogging(log.DebugLevel); err != nil {
				log.Panicln("Failed to set log level to debug")
			}
		} else if os.Args[0] == "-get-caches" {
			if len(os.Args) < 2 {
				log.Errorln("-get-caches requires an argument")
				os.Exit(1)
			}
			testCachePath = os.Args[1]
			os.Args = os.Args[1:]
			getCaches = true
		} else if strings.HasPrefix(os.Args[0], "-") {
			log.Errorln("Do not understand the option:", os.Args[0])
			os.Exit(1)
		} else {
			// Must be the start of a source / destination
			break
		}
		// Pop off the args
		_, os.Args = os.Args[0], os.Args[1:]
	}

	if getCaches {
		urls, err := stashcp.GetCacheHostnames(testCachePath)
		if err != nil {
			log.Panicln("Failed to get cache URLs:", err)
		}

		cachesToTry := stashcp.CachesToTry
		if cachesToTry > len(urls) {
			cachesToTry = len(urls)
		}

		for _, url := range urls[:cachesToTry] {
			fmt.Println(url)
		}
		os.Exit(0)
	}

	var source []string
	var dest string
	var result error
	//var downloaded int64 = 0
	var transfers []Transfer

	if len(os.Args) == 0 && (infile == "" || outfile == "") {
		fmt.Fprint(os.Stderr, "No source or destination specified\n")
		os.Exit(1)
	}

	if len(os.Args) == 0 {
		// Open the input and output files
		infileFile, err := os.Open(infile)
		if err != nil {
			log.Panicln("Failed to open infile:", err)
		}
		defer infileFile.Close()
		// Read in classad from stdin
		transfers, err = readMultiTransfers(*bufio.NewReader(infileFile))
		if err != nil {
			log.Errorln("Failed to read in from stdin:", err)
			os.Exit(1)
		}
	} else {
		source = os.Args[:len(os.Args)-1]
		dest = os.Args[len(os.Args)-1]
		for _, src := range source {
			transfers = append(transfers, Transfer{url: src, localFile: dest})
		}
	}

	var resultAds []*classads.ClassAd
	retryable := false
	for _, transfer := range transfers {

		var tmpDownloaded int64
		if upload {
			source = append(source, transfer.localFile)
			log.Debugln("Uploading:", transfer.localFile, "to", transfer.url)
			tmpDownloaded, result = stashcp.DoStashCPSingle(transfer.localFile, transfer.url, methods, false)
		} else {
			source = append(source, transfer.url)
			log.Debugln("Downloading:", transfer.url, "to", transfer.localFile)
			tmpDownloaded, result = stashcp.DoStashCPSingle(transfer.url, transfer.localFile, methods, false)
		}
		startTime := time.Now().Unix()
		resultAd := classads.NewClassAd()
		resultAd.Set("TransferStartTime", startTime)
		resultAd.Set("TransferEndTime", time.Now().Unix())
		hostname, _ := os.Hostname()
		resultAd.Set("TransferLocalMachineName", hostname)
		resultAd.Set("TransferProtocol", "stash")
		resultAd.Set("TransferUrl", transfer.url)
		resultAd.Set("TransferFileName", transfer.localFile)
		if upload {
			resultAd.Set("TransferType", "upload")
		} else {
			resultAd.Set("TransferType", "download")
		}
		if result == nil {
			resultAd.Set("TransferSuccess", true)
			resultAd.Set("TransferFileBytes", tmpDownloaded)
			resultAd.Set("TransferTotalBytes", tmpDownloaded)
		} else {
			resultAd.Set("TransferSuccess", false)
			if stashcp.GetErrors() == "" {
				resultAd.Set("TransferError", result.Error())
			} else {
				errMsg := " Failure "
				if upload {
					errMsg += "uploading "
				} else {
					errMsg += "downloading "
				}
				errMsg += transfer.url + ": " + stashcp.GetErrors()
				resultAd.Set("TransferError", errMsg)
				stashcp.ClearErrors()
			}
			resultAd.Set("TransferFileBytes", 0)
			resultAd.Set("TransferTotalBytes", 0)
			if stashcp.ErrorsRetryable() {
				resultAd.Set("TransferRetryable", true)
				retryable = true
			} else {
				resultAd.Set("TransferRetryable", false)
				retryable = false

			}
		}
		resultAds = append(resultAds, resultAd)

	}

	outputFile := os.Stdout
	if useOutFile {
		var err error
		outputFile, err = os.Create(outfile)
		if err != nil {
			log.Panicln("Failed to open outfile:", err)
		}
		defer outputFile.Close()
	}

	success := true
	for _, resultAd := range resultAds {
		_, err := outputFile.WriteString(resultAd.String() + "\n")
		if err != nil {
			log.Panicln("Failed to write to outfile:", err)
		}
		transferSuccess, err := resultAd.Get("TransferSuccess")
		if err != nil {
			log.Errorln("Failed to get TransferSuccess:", err)
			success = false
		}
		success = success && transferSuccess.(bool)
	}

	if success {
		os.Exit(0)
	} else if retryable {
		os.Exit(11)
	} else {
		os.Exit(1)
	}
}

func setLogging(logLevel log.Level) error {
	textFormatter := log.TextFormatter{}
	textFormatter.DisableLevelTruncation = true
	textFormatter.FullTimestamp = true
	log.SetFormatter(&textFormatter)
	log.SetLevel(logLevel)
	return nil
}

// readMultiTransfers reads the transfers from a Reader, such as stdin
func readMultiTransfers(stdin bufio.Reader) (transfers []Transfer, err error) {
	// Check stdin for a list of transfers
	ads, err := classads.ReadClassAd(&stdin)
	if err != nil {
		return nil, err
	}
	if ads == nil {
		return nil, errors.New("No transfers found")
	}
	for _, ad := range ads {
		url, err := ad.Get("Url")
		if err != nil {
			return nil, err
		}
		destination, err := ad.Get("LocalFileName")
		if err != nil {
			return nil, err
		}
		transfers = append(transfers, Transfer{url: url.(string), localFile: destination.(string)})
	}

	return transfers, nil
}

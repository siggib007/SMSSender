package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/siggib007/goutils/logger"
	"github.com/siggib007/goutils/smssender"
	"github.com/siggib007/goutils/utils"
)

func main() {
	// Create default base paths
	objPaths, err := utils.BasePaths()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot determine base paths: "+err.Error())
		os.Exit(3)
	}
	strAppName := objPaths.AppName

	// Load config — three tier: INI -> env vars -> CLI flags
	objCfg := defaultConfig()

	// CLI flags
	strMsgFrom := flag.String("from", "", "What number or string are you sending from")
	strMsgTo := flag.String("to", "", "What number are you sending to")
	strMessage := flag.String("msg", "", "What message do you want to send")
	iVerbose := flag.Int("v", 1, "Verbosity level (1-5)")
	strConfFile := flag.String("c", objPaths.DefConf, "Path to configuration file, defaults to config.ini in the application directory.")
	strBaseURL := flag.String("url", "", "Base URL for API calls")
	bUseEnv := flag.Bool("e", false, "Indicates not to try to load config file, only use environment variables")
	strProxy := flag.String("proxy", "", "Proxy for API calls")
	iTimeout := flag.Int("t", objCfg.TimeOut, "Timeout value on API calls, number of seconds")
	flag.Parse()

	fmt.Print("This is a application to send SMS via Twilio Service.\n")
	fmt.Printf("Running from: %s\n", objPaths.ExeDir)
	fmt.Printf("The time now is %s\n", time.Now().Format("Monday 02 January 2006 15:04:05"))
	fmt.Printf("Logs saved to %s\n", objPaths.DefLogFile)

	// Initialize logger
	objLogger, err := logger.NewLogger(objPaths.DefLogFile, *iVerbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create log file: %s\n", err)
		os.Exit(1)
	}

	defer objLogger.Close()
	defer objLogger.RecoverAbort()
	strAppHost, err := os.Hostname()
	if err != nil {
		objLogger.Log("Failed to determine hostname: " + err.Error())
		strAppHost = "HOSTNAME-LOOKUP-FAILED"
	}

	objLogger.Log(fmt.Sprintf("Starting up application %s on %s", strAppName, strAppHost))
	objLogger.Log(fmt.Sprintf("Verbosity set to %d", *iVerbose))

	if !*bUseEnv {
		objLogger.Log(fmt.Sprintf("Config file set to: %s", *strConfFile))
		bFail := false
		bIsDir, _, err := utils.CheckPath(*strConfFile)
		if err != nil {
			objLogger.LogEntry(fmt.Sprintf("Invalid config path: %v", err), 0, false)
			bFail = true
		}
		if bIsDir {
			objLogger.LogEntry("Config path, is just a directory not a file:", 0, false)
			bFail = true
		}
		if bFail {
			objLogger.Log(fmt.Sprintf("Searching for a viable config file in %v", objPaths.ExeDir))
			lstFiles := utils.FindFilesExt(objPaths.ExeDir, ".ini")
			if len(lstFiles) == 0 {
				objLogger.Log("Failed to find any configuration files in the execution directory")
				*strConfFile = utils.GetInput("Please provide a full path to the desired configuration file, or specify env to use environment variables instead: ")
				if *strConfFile != "env" && (*strConfFile == "" || !utils.FileExists(*strConfFile)) {
					objLogger.LogEntry("Can't go on without a valid configuration file", 0, true)
				}
			} else if len(lstFiles) == 1 {
				objLogger.Log(fmt.Sprintf("Found a possible configuration files, do you want %v ?", lstFiles[0]))
				strResponse := utils.GetInput("Type yes to accept, or provide a full path to the desired configuration file, or specify env to use environment variables instead: ")
				if strResponse == "yes" {
					*strConfFile = filepath.Join(objPaths.ExeDir, lstFiles[0])
				} else {
					*strConfFile = strResponse
				}
				if *strConfFile != "env" && (*strConfFile == "" || !utils.FileExists(*strConfFile)) {
					objLogger.LogEntry("Can't go on without a valid configuration file", 0, true)
				}
			} else {
				objLogger.Log("Found few possible configuration files, would any of these work?")
				for i, strEntry := range lstFiles {
					objLogger.Log(fmt.Sprintf("   %d: %s", i+1, strEntry))
				}
				objLogger.Log(fmt.Sprintf("   %d: Provide manually", len(lstFiles)+1))
				objLogger.Log(fmt.Sprintf("   %d: Use environment variables", len(lstFiles)+2))
				objLogger.Log(fmt.Sprintf("   %d: Abort", len(lstFiles)+3))
				strResponse := utils.GetInput("Type the number of your choice: ")
				strInput := strings.TrimSpace(strResponse)
				iChoice, err := strconv.Atoi(strInput)
				if err != nil {
					objLogger.LogEntry(fmt.Sprintf("Invalid selection %v!! Aborting.", strResponse), 0, true)
				}
				objLogger.Log(fmt.Sprintf("You selected %v", iChoice))
				objLogger.LogEntry(fmt.Sprintf("List len: %v", len(lstFiles)), 3, false)

				if iChoice < 1 || iChoice > len(lstFiles)+3 {
					objLogger.LogEntry(fmt.Sprintf("selection %v out of range!! Aborting.", strResponse), 0, true)
				}
				if iChoice == len(lstFiles)+3 {
					objLogger.LogEntry("OK Got it, bailing", 0, true)
				}
				if iChoice == len(lstFiles)+2 {
					*strConfFile = "env"
				}
				if iChoice == len(lstFiles)+1 {
					*strConfFile = utils.GetInput("Please specify full path for your desired config file: ")
					if *strConfFile != "env" && (*strConfFile == "" || !utils.FileExists(*strConfFile)) {
						objLogger.LogEntry("Can't go on without a valid configuration file", 0, true)
					}
				}
				if iChoice < len(lstFiles)+1 {
					*strConfFile = filepath.Join(objPaths.ExeDir, lstFiles[iChoice-1])
					objLogger.Log(fmt.Sprintf("Conf file is now %v", *strConfFile))
				}
				if *strConfFile != "env" && (*strConfFile == "" || !utils.FileExists(*strConfFile)) {
					objLogger.LogEntry("Can't go on without a valid configuration file", 0, true)
				}
			}
		}
	} else {
		*strConfFile = "env"
	}
	objLogger.Log(fmt.Sprintf("Loading config file %v", *strConfFile))

	objCfg.Verbose = *iVerbose

	if *strConfFile != "env" {
		if err := parseINI(*strConfFile, &objCfg); err != nil {
			objLogger.Log(fmt.Sprintf("Could not read config file %s: %s", *strConfFile, err))
		}
	} else {
		objLogger.Log("Not loading configuration file, relying on environment variables. Make sure they are set correctly")
	}
	applyEnvVars(&objCfg)

	dictFlagsSet := make(map[string]bool)
	flag.Visit(func(objFlag *flag.Flag) {
		dictFlagsSet[objFlag.Name] = true
	})

	// CLI flags override everything
	if *strBaseURL != "" {
		objCfg.BaseURL = *strBaseURL
	}
	if *strProxy != "" {
		objCfg.Proxy = *strProxy
	}
	if *strMsgFrom != "" {
		objCfg.MsgFrom = *strMsgFrom
	}

	if dictFlagsSet["t"] {
		objCfg.TimeOut = *iTimeout
	}

	if *strMsgTo == "" {
		*strMsgTo, err = utils.ReadLine("What number do you want to send to: ")
		if err != nil {
			objLogger.LogEntry(fmt.Sprintf("Failed to read phone number: %v", err), 0, true)
		}
	}
	if *strMessage == "" {
		*strMessage, err = utils.ReadLine("What message are you sending: ")
		if err != nil {
			objLogger.LogEntry(fmt.Sprintf("Failed to read message: %v", err), 0, true)
		}
	}
	objTwilioCfg := smssender.TwilioConfig{}
	objTwilioCfg.BaseURL = objCfg.BaseURL
	objTwilioCfg.ClientID = objCfg.ClientID
	objTwilioCfg.ClientSecret = objCfg.ClientSecret
	objTwilioCfg.MaxMsgLen = objCfg.MaxMsgLen
	objTwilioCfg.MinQuiet = objCfg.MinQuiet
	objTwilioCfg.MsgFrom = objCfg.MsgFrom
	objTwilioCfg.Proxy = objCfg.Proxy
	objTwilioCfg.TimeOut = objCfg.TimeOut

	objSendOptions := smssender.SendOptions{}
	objSendOptions.AppName = objPaths.AppName
	objSendOptions.Message = *strMessage
	objSendOptions.MsgTo = *strMsgTo
	smssender.SendSMS(objSendOptions, objTwilioCfg, objLogger)
}

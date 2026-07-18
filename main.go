package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/siggib007/goutils/comms"
	"github.com/siggib007/goutils/logger"
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

	utils.ValidateConfPath(objLogger, strConfFile, *bUseEnv, *objPaths)

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
	objTwilioCfg := comms.TwilioConfig{}
	objTwilioCfg.BaseURL = objCfg.BaseURL
	objTwilioCfg.ClientID = objCfg.ClientID
	objTwilioCfg.ClientSecret = objCfg.ClientSecret
	objTwilioCfg.MaxMsgLen = objCfg.MaxMsgLen
	objTwilioCfg.MinQuiet = objCfg.MinQuiet
	objTwilioCfg.MsgFrom = objCfg.MsgFrom
	objTwilioCfg.Proxy = objCfg.Proxy
	objTwilioCfg.TimeOut = objCfg.TimeOut

	objSendOptions := comms.SendOptions{}
	objSendOptions.AppName = objPaths.AppName
	objSendOptions.Message = *strMessage
	objSendOptions.MsgTo = *strMsgTo
	if err := comms.SendSMS(objSendOptions, objTwilioCfg, objLogger); err != nil {
		objLogger.LogEntry(err.Error(), 0, true)
	}

}

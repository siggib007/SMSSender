package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/siggib007/goutils/apiclient"
	"github.com/siggib007/goutils/logger"
	"github.com/siggib007/goutils/utils"
)

const iMaxMessageLen = 600

func main() {
	// Create default base paths
	objPaths, err := utils.BasePaths()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot determine base paths: "+err.Error())
		os.Exit(3)
	}
	strScriptName := objPaths.StrScriptName

	// Load config — three tier: INI -> env vars -> CLI flags
	objCfg := defaultConfig()

	// CLI flags
	strMsgFrom := flag.String("from", "", "What number or string are you sending from")
	strMsgTo := flag.String("to", "", "What number are you sending to")
	strMessage := flag.String("msg", "", "What message do you want to send")
	iVerbose := flag.Int("v", 1, "Verbosity level (1-5)")
	strConfFile := flag.String("c", objPaths.StrDefConf, "Path to configuration file, defaults to file with same name as the application in the application directory.")
	strBaseURL := flag.String("url", "", "Base URL for API calls")
	bUseEnv := flag.Bool("e", false, "Indicates not to try to load config file, only use environment variables")
	strProxy := flag.String("proxy", "", "Proxy for API calls")
	iTimeout := flag.Int("t", objCfg.TimeOut, "Timeout value on API calls, number of seconds")
	flag.Parse()

	fmt.Print("This is a script to send SMS via Twilio Service.\n")
	fmt.Printf("Running from: %s\n", objPaths.StrExeDir)
	fmt.Printf("The time now is %s\n", time.Now().Format("Monday 02 January 2006 15:04:05"))
	fmt.Printf("Logs saved to %s\n", objPaths.StrDefLogFile)

	// Initialize logger
	objLogger, err := logger.NewLogger(objPaths.StrDefLogFile, *iVerbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create log file: %s\n", err)
		os.Exit(1)
	}

	defer objLogger.Close()
	defer objLogger.RecoverAbort()
	strScriptHost, err := os.Hostname()
	if err != nil {
		objLogger.Log("Failed to determine hostname: " + err.Error())
		strScriptHost = "HOSTNAME-LOOKUP-FAILED"
	}

	objLogger.Log(fmt.Sprintf("Starting up script %s on %s", strScriptName, strScriptHost))
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
			objLogger.Log(fmt.Sprintf("Searching for a viable config file in %v", objPaths.StrExeDir))
			lstFiles := utils.FindFilesExt(objPaths.StrExeDir, ".ini")
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
					*strConfFile = filepath.Join(objPaths.StrExeDir, lstFiles[0])
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
					*strConfFile = filepath.Join(objPaths.StrExeDir, lstFiles[iChoice-1])
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

	// Validate required config
	if objCfg.BaseURL == "" || objCfg.ClientID == "" || objCfg.ClientSecret == "" {
		objLogger.LogEntry("No URL or API auth config, exiting", 0, true)
	}

	if *strMsgTo == "" {
		*strMsgTo, err = ReadLine("What number do you want to send to: ")
		if err != nil {
			objLogger.LogEntry(fmt.Sprintf("Failed to read phone number: %v", err), 0, true)
		}
	}
	if *strMessage == "" {
		*strMessage, err = ReadLine("What message are you sending: ")
		if err != nil {
			objLogger.LogEntry(fmt.Sprintf("Failed to read message: %v", err), 0, true)
		}
	}

	if err := ValidateAlphanumericSenderId(objCfg.MsgFrom); err != nil {
		objLogger.LogEntry(err.Error(), 0, true)
	}

	strPhone, err := SanitizePhone(*strMsgTo)
	if err != nil {
		objLogger.LogEntry(err.Error(), 0, true)
	}

	strMsg, err := SanitizeSmsBody(*strMessage)
	if err != nil {
		objLogger.LogEntry(err.Error(), 0, true)
	}

	objValues := url.Values{}
	objValues.Set("From", objCfg.MsgFrom)
	objValues.Set("Body", strMsg)
	objValues.Set("To", strPhone)
	strEncoded := objValues.Encode()

	objAPI := apiclient.NewAPIClient(objCfg.Proxy, objCfg.TimeOut, objCfg.MinQuiet, objLogger)
	dictHeader := make(map[string]string)
	dictHeader["Content-Type"] = "application/x-www-form-urlencoded"
	dictHeader["Accept"] = "*/*"
	dictHeader["Application"] = strScriptName
	dictHeader["User-Agent"] = fmt.Sprintf("Go/%s", strScriptName)
	dictMyParams := make(map[string]string)
	strURL := apiclient.BuildURL(objCfg.BaseURL, objCfg.ClientID+"/Messages.json", dictMyParams)
	objCallOptions := apiclient.APICallOptions{}
	objCallOptions.StrURL = strURL
	objCallOptions.DictHeader = dictHeader
	objCallOptions.StrMethod = "POST"
	objCallOptions.StrRawBody = strEncoded
	objCallOptions.StrUser = objCfg.ClientID
	objCallOptions.StrPWD = objCfg.ClientSecret

	objLogger.Log("Posting Message")
	objResp := objAPI.MakeAPICall(objCallOptions)
	if !objResp.BSuccess {
		objLogger.LogEntry(fmt.Sprintf("Failed to send message: %s", objResp.StrError), 0, true)
	}
	dictResp, ok := objResp.ObjData.(map[string]any)
	if !ok {
		objLogger.LogEntry("Unexpected response format", 0, true)
	}
	strStatus, ok := dictResp["status"].(string)
	if !ok {
		objLogger.LogEntry("No status in response", 0, true)
	}
	objLogger.Log(fmt.Sprintf("Status: %v", strStatus))
}

var reNonDigit = regexp.MustCompile(`[^0-9]`)

// SanitizePhone strips formatting characters and validates that what's
// left looks like a plausible phone number. Returns an error (not nil)
// on any failure, so callers can fail loud instead of silently proceeding.
func SanitizePhone(strInput string) (string, error) {
	strTrimmed := strings.TrimSpace(strInput)
	if strTrimmed == "" {
		return "", fmt.Errorf("phone number is empty")
	}

	bHasLeadingPlus := strings.HasPrefix(strTrimmed, "+")

	strDigitsOnly := reNonDigit.ReplaceAllString(strTrimmed, "")
	if strDigitsOnly == "" {
		return "", fmt.Errorf("phone number %q contains no digits", strInput)
	}

	iLen := len(strDigitsOnly)
	if iLen < 7 || iLen > 15 {
		return "", fmt.Errorf("phone number %q has %d digits, expected 7-15", strInput, iLen)
	}

	strResult := strDigitsOnly
	if bHasLeadingPlus {
		strResult = "+" + strDigitsOnly
	}

	return strResult, nil
}

// SanitizeSmsBody removes control characters that have no legitimate
// place in message text, while leaving normal language, punctuation,
// and unicode untouched. Returns an error if the message is empty,
// oversized, or made entirely of characters that got stripped.
func SanitizeSmsBody(strInput string) (string, error) {
	if strInput == "" {
		return "", fmt.Errorf("message body is empty")
	}

	strCleaned := strings.Map(func(rChar rune) rune {
		if rChar == '\n' || rChar == '\t' {
			return rChar
		}
		if unicode.IsControl(rChar) {
			return -1
		}
		return rChar
	}, strInput)

	strTrimmed := strings.TrimSpace(strCleaned)
	if strTrimmed == "" {
		return "", fmt.Errorf("message body contained no usable text after sanitization")
	}

	iLen := len([]rune(strTrimmed))
	if iLen > iMaxMessageLen {
		return "", fmt.Errorf("message body is %d characters, exceeds max of %d", iLen, iMaxMessageLen)
	}

	return strTrimmed, nil
}

const iMaxSenderIdLen = 11

var reValidSenderIdChars = regexp.MustCompile(`^[A-Za-z0-9 &_-]+$`)

// ValidateAlphanumericSenderId enforces Twilio's alphanumeric sender ID
// requirements: up to 11 characters, letters/digits/spaces plus
// hyphen, underscore, and ampersand only. Returns an error describing
// the specific violation rather than a bare pass/fail.
func ValidateAlphanumericSenderId(strSenderId string) error {
	if strSenderId == "" {
		return fmt.Errorf("sender ID is empty")
	}

	iLen := len(strSenderId)
	if iLen > iMaxSenderIdLen {
		return fmt.Errorf("sender ID %q is %d characters, exceeds max of %d", strSenderId, iLen, iMaxSenderIdLen)
	}

	if !reValidSenderIdChars.MatchString(strSenderId) {
		return fmt.Errorf("sender ID %q contains characters outside the allowed set (letters, digits, space, -, _, &)", strSenderId)
	}

	return nil
}

// ReadLine prompts on stdout (if strPrompt is non-empty) and reads a
// single line from stdin, spaces and all. Returns an error if stdin
// is closed/exhausted before a line is read, or if the scanner itself
// fails (e.g. an underlying I/O error).
func ReadLine(strPrompt string) (string, error) {
	if strPrompt != "" {
		fmt.Print(strPrompt)
	}

	objScanner := bufio.NewScanner(os.Stdin)

	bHasLine := objScanner.Scan()
	if !bHasLine {
		objErr := objScanner.Err()
		if objErr != nil {
			return "", fmt.Errorf("failed to read line: %w", objErr)
		}
		return "", fmt.Errorf("no input received (stdin closed)")
	}

	strLine := objScanner.Text()
	return strLine, nil
}

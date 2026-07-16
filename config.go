package main

import (
	"os"
	"strconv"
	"strings"

	"gopkg.in/ini.v1"
)

type Config struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	Environment  string
	Attachments  string
	InFile       string
	Proxy        string
	CSVDelim     rune
	Deductible   bool
	EmployeeID   string
	LogFile      string
	ConfFile     string
	Verbose      int
	MinQuiet     int
	TimeOut      int
}

func defaultConfig() Config {
	return Config{
		CSVDelim:   ',',
		Deductible: true,
		EmployeeID: "name",
		Verbose:    1,
		MinQuiet:   2,
		TimeOut:    15,
	}
}

func parseINI(strPath string, objCfg *Config) error {
	objFile, err := ini.Load(strPath)
	if err != nil {
		return err
	}

	objSec := objFile.Section("")
	if strValue := objSec.Key("API_URL").String(); strValue != "" {
		objCfg.BaseURL = strValue
	}
	if strValue := objSec.Key("CLIENT_ID").String(); strValue != "" {
		objCfg.ClientID = strValue
	}
	if strValue := objSec.Key("CLIENT_SECRET").String(); strValue != "" {
		objCfg.ClientSecret = strValue
	}
	if strValue := objSec.Key("ATTACHMENTS").String(); strValue != "" {
		objCfg.Attachments = strValue
	}
	if strValue := objSec.Key("IN_FILE").String(); strValue != "" {
		objCfg.InFile = strValue
	}
	if strValue := objSec.Key("PROXY").String(); strValue != "" {
		objCfg.Proxy = strValue
	}
	if strValue := objSec.Key("CSV_DELIM").String(); strValue != "" {
		objCfg.CSVDelim = rune(strValue[0])
	}
	if strValue := objSec.Key("DEDUCTABLE").String(); strValue != "" {
		objCfg.Deductible = strings.ToLower(strValue) == "true"
	}
	if strValue := objSec.Key("EMPLOYEE_ID").String(); strValue != "" {
		objCfg.EmployeeID = strValue
	}
	if strValue := objSec.Key("Environment").String(); strValue != "" {
		objCfg.Environment = strValue
	}
	if iValue, err := objSec.Key("TimeOut").Int(); err == nil {
		objCfg.TimeOut = iValue
	}
	if iValue, err := objSec.Key("MinQuiet").Int(); err == nil {
		objCfg.MinQuiet = iValue
	}
	return nil
}

func applyEnvVars(cfg *Config) {
	if strValue := os.Getenv("API_URL"); strValue != "" {
		cfg.BaseURL = strValue
	}
	if strValue := os.Getenv("CLIENT_ID"); strValue != "" {
		cfg.ClientID = strValue
	}
	if strValue := os.Getenv("CLIENT_SECRET"); strValue != "" {
		cfg.ClientSecret = strValue
	}
	if strValue := os.Getenv("ATTACHMENTS"); strValue != "" {
		cfg.Attachments = strValue
	}
	if strValue := os.Getenv("CSV_DELIM"); strValue != "" {
		cfg.CSVDelim = rune(strValue[0])
	}
	if strValue := os.Getenv("DEDUCTABLE"); strValue != "" {
		cfg.Deductible = strings.ToLower(strValue) == "true"
	}
	if strValue := os.Getenv("PROXY"); strValue != "" {
		cfg.Proxy = strValue
	}
	if strValue := os.Getenv("EMPLOYEE_ID"); strValue != "" {
		cfg.EmployeeID = strValue
	}
	if strValue := os.Getenv("TIMEOUT"); strValue != "" {
		iVal, err := strconv.Atoi(strValue)
		if err == nil {
			cfg.TimeOut = iVal
		}
	}
}

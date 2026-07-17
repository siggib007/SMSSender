package main

import (
	"os"
	"strconv"

	"gopkg.in/ini.v1"
)

type Config struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	MsgFrom      string
	Proxy        string
	LogFile      string
	ConfFile     string
	Verbose      int
	MinQuiet     int
	TimeOut      int
	MaxMsgLen    int
}

func defaultConfig() Config {
	return Config{
		Verbose:   1,
		MinQuiet:  2,
		TimeOut:   15,
		MaxMsgLen: 800,
	}
}

func parseINI(strPath string, objCfg *Config) error {
	objFile, err := ini.Load(strPath)
	if err != nil {
		return err
	}

	objSec := objFile.Section("")
	if strValue := objSec.Key("BaseURL").String(); strValue != "" {
		objCfg.BaseURL = strValue
	}
	if strValue := objSec.Key("TwilioSID").String(); strValue != "" {
		objCfg.ClientID = strValue
	}
	if strValue := objSec.Key("TwilioToken").String(); strValue != "" {
		objCfg.ClientSecret = strValue
	}
	if strValue := objSec.Key("MsgFrom").String(); strValue != "" {
		objCfg.MsgFrom = strValue
	}

	if strValue := objSec.Key("PROXY").String(); strValue != "" {
		objCfg.Proxy = strValue
	}

	if iValue, err := objSec.Key("TimeOut").Int(); err == nil {
		objCfg.TimeOut = iValue
	}
	if iValue, err := objSec.Key("MinQuiet").Int(); err == nil {
		objCfg.MinQuiet = iValue
	}
	if iValue, err := objSec.Key("MaxMsgLen").Int(); err == nil {
		objCfg.MaxMsgLen = iValue
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
	if strValue := os.Getenv("MSG_FROM"); strValue != "" {
		cfg.MsgFrom = strValue
	}

	if strValue := os.Getenv("PROXY"); strValue != "" {
		cfg.Proxy = strValue
	}
	if strValue := os.Getenv("TIMEOUT"); strValue != "" {
		iVal, err := strconv.Atoi(strValue)
		if err == nil {
			cfg.TimeOut = iVal
		}
	}
	if strValue := os.Getenv("MAXMSG"); strValue != "" {
		iVal, err := strconv.Atoi(strValue)
		if err == nil {
			cfg.MaxMsgLen = iVal
		}
	}

}

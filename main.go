package main

import (
	"flag"
	"log"
	"os"
	"strings"
	"time"

	lumberjack "gopkg.in/natefinch/lumberjack.v2"

	"github.com/clearblade/BLE-ADAPTER-GO/bleadapter"
	cb "github.com/clearblade/Go-SDK"
	"github.com/hashicorp/logutils"
)

var (
	platformURL  string
	messagingURL string
	sysKey       string
	sysSec       string
	deviceName   string //See if we can get the edge device name dynamically
	password     string
	scanInterval int
	logLevel     string

	deviceClient *cb.DeviceClient
)

const (
	platURL = "http://localhost:9000"
	messURL = "localhost:1883"
)

func init() {
	flag.StringVar(&sysKey, "systemKey", "", "system key (required)")
	flag.StringVar(&sysSec, "systemSecret", "", "system secret (required)")
	flag.StringVar(&deviceName, "deviceName", "", "name of device (required)")
	flag.StringVar(&password, "password", "", "password (or active key) for device authentication (required)")
	flag.StringVar(&platformURL, "platformURL", platURL, "platform url (optional)")
	flag.StringVar(&messagingURL, "messagingURL", messURL, "messaging URL (optional)")
	flag.IntVar(&scanInterval, "scanInterval", 360, "The number of seconds to scan for BLE devices (optional)")
	flag.StringVar(&logLevel, "logLevel", "warn", "The level of logging to use. Available levels are 'debug', 'warn', 'error' (optional)")
}

func usage() {
	log.Printf("Usage: ble-adapter [options]\n\n")
	flag.PrintDefaults()
}

func validateFlags() {
	flag.Parse()
	if sysKey == "" ||
		sysSec == "" ||
		deviceName == "" ||
		password == "" {

		log.Printf("[ERROR] Missing required flags\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if logLevel != "error" && logLevel != "warn" && logLevel != "debug" {
		log.Printf("[ERROR] Invalid log level specified\n\n")
		flag.Usage()
		os.Exit(1)
	}
}

//ClearBlade Device Client init helper
func initCbDeviceClient() {
	log.Printf("[DEBUG] setting platform URL to %s", platformURL)
	log.Printf("[DEBUG] setting messaging URL to %s", messagingURL)

	deviceClient = cb.NewDeviceClientWithAddrs(platformURL, messagingURL, sysKey, sysSec, deviceName, password)

	for err := deviceClient.Authenticate(); err != nil; {
		log.Printf("[WARN] Error authenticating platform broker: %s", err.Error())
		log.Printf("[WARN] Will retry in 1 minute...")

		// sleep 1 minute
		time.Sleep(time.Duration(time.Minute * 1))
		err = deviceClient.Authenticate()
	}
}

func main() {
	filter := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DEBUG", "WARN", "ERROR"},
		MinLevel: logutils.LogLevel(strings.ToUpper(logLevel)),
		Writer:   os.Stderr,
	}
	log.SetOutput(filter)

	flag.Usage = usage
	log.Printf("[DEBUG] Validating command line options")
	validateFlags()

	_, err := os.OpenFile("/var/log/bleadapter.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)

	if err != nil {
		log.Printf("error opening file: %s", err.Error())
		os.Exit(1)
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	//Set rolling log files
	log.SetOutput(&lumberjack.Logger{
		Filename:   "/var/log/bleadapter.log",
		MaxSize:    10, // megabytes
		MaxBackups: 5,
		MaxAge:     28, //days
	})

	bleAdapter := bleadapter.BleAdapter{}

	log.Printf("[DEBUG] Initializing CB device client")
	initCbDeviceClient()

	log.Printf("[DEBUG] Starting BLE Adapter")
	bleAdapter.Start(deviceClient, scanInterval)
}

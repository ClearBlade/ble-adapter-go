package main

import (
	"flag"
	"log"
	"os"

	"github.com/clearblade/BLE-ADAPTER-GO/bleadapter"
	cb "github.com/clearblade/Go-SDK"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

var (
	platformURL  string
	messagingURL string
	sysKey       string
	sysSec       string
	deviceName   string //See if we can get the edge device name dynamically
	password     string
	scanInterval int

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

		log.Printf("Missing required flags\n\n")
		flag.Usage()
		os.Exit(1)
	}
}

//create and initialize the clearblade platform device client
func initCbDeviceClient() {
	deviceClient = cb.NewDeviceClient(sysKey, sysSec, deviceName, password)

	if platformURL != "" {
		log.Printf("setting custom platform URL to %s", platformURL)
		deviceClient.HttpAddr = platformURL
	}

	if messagingURL != "" {
		log.Printf("setting custom messaging URL to %s", messagingURL)
		deviceClient.MqttAddr = messagingURL
	}
}

func main() {
	_, err := os.OpenFile("/var/log/bleadapter.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)

	if err != nil {
		log.Printf("error opening file: %s", err.Error())
		os.Exit(1)
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	//Set rolling log files
	log.SetOutput(&lumberjack.Logger{
		Filename:   "/var/log/bleadapter.log",
		MaxSize:    100, // megabytes
		MaxBackups: 5,
		MaxAge:     28, //days
	})

	flag.Usage = usage
	validateFlags()

	initCbDeviceClient()
	if err := deviceClient.Authenticate(); err != nil {
		log.Fatalf("Error authenticating: %s", err.Error())
	}

	if err := deviceClient.InitializeMQTT("bleadapter_"+deviceName, "", 30, nil, nil); err != nil {
		log.Fatalf("Unable to initialize MQTT: %s", err.Error())
	}

	bleAdapter := bleadapter.BleAdapter{}
	bleAdapter.Start(deviceClient, scanInterval)
}

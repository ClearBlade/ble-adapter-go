package main

import (
	"flag"
	"log"
	"os"

	"github.com/clearblade/BLE-ADAPTER-GO/bleadapter"
	cb "github.com/clearblade/Go-SDK"
)

var (
	platformURL  = platURL
	messagingURL = messURL
	sysKey       string
	sysSec       string
	deviceName   string //See if we can get the edge device name dynamically
	password     string

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
	flag.StringVar(&platformURL, "platformURL", "", "platform url (optional)")
	flag.StringVar(&messagingURL, "messagingURL", "", "messaging URL (optional)")
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
		log.Println("setting custom platform URL to ", platformURL)
		deviceClient.HttpAddr = platformURL
	}

	if messagingURL != "" {
		log.Println("setting custom messaging URL to ", messagingURL)
		deviceClient.MqttAddr = messagingURL
	}
}

func main() {
	flag.Usage = usage
	validateFlags()

	//TODO - This would need a developer ID. May need to create a service account
	//within the platform.
	// If the command line arguments are valid, we need to verify the status
	// of the data collections. If they do not exist, they need to be created
	//

	initCbDeviceClient()

	log.Println("Authenticating to platform with device ", deviceName)
	log.Println("Authenticating to platform with password ", password)

	if err := deviceClient.Authenticate(); err != nil {
		log.Fatalf("Error authenticating: %s", err.Error())
	}
	log.Printf("%+v\n", deviceClient)

	if err := deviceClient.InitializeMQTT("bleadapter_"+deviceName, "", 30, nil, nil); err != nil {
		log.Fatalf("Unable to initialize MQTT: %s", err.Error())
	}

	bleAdapter := bleadapter.BleAdapter{
		CbDeviceClient: deviceClient,
	}

	bleAdapter.ScanForDevices()
}

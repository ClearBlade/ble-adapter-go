package bleadapter

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/clearblade/BLE-ADAPTER-GO/ble"
	cb "github.com/clearblade/Go-SDK"
)

var (
	uuidFilters  []string
	publishTopic = devicePublishTopic
)

const (
	deviceFiltersCollectionName = "BLE_Device_Filters"
	adapterConfigCollectionName = "BLE_Adapter_Config"
	devicePublishTopic          = "/bleadapter/bledevice"
	msgPublishQos               = 2
	deviceManufacturerData      = "Manufacturer"
	deviceAddress               = "Address"
	deviceAlias                 = "Alias"
	deviceUUIDs                 = "UUIDs"
	deviceRSSI                  = "RSSI"
)

//BleAdapter - Struct that represents a BLE Adapter
type BleAdapter struct {
	connection     *ble.Connection
	CbDeviceClient *cb.DeviceClient

	//platformURL         string
	//messagingURL        string
	//systemKey           string
	//systemSecret        string
	//authDeviceName      string //See if we can get the edge name dynamically
	//authDeviceActiveKey string
}

//ScanForDevices - Scan for ble devices
func (adapt *BleAdapter) ScanForDevices() {
	//Retrieve the UUID's to filter on
	//uuidFilters := []string{}

	uuidFilters := adapt.getDeviceFilters()
	log.Println("UUID Filters retrieved = ", uuidFilters)

	//Retrieve the adapter configuration information
	adapt.getAdapterConfig()

	//Open a connection to the System Dbus to begin scanning
	log.Println("Creating DBUS connection")
	connection, err := ble.Open()
	if err != nil {
		log.Fatal(err)
	}

	var deviceChannel chan *ble.Device
	filterString := strings.TrimSpace(strings.Join(uuidFilters, " "))

	log.Println("Starting BLE Discovery")
	if filterString != "" {
		deviceChannel, err = connection.InitiateDiscovery(filterString)
	} else {
		deviceChannel, err = connection.InitiateDiscovery()
	}

	if err != nil {
		log.Fatal(err)
	}

	log.Println("Waiting for BLE Devices")
	for device := range deviceChannel {
		log.Println("Device added = %#v", *device)
		if deviceJSON, err := adapt.createBleDeviceJSON(*device); err != nil {
			log.Printf("error marshaling device into json: %s", err.Error())
		} else {
			log.Printf("Publishing message: %s", deviceJSON)

			//Publish each device to the specified topic
			if err := adapt.CbDeviceClient.Publish(publishTopic, deviceJSON, msgPublishQos); err != nil {
				log.Printf("Error occurred when publishing device to MQTT: %v", err)
			}
		}
	}
}

func (adapt *BleAdapter) getDeviceFilters() []string {
	//Retrieve the uuids that we wish to filter on
	//var query cb.Query //A nil query results in all rows being returned
	results, err := adapt.CbDeviceClient.GetDataByName(deviceFiltersCollectionName, &cb.Query{})

	dataLength := len(results["DATA"].([]interface{}))

	if err != nil || dataLength == 0 {
		log.Println("No device filters enabled.")
		return []string{}
	}

	uuids := []string{}

	for i, uuid := range results["DATA"].([]interface{}) {
		if results["DATA"].([]interface{})[i].(map[string]interface{})["enabled"].(bool) == true {
			uuids = append(uuids, uuid.(map[string]interface{})["ble_uuid"].(string))
		}
	}

	return uuids
}

func (adapt *BleAdapter) getAdapterConfig() error {
	//Retrieve the adapter configuration row
	//var query cb.Query //A nil query results in all rows being returned
	results, err := adapt.CbDeviceClient.GetDataByName(adapterConfigCollectionName, &cb.Query{})
	if err != nil {
		log.Println("Adapter configuration could not be retrieved. Using defaults")
		return err
	}

	publishTopic = results["DATA"].([]interface{})[0].(map[string]interface{})["publish_topic"].(string)
	return nil
}

func (adapt *BleAdapter) createBleDeviceJSON(device ble.Device) ([]byte, error) {

	//Create json to publish to mqtt
	//Need manufacturer data, RSSI,
	bleDevice := map[string]interface{}{}
	bleDevice[deviceAddress] = device.Address()
	bleDevice[deviceAlias] = device.Alias()
	bleDevice[deviceUUIDs] = device.UUIDs()
	bleDevice[deviceRSSI] = device.RSSI()
	bleDevice[deviceManufacturerData] = device.ManufacturerData() //Need to stringify this

	return json.Marshal(bleDevice)
}

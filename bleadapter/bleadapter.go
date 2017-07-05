package bleadapter

import (
	"encoding/json"
	"log"
	"time"

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
	devicePath                  = "path"
	deviceManufacturerData      = "manufacturer"
	deviceAddress               = "address"
	deviceAlias                 = "alias"
	deviceUUIDs                 = "uuids"
	deviceRSSI                  = "rssi"

	//Devices advertise at specific intervals. This should be set to at least 2N, where N is the
	//amount of time associated with the advertising interval.
	defaultScanInterval = 6

	//http://www.bluez.org/bluez-5-api-introduction-and-porting-guide/
	//Once the discovery stops, devices neither connected to or paired will be automatically removed
	//by bluetoothd within three minutes.
	discoveryPauseInterval = 1
)

//BleAdapter - Struct that represents a BLE Adapter
type BleAdapter struct {
	connection     *ble.Connection
	cbDeviceClient *cb.DeviceClient
	scanInterval   int64
	deviceChannel  chan *ble.Device
}

func (adapt *BleAdapter) Start(devClient *cb.DeviceClient, theScanInterval int) {
	adapt.cbDeviceClient = devClient

	if theScanInterval > 0 {
		log.Printf("Setting scan interval to %d", theScanInterval)
		adapt.scanInterval = int64(theScanInterval)

	} else {
		log.Printf("Setting scan interval to default value %d", defaultScanInterval)
		adapt.scanInterval = defaultScanInterval
	}

	//Retrieve the adapter configuration from the CB Platform data collection
	adapt.getAdapterConfig()

	stopDiscoveryChannel := make(chan bool)
	stopHandleDevicesChannel := make(chan bool)
	defer close(stopDiscoveryChannel)
	defer close(stopHandleDevicesChannel)

	for true {
		adapt.scanForDevices(stopDiscoveryChannel, stopHandleDevicesChannel)

		// wait until the interval elapses
		interval := time.Duration(int64(adapt.scanInterval) * time.Minute.Nanoseconds())
		time.Sleep(interval)

		//Write to the stopDiscovery channel so that scanning is stopped
		stopDiscoveryChannel <- true

		// wait until the interval elapses
		interval = time.Duration(int64(discoveryPauseInterval) * time.Minute.Nanoseconds())
		time.Sleep(interval)
	}

	//TODO - Add code to connect to devices, pair with devices,
	//read/write to/from devices
	//
	// 1. Create a channel to use to subscribe to BLE topics
	// 2. Inifinitely read from channel and respond to BLE device requests
	//
}

//scanForDevices - Scan for ble devices
func (adapt *BleAdapter) scanForDevices(stopDiscoveryChannel <-chan bool, stopHandleDevicesChannel <-chan bool) {
	//Retrieve the UUID's to filter on
	uuidFilters := adapt.getDeviceFilters()
	log.Println("UUID Filters retrieved = ", uuidFilters)

	//Open a connection to the System Dbus to begin scanning
	var err error
	if adapt.connection, err = ble.Open(); err != nil {
		log.Fatal(err)
	}

	if adapt.deviceChannel = adapt.connection.StartDiscovery(stopDiscoveryChannel, uuidFilters...); adapt.deviceChannel == nil {
		log.Fatal("Could not initiate discovery, shutting down BLE Adapter.")
	}

	//Start a separate process to listen for ble device related events
	go adapt.handleDeviceSignal(stopHandleDevicesChannel)
}

//ScanForDevices - Scan for ble devices
func (adapt *BleAdapter) handleDeviceSignal(stopHandleDevicesChannel <-chan bool) {
	log.Println("Waiting for BLE Devices")

	for {
		select {
		case device, ok := <-adapt.deviceChannel:
			//TODO - Handle added, removed, and changed devices
			if ok {
				adapt.handleDeviceAdded(device)
			}
		case stopChannel, ok := <-stopHandleDevicesChannel:
			if ok && stopChannel {
				//End the current go routine when the stop discovery signal is received
				return
			}
		}
	}
}

func (adapt *BleAdapter) handleDeviceAdded(device *ble.Device) {
	log.Println("Device added = %#v", *device)
	if deviceJSON, err := adapt.createBleDeviceJSON(*device); err != nil {
		log.Printf("error marshaling device into json: %s", err.Error())
	} else {
		log.Printf("Publishing message: %s", deviceJSON)

		if err := adapt.cbDeviceClient.Publish(publishTopic, deviceJSON, msgPublishQos); err != nil {
			log.Printf("Error occurred when publishing device to MQTT: %v", err)
		}
	}
}

func (adapt *BleAdapter) handleDeviceRemoved(device *ble.Device) {
}

func (adapt *BleAdapter) getDeviceFilters() []string {
	//Retrieve the uuids that we wish to filter on
	//var query cb.Query //A nil query results in all rows being returned
	results, err := adapt.cbDeviceClient.GetDataByName(deviceFiltersCollectionName, &cb.Query{})

	if err != nil || len(results["DATA"].([]interface{})) == 0 {
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
	//Retrieve the adapter configuration row. Passing a nil query results in all rows being returned
	results, err := adapt.cbDeviceClient.GetDataByName(adapterConfigCollectionName, &cb.Query{})
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
	bleDevice[devicePath] = device.Path()
	bleDevice[deviceAddress] = device.Address()
	bleDevice[deviceAlias] = device.Alias()
	bleDevice[deviceUUIDs] = device.UUIDs()
	bleDevice[deviceRSSI] = device.RSSI()
	bleDevice[deviceManufacturerData] = device.ManufacturerData() //Need to stringify this

	return json.Marshal(bleDevice)
}

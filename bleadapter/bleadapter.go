package bleadapter

import (
	"encoding/json"
	"log"
	"strings"
	"time"

	cb "github.com/clearblade/Go-SDK"
	"github.com/clearblade/ble-adapter-go/ble"
)

var (
	uuidFilters  []string
	publishTopic = devicePublishTopic

	//Devices advertise at specific intervals. This should be set to at least 2N, where N is the
	//amount of time associated with the advertising interval.
	scanInterval int64 = 360 //seconds

	//http://www.bluez.org/bluez-5-api-introduction-and-porting-guide/
	//Once the discovery stops, devices neither connected to or paired will be automatically removed
	//by bluetoothd within three minutes.
	pauseInterval int64 = 60 //seconds

	handleRemoved = false //Should the InterfacesRemoved signal be handled
	handleChanged = false //Should the PropertiesChanged signal be handled
)

const (
	//DBus signals
	addrule        = "type='signal',interface='org.freedesktop.DBus.ObjectManager',member='InterfacesAdded'"
	removerule     = "type='signal',interface='org.freedesktop.DBus.ObjectManager',member='InterfacesRemoved'"
	propertiesrule = "type='signal',interface='org.freedesktop.DBus.Properties',member='PropertiesChanged'"

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
)

//BleAdapter - Struct that represents a BLE Adapter
type BleAdapter struct {
	connection     *ble.Connection
	cbDeviceClient *cb.DeviceClient
	deviceChannel  chan *ble.Device
}

//Start - Starts execution of the BLEAdapter
func (adapt *BleAdapter) Start(devClient *cb.DeviceClient, theScanInterval int) {
	adapt.cbDeviceClient = devClient

	if theScanInterval > 0 {
		scanInterval = int64(theScanInterval)
	}

	stopDiscoveryChannel := make(chan bool)
	stopHandleDevicesChannel := make(chan bool)
	defer close(stopDiscoveryChannel)
	defer close(stopHandleDevicesChannel)

	for true {
		//Retrieve the adapter configuration from the CB Platform data collection
		adapt.getAdapterConfig()
		log.Printf("Beginning scan. Scan duration = %d", scanInterval)

		adapt.scanForDevices(stopDiscoveryChannel, stopHandleDevicesChannel)

		// wait until the interval elapses
		interval := time.Duration(int64(scanInterval) * time.Second.Nanoseconds())
		time.Sleep(interval)

		if err := adapt.removeDbusEvents(); err != nil {
			return
		}

		//Write to the stopDiscovery channel so that scanning is stopped
		stopDiscoveryChannel <- true

		// wait until the interval elapses
		log.Printf("Beginning pause. Pause duration = %d", pauseInterval)
		interval = time.Duration(int64(pauseInterval) * time.Second.Nanoseconds())
		time.Sleep(interval)
	}

	//TODO - Add code to connect to devices, pair with devices,
	//read/write to/from devices
	//
	// 1. Create a channel to use to subscribe to BLE topics
	// 2. Inifinitely read from channel and respond to BLE device requests
	//
}

func (adapt *BleAdapter) addDbusEvents() error {
	var err error
	if err = adapt.connection.AddMatch(addrule); err != nil {
		log.Printf("Error adding InterfacesAdded match: %s", err.Error())
		return err
	}

	if handleRemoved == true {
		if err = adapt.connection.AddMatch(removerule); err != nil {
			log.Printf("Error adding InterfacesRemoved match %s", err.Error())
			return err
		}
	}

	if handleChanged == true {
		if err = adapt.connection.AddMatch(propertiesrule); err != nil {
			log.Printf("Error adding PropertiesChanged match %s", err.Error())
			return err
		}
	}

	return nil
}

func (adapt *BleAdapter) removeDbusEvents() error {
	var err error
	if err = adapt.connection.RemoveMatch(addrule); err != nil {
		log.Printf("Error removing InterfacesAdded match: %s", err.Error())
		return err
	}

	if handleRemoved == true {
		if err = adapt.connection.RemoveMatch(removerule); err != nil {
			log.Printf("Error removing InterfacesRemoved match %s", err.Error())
			return err
		}
	}

	if handleChanged == true {
		if err = adapt.connection.RemoveMatch(propertiesrule); err != nil {
			log.Printf("Error removing PropertiesChanged match %s", err.Error())
			return err
		}
	}
	return nil
}

//scanForDevices - Scan for ble devices
func (adapt *BleAdapter) scanForDevices(stopDiscoveryChannel <-chan bool, stopHandleDevicesChannel <-chan bool) {
	//Retrieve the UUID's to filter on.  If an error is encountered, use the filters that were previously specified
	theFilters, err := adapt.getDeviceFilters()

	if err != nil {
		log.Printf("Error encountered while retrieving UUID Filters: %s", err.Error())
	} else {
		log.Printf("UUID Filters retrieved = #%v", uuidFilters)
		uuidFilters = theFilters
	}

	//Open a connection to the System Dbus to begin scanning
	if adapt.connection, err = ble.Open(); err != nil {
		log.Fatal(err)
	}

	//Add the DBus events the adapter should listen for
	if err := adapt.addDbusEvents(); err != nil {
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
	log.Printf("Waiting for BLE Devices")

	for {
		select {
		case device, ok := <-adapt.deviceChannel:
			//Handle added, removed, and changed devices
			if ok {
				if string((*device).Path()) == "" {
					//Device Properties changed
					adapt.handleDeviceRemoved(device)
				} else {
					//Device Added or changed
					adapt.handleDeviceAddedChanged(device)
				}
			}
		case stopChannel, ok := <-stopHandleDevicesChannel:
			if ok && stopChannel {
				//End the current go routine when the stop discovery signal is received
				return
			}
		}
	}
}

func (adapt *BleAdapter) handleDeviceAddedChanged(device *ble.Device) {
	if adapt.shouldPublishDevice(*device) == true {
		if deviceJSON, err := adapt.createBleDeviceJSON(*device); err != nil {
			log.Printf("error marshaling device into json: %s", err.Error())
		} else {
			log.Printf("Publishing message: %s", deviceJSON)

			if err := adapt.cbDeviceClient.Publish(adapt.cbDeviceClient.DeviceName+"/"+publishTopic, deviceJSON, msgPublishQos); err != nil {
				log.Printf("Error occurred when publishing device to MQTT: %v", err)
			}
		}
	} else {
		log.Printf("Device does not contain any uuid specified in the uuid filter. Skipping device: %#v", device)
	}
}

func (adapt *BleAdapter) shouldPublishDevice(device ble.Device) bool {

	//If device uuid filters were specified, ensure one of the UUID's exists
	//in the uuids property for the device.
	if len(uuidFilters) == 0 {
		return true
	}

	//Loop over the uuid filter array
	for _, uuid := range uuidFilters {
		deviceUuids := device.UUIDs()

		//If there are no uuids specified for the device, skip this device
		if len(uuidFilters) == 0 {
			return false
		}
		for _, deviceuuid := range deviceUuids {
			if strings.ToUpper(deviceuuid) == strings.ToUpper(uuid) {
				return true
			}
		}
	}

	return false
}

func (adapt *BleAdapter) handleDeviceRemoved(device *ble.Device) {
	log.Printf("Device removed = %#v", *device)
}

func (adapt *BleAdapter) getDeviceFilters() ([]string, error) {
	//Retrieve the uuids that we wish to filter on
	//var query cb.Query - A nil query results in all rows being returned
	results, err := adapt.cbDeviceClient.GetDataByName(deviceFiltersCollectionName, &cb.Query{})

	if err != nil {
		return nil, err
	}

	if len(results["DATA"].([]interface{})) == 0 {
		log.Printf("No device filters enabled.")
	}

	uuids := []string{}

	for i, uuid := range results["DATA"].([]interface{}) {
		if results["DATA"].([]interface{})[i].(map[string]interface{})["enabled"].(bool) == true {
			uuids = append(uuids, uuid.(map[string]interface{})["ble_uuid"].(string))
		}
	}

	return uuids, nil
}

func (adapt *BleAdapter) getAdapterConfig() error {
	//Retrieve the adapter configuration row. Passing a nil query results in all rows being returned
	results, err := adapt.cbDeviceClient.GetDataByName(adapterConfigCollectionName, &cb.Query{})
	if err != nil {
		log.Printf("Adapter configuration could not be retrieved. Using defaults. Error: %s", err.Error())
		return err
	}

	publishTopic = results["DATA"].([]interface{})[0].(map[string]interface{})["publish_topic"].(string)

	if results["DATA"].([]interface{})[0].(map[string]interface{})["discovery_scan_seconds"] != nil {
		scanInterval = int64(results["DATA"].([]interface{})[0].(map[string]interface{})["discovery_scan_seconds"].(float64))
	}

	if results["DATA"].([]interface{})[0].(map[string]interface{})["discovery_pause_seconds"] != nil {
		pauseInterval = int64(results["DATA"].([]interface{})[0].(map[string]interface{})["discovery_pause_seconds"].(float64))
	}

	if results["DATA"].([]interface{})[0].(map[string]interface{})["handle_removed"] != nil &&
		results["DATA"].([]interface{})[0].(map[string]interface{})["handle_removed"] == true {
		handleRemoved = true
	} else {
		handleRemoved = false
	}

	if results["DATA"].([]interface{})[0].(map[string]interface{})["handle_changed"] != nil &&
		results["DATA"].([]interface{})[0].(map[string]interface{})["handle_changed"] == true {
		handleChanged = true
	} else {
		handleChanged = false
	}

	return nil
}

func (adapt *BleAdapter) createBleDeviceJSON(device ble.Device) ([]byte, error) {

	//Create json to publish to mqtt
	bleDevice := map[string]interface{}{}
	bleDevice[devicePath] = device.Path()
	bleDevice[deviceAddress] = device.Address()
	bleDevice[deviceAlias] = device.Alias()
	bleDevice[deviceUUIDs] = device.UUIDs()
	bleDevice[deviceRSSI] = device.RSSI()
	bleDevice[deviceManufacturerData] = device.ManufacturerData() //Need to stringify this

	return json.Marshal(bleDevice)
}

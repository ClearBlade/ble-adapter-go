package bleadapter

import (
	"encoding/json"
	"log"
	"strings"
	"time"

	cb "github.com/clearblade/Go-SDK"
	cbble "github.com/clearblade/ble-adapter-go/ble"
	mqttTypes "github.com/clearblade/mqtt_parsing"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/godbus/dbus"
)

var (
	uuidFilters     []string
	publishTopic    = devicePublishTopic
	subscribeTopic  = deviceSubscribeTopic
	mqttIsConnected = false

	//Devices advertise at specific intervals. This should be set to at least 2N, where N is the
	//amount of time associated with the advertising interval.
	scanInterval int64 = 360 //seconds

	//http://www.bluez.org/bluez-5-api-introduction-and-porting-guide/
	//Once the discovery stops, devices neither connected to or paired will be automatically removed
	//by bluetoothd within three minutes.
	pauseInterval int64 = 60 //seconds

	handleRemoved = false //Should the InterfacesRemoved signal be handled
	handleChanged = false //Should the PropertiesChanged signal be handled

	//Channel used to send a signal to stop ble discovery
	stopDiscoveryChannel chan bool

	//Channel used to send a signal to stop listening for ble commands
	stopBleCommandsChannel chan bool

	//Channel used to send a signal to stop listening for ble commands
	stopScanLoopChannel chan bool
)

const (
	deviceFiltersCollectionName = "BLE_Device_Filters"
	adapterConfigCollectionName = "BLE_Adapter_Config"
	devicePublishTopic          = "bleadapter/bledevice"
	deviceSubscribeTopic        = "bleadapter/bledevice/command"
	messagingQos                = 2
	devicePath                  = "path"
	deviceManufacturerData      = "manufacturer"
	deviceAddress               = "address"
	deviceAlias                 = "alias"
	deviceUUIDs                 = "uuids"
	deviceRSSI                  = "rssi"
)

//BleAdapter - Struct that represents a BLE Adapter
type BleAdapter struct {
	connection     *cbble.Connection
	cbDeviceClient *cb.DeviceClient

	//Channel used to receive ble device discovery related signals
	deviceChannel chan *dbus.Signal

	//Channel used to receive ble related commands (read/write) from the platform
	bleCommandsChannel <-chan *mqttTypes.Publish
}

//Start - Starts execution of the BLEAdapter
func (adapt *BleAdapter) Start(devClient *cb.DeviceClient, theScanInterval int) {
	adapt.cbDeviceClient = devClient

	log.Printf("[DEBUG] Initializing MQTT with callbacks")
	var callbacks = &cb.Callbacks{OnConnectionLostCallback: adapt.OnConnectLost, OnConnectCallback: adapt.OnConnect}
	if err := adapt.cbDeviceClient.InitializeMQTTWithCallback("bleadapter_"+adapt.cbDeviceClient.DeviceName, "", 30, nil, nil, callbacks); err != nil {
		log.Fatalf("[ERROR] initCbClient: Unable to initialize MQTT connection: %s", err.Error())
	}

	if theScanInterval > 0 {
		scanInterval = int64(theScanInterval)
	}

	//Open a connection to the System Dbus to begin scanning
	var connErr error
	if adapt.connection, connErr = cbble.Open(); connErr != nil {
		log.Fatal("[ERROR] " + connErr.Error())
	}

	stopDiscoveryChannel = make(chan bool)

	//Clean up after ourselves
	defer close(stopDiscoveryChannel)

	//Make sure we close the dbus connection
	defer adapt.connection.Close()

	for true {
		//If the MQTT Client is not connected to the platform broker,
		//there's no need to scan.
		if mqttIsConnected {
			log.Printf("[DEBUG] MQTT is connected.")

			//Refresh the list of managed dbus objects so that updates to the
			//Discovering property are properly reflected
			adapt.connection.Update()
			if deviceAdapter, adaptErr := adapt.connection.GetAdapter(); adaptErr != nil {
				log.Printf("[ERROR] Device BLE adapter could not be retrieved: %s", adaptErr.Error())
				log.Printf("[DEBUG] Waiting 30 seconds before retrying device BLE adapter retrieval.")
				time.Sleep(time.Duration(30 * time.Second.Nanoseconds()))
			} else {
				if deviceAdapter.Discovering() == false {
					log.Printf("[DEBUG] Device ble adapter is not discovering.")

					//Retrieve the adapter configuration from the CB Platform data collection
					adapt.getAdapterConfig()
					log.Printf("Beginning scan. Scan duration = %d", scanInterval)

					stopScanLoopChannel = make(chan bool)

					adapt.scanForDevices(stopDiscoveryChannel)

					//If a scan interval was specified wait until the interval elapses
					var timer *time.Timer
					if scanInterval > 0 {
						timer = time.AfterFunc(time.Duration(int64(scanInterval)*time.Second.Nanoseconds()), func() {
							adapt.stopDiscoveryScan()
						})
					}

					//Wait for the stop loop command
					for stopLoop := range stopScanLoopChannel {
						if timer != nil {
							timer.Stop()
						}
						if stopLoop {
							log.Printf("[DEBUG] Stopping the scan loop")
							close(stopScanLoopChannel)
						} else {
							log.Printf("[DEBUG] Invalid value received for stoploop: %t", stopLoop)
						}
					}

					if scanInterval > 0 && pauseInterval > 0 {
						// wait until the pause interval elapses
						log.Printf("Beginning pause. Pause duration = %d", pauseInterval)
						time.Sleep(time.Duration(int64(pauseInterval) * time.Second.Nanoseconds()))
					}
				} else {
					log.Printf("[DEBUG] Device ble adapter is STILL discovering.")
					time.Sleep(time.Duration(5 * time.Second.Nanoseconds()))
				}
			}
		} else {
			log.Printf("[DEBUG] Cannot start BLE Scan, waiting 10 seconds for MQTT connection to be established.")
			time.Sleep(time.Duration(10 * time.Second.Nanoseconds()))
		}
		log.Printf("[DEBUG] Starting next loop iteration")
	}
}

//stopDiscoveryScan - Stop the BLE discovery process
func (adapt *BleAdapter) stopDiscoveryScan() {
	//Remove the dbus events prior to stopping discovery so that a write to
	//a closed channel does not occurr
	if err := adapt.removeDbusEvents(); err != nil {
		log.Printf("[ERROR] Error removing DBUS events: %s", err.Error())
	}

	//End the existing goRoutines
	log.Printf("[DEBUG] Stopping BLE discovery")
	stopDiscoveryChannel <- true
	stopScanLoopChannel <- true

	log.Printf("[DEBUG] Returning from stopDiscoveryScan")
}

//addDbusEvents - Add DBUS signals we wish to handle to the DBUS connection
func (adapt *BleAdapter) addDbusEvents() error {
	log.Printf("[DEBUG] Adding DBUS events")
	var err error
	if err = adapt.connection.AddMatch(cbble.AddRule); err != nil {
		log.Printf("[ERROR] Error adding InterfacesAdded match: %s", err.Error())
		return err
	}

	if handleRemoved == true {
		if err = adapt.connection.AddMatch(cbble.RemoveRule); err != nil {
			log.Printf("[ERROR] Error adding InterfacesRemoved match %s", err.Error())
			return err
		}
	}

	if handleChanged == true {
		if err = adapt.connection.AddMatch(cbble.PropertiesRule); err != nil {
			log.Printf("[ERROR] Error adding PropertiesChanged match %s", err.Error())
			return err
		}
	}

	return nil
}

//removeDbusEvents - Remove DBUS signal matches from the DBUS connection
func (adapt *BleAdapter) removeDbusEvents() error {
	log.Printf("[DEBUG] Removing DBUS events")
	var err error
	if err = adapt.connection.RemoveMatch(cbble.AddRule); err != nil {
		log.Printf("[ERROR] Error removing InterfacesAdded match: %s", err.Error())
		return err
	}

	if handleRemoved == true {
		if err = adapt.connection.RemoveMatch(cbble.RemoveRule); err != nil {
			log.Printf("[ERROR] Error removing InterfacesRemoved match %s", err.Error())
			return err
		}
	}

	if handleChanged == true {
		if err = adapt.connection.RemoveMatch(cbble.PropertiesRule); err != nil {
			log.Printf("[ERROR] Error removing PropertiesChanged match %s", err.Error())
			return err
		}
	}
	return nil
}

//scanForDevices - Scan for ble devices
func (adapt *BleAdapter) scanForDevices(stopDiscoveryChannel <-chan bool) {
	//Retrieve the UUID's to filter on.  If an error is encountered, use the filters that were previously specified
	theFilters, err := adapt.getDeviceFilters()

	if err != nil {
		log.Printf("[ERROR] Error encountered while retrieving UUID Filters: %s", err.Error())
	} else {
		log.Printf("[DEBUG] UUID Filters retrieved = #%v", uuidFilters)
		uuidFilters = theFilters
	}

	//Add the DBus events the adapter should listen for
	if err := adapt.addDbusEvents(); err != nil {
		log.Fatal("[ERROR] Error adding DBUS event: " + err.Error())
	}

	if adapt.deviceChannel = adapt.connection.StartDiscovery(stopDiscoveryChannel, uuidFilters...); adapt.deviceChannel == nil {
		log.Fatal("[ERROR] Could not initiate discovery, shutting down BLE Adapter.")
	}

	//Start a separate process to listen for ble device discovery related signals
	go adapt.handleDBUSSignal()
}

//handleDBUSSignal - Wait for DBUS signals to be broadcasted from DBUS
func (adapt *BleAdapter) handleDBUSSignal() {
	log.Printf("Waiting for BLE Devices")

	//Range over the device channel. When the channel is closed
	//this goroutine will end. The channel is closed automatically
	//when discovery is stopped
	for dbussignal := range adapt.deviceChannel {
		log.Printf("[DEBUG] DBUS signal received: %#v", dbussignal)
		switch dbussignal.Name {
		case cbble.InterfacesAdded:
			HandleInterfaceAdded(*adapt, dbussignal)
		case cbble.InterfacesRemoved:
			HandleInterfaceRemoved(*adapt, dbussignal)
		case cbble.PropertiesChanged:
			HandlePropertyChanged(*adapt, dbussignal)
		}
	}

	log.Printf("[DEBUG] adapt.deviceChannel closed. Ending goroutine")
	return
}

//publishDevice
//		1. Retrieve the BLE device from the DBUS object cache
//		2. Verify the device contains the appropriate UUIDs
//		3. Create a JSON representation for the device
//		4. Publish the JSON to the platform
func (adapt *BleAdapter) publishDevice(address string) {
	//Refresh the list of managed objects
	if err := adapt.connection.Update(); err != nil {
		log.Printf("[Error]Error updating object cache: %#v", err)
	}

	if device, geterr := adapt.connection.GetDeviceByAddress(address); geterr == nil {
		if adapt.shouldPublishDevice(&device) == true {
			if deviceJSON, jsonerr := adapt.createBleDeviceJSON(&device); jsonerr != nil {
				log.Printf("[ERROR] error marshaling device into json: %s", jsonerr.Error())
			} else {
				log.Printf("Publishing message: %s", deviceJSON)

				if puberr := adapt.cbDeviceClient.Publish(adapt.cbDeviceClient.DeviceName+"/"+publishTopic, deviceJSON, messagingQos); puberr != nil {
					log.Printf("[ERROR] Error occurred when publishing device to MQTT: %v", puberr)
				}
			}
		} else {
			log.Printf("[WARN] Device does not contain any uuid specified in the uuid filter. Skipping device: %#v", device)
		}
	} else {
		log.Printf(geterr.Error())
	}
}

//shouldPublishDevice - Ensure the device contains one of the UUIDs that are being filtered on
func (adapt *BleAdapter) shouldPublishDevice(device *cbble.Device) bool {

	//If device uuid filters were specified, ensure one of the UUID's exists
	//in the uuids property for the device.
	if len(uuidFilters) == 0 {
		log.Printf("[DEBUG] No UUIDs being filtered on. shouldPublishDevice returning true")
		return true
	}

	//Loop over the uuid filter array
	for _, uuid := range uuidFilters {
		deviceUuids := (*device).UUIDs()

		//If there are no uuids specified for the device, skip this device
		if len(uuidFilters) == 0 {
			log.Printf("[DEBUG] No UUIDs on device. shouldPublishDevice returning false")
			return false
		}
		for _, deviceuuid := range deviceUuids {
			if strings.ToUpper(deviceuuid) == strings.ToUpper(uuid) {
				log.Printf("[DEBUG] UUID found on device. shouldPublishDevice returning true")
				return true
			}
		}
	}

	return false
}

//getDeviceFilters - Retrieve the UUIDs that should be filtered on
func (adapt *BleAdapter) getDeviceFilters() ([]string, error) {
	//Retrieve the uuids that we wish to filter on
	//var query cb.Query - A nil query results in all rows being returned
	results, err := adapt.cbDeviceClient.GetDataByName(deviceFiltersCollectionName, &cb.Query{})

	if err != nil {
		return nil, err
	}

	if len(results["DATA"].([]interface{})) == 0 {
		log.Printf("[DEBUG] No device filters enabled.")
	}

	uuids := []string{}

	for i, uuid := range results["DATA"].([]interface{}) {
		if results["DATA"].([]interface{})[i].(map[string]interface{})["enabled"].(bool) == true {
			//DBUS uses lowercase characters in the uuids. Ensure we convert them to lowercase
			uuids = append(uuids, strings.ToLower(uuid.(map[string]interface{})["ble_uuid"].(string)))
		}
	}

	log.Printf("[DEBUG] Returning UUIDs to filter on: %#v", uuids)
	return uuids, nil
}

//getAdapterConfig - Retrieve BLE Adapter configuration parameters from a platform data collection
func (adapt *BleAdapter) getAdapterConfig() error {
	//Retrieve the adapter configuration row. Passing a nil query results in all rows being returned
	results, err := adapt.cbDeviceClient.GetDataByName(adapterConfigCollectionName, &cb.Query{})
	if err != nil {
		log.Printf("[WARN] Adapter configuration could not be retrieved. Using defaults. Error: %s", err.Error())
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

//createBleDeviceJSON - Create a JSON representation of a BLE device
func (adapt *BleAdapter) createBleDeviceJSON(device *cbble.Device) ([]byte, error) {
	log.Printf("[DEBUG] Creating device JSON")

	//Create json to publish to mqtt
	bleDevice := map[string]interface{}{}
	bleDevice[devicePath] = (*device).Path()
	bleDevice[deviceAddress] = (*device).Address()
	bleDevice[deviceAlias] = (*device).Alias()
	bleDevice[deviceUUIDs] = (*device).UUIDs()

	if rssi := (*device).RSSI(); rssi != -1 {
		bleDevice[deviceRSSI] = rssi
	}

	bleDevice["interface"] = (*device).Interface()
	bleDevice["name"] = (*device).Name()

	if icon := (*device).Icon(); icon != "" {
		bleDevice["icon"] = icon
	}

	if class := (*device).Class(); class != 0 {
		bleDevice["class"] = class
	}

	if app := (*device).Appearance(); app != 0 {
		bleDevice["appearance"] = app
	}

	if modalias := (*device).Modalias(); modalias != "" {
		bleDevice["modalias"] = modalias
	}

	if txPower := (*device).TxPower(); txPower != -1 {
		bleDevice["txPower"] = txPower
	}

	bleDevice[deviceManufacturerData] = (*device).ManufacturerData()
	bleDevice["serviceData"] = (*device).ServiceData()
	bleDevice["servicesResolved"] = (*device).ServicesResolved()

	if advFlags := (*device).AdvertisingFlags(); len(advFlags) > 0 {
		bleDevice["advertisingFlags"] = advFlags
	}

	bleDevice["paired"] = (*device).Paired()
	bleDevice["connected"] = (*device).Connected()
	bleDevice["trusted"] = (*device).Trusted()
	bleDevice["blocked"] = (*device).Blocked()
	bleDevice["adapter"] = (*device).Adapter()
	bleDevice["legacyPairing"] = (*device).LegacyPairing()

	return json.Marshal(bleDevice)
}

//handleBLECommands - Goroutine used to listen for BLE commands sent from the platform
func (adapt *BleAdapter) handleBLECommands() {
	//Wait for BLE Commands to be received from the platform.
	//
	// The structure of the command payload will need to resemble the following:
	//
	// {
	//		"command": "read" | "write"
	//		"deviceAddress": MAC address
	//		"devicePath": ""
	//		"gattCharacteristic" - (uuid)
	//		"gattCharacteristicValue"
	//		"stayConnected" - true|false
	// }
	//
	log.Printf("Waiting for BLE Commands")

	//As a command comes in, we need to start a new goroutine to handle the command

	//We have to use the stopBleCommandsChannel because the bleCommandsChannel,
	//returned from MQTT subscription, is read-only and we can't close it.
	//Disconnecting from the message broker isn't closing the channel
	for {
		select {
		case message, ok := <-adapt.bleCommandsChannel:
			//Process ble commands sent from the platform
			if ok {
				log.Printf("[DEBUG] BLE command received")

				//Start a goroutine to process the command
				go adapt.processBLECommand(message)
			}
		case stopChannel, ok := <-stopBleCommandsChannel:
			log.Printf("[DEBUG] Stop handleBLECommands received, value = %t", stopChannel)
			log.Printf("[DEBUG] Channel ok value = %t", ok)
			if !ok || stopChannel {
				//End the current go routine when the stop signal is received
				log.Printf("[DEBUG] Stopping BLE command handler")
				return
			}
		}
	}
}

//processBLECommand - Goroutine used to process individual BLE commands sent from the platform
func (adapt *BleAdapter) processBLECommand(message *mqttTypes.Publish) {

	//Separate goroutine to handle individual ble commands
	log.Printf("[DEBUG] Processing BLE command")

	var blecommand map[string]interface{}

	err := json.Unmarshal(message.Payload, &blecommand)
	if err != nil {
		log.Printf("[ERROR] Invalid JSON received for BLE Command: %s", err.Error())

		blecommand = make(map[string]interface{})

		//Send an error back to the platform
		//Create a new JSON command
		blecommand["command"] = ""
		blecommand["err"] = true
		blecommand["sentCommand"] = string(message.Payload)

		//Create a new BLECommand instance
		bleCmd := NewBLECommand(adapt, blecommand)
		bleCmd.sendError("Invalid JSON received for BLE Command")
		return
	}

	log.Printf("[DEBUG] Received BLE %s Command", blecommand["command"])

	//Refresh the list of managed objects
	if err := adapt.connection.Update(); err != nil {
		log.Printf("[Error]Error updating object cache: %#v", err)
	}

	//Create a new BLECommand instance
	bleCmd := NewBLECommand(adapt, blecommand)

	if err := bleCmd.Execute(); err != nil {
		log.Printf("[ERROR] Error while executing ble command: %s", err.Error())
		bleCmd.sendError("BLE command failed. " + err.Error())
		return
	}

	log.Printf("[DEBUG] BLE command success")
	bleCmd.sendSuccess("BLE command " + bleCmd.command["command"].(string) + " executed successfully")
	return
}

//OnConnectLost - MQTT callback invoked when a connection to a broker is lost
//If the connection to the broker is lost, we need to reconnect and
//re-establish all of the subscriptions
func (adapt *BleAdapter) OnConnectLost(client MQTT.Client, connerr error) {
	log.Printf("[WARN] Connection to broker was lost: %s", connerr.Error())

	mqttIsConnected = false

	//Stop ble scanning
	adapt.stopDiscoveryScan()

	//End the existing goRoutines
	log.Printf("[DEBUG] Stopping BLE commands channel")
	stopBleCommandsChannel <- true

	//Close the existing channels
	log.Printf("[DEBUG] Closing BLE commands channel")
	close(stopBleCommandsChannel)

	//We don't need to worry about manally re-initializing the mqtt client. The auto reconnect logic will
	//automatically try and reconnect. The reconnect interval could be as much as 20 minutes.
}

//OnConnect - MQTT callback invoked when a connection is established with a broker
//When the connection to the broker is complete, set up the subscriptions
func (adapt *BleAdapter) OnConnect(client MQTT.Client) {
	log.Printf("Connected to ClearBlade Platform MQTT broker")
	mqttIsConnected = true

	log.Printf("[DEBUG] Begin Configuring Subscription(s)")

	var err error
	log.Printf("[DEBUG] device client: %#v", adapt.cbDeviceClient)
	log.Printf("[DEBUG] topic: %s", adapt.cbDeviceClient.DeviceName+"/"+subscribeTopic)
	log.Printf("[DEBUG] qos: %d", messagingQos)

	for adapt.bleCommandsChannel, err = adapt.cbDeviceClient.Subscribe(adapt.cbDeviceClient.DeviceName+"/"+subscribeTopic, messagingQos); err != nil; {
		log.Printf("[WARN] Error subscribing to topics: %s", err.Error())

		//Wait 30 seconds and retry
		log.Printf("[DEBUG] Waiting 30 seconds to retry subscriptions")
		time.Sleep(time.Duration(30 * time.Second))
		adapt.bleCommandsChannel, err = adapt.cbDeviceClient.Subscribe(adapt.cbDeviceClient.DeviceName+"/"+subscribeTopic, messagingQos)
	}

	stopBleCommandsChannel = make(chan bool)

	//Start the goRoutine to listen for ble commands published to the Platform
	log.Printf("[DEBUG] Starting ble command listener")
	go adapt.handleBLECommands()
}

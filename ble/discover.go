package ble

import (
	"log"
	"strings"

	"github.com/godbus/dbus"
)

//AddMatch - Adds a signal matching rule to DBUS. Allows a specific type of DBUS signal to be handled within a program.
func (conn *Connection) AddMatch(rule string) error {
	return conn.bus.BusObject().Call(
		"org.freedesktop.DBus.AddMatch",
		0,
		rule,
	).Err
}

//RemoveMatch - Removes a signal matching rule from DBUS.
func (conn *Connection) RemoveMatch(rule string) error {
	return conn.bus.BusObject().Call(
		"org.freedesktop.DBus.RemoveMatch",
		0,
		rule,
	).Err
}

// StartDiscovery - Initiates discovery of LE peripherals with the given UUIDs.
func (conn *Connection) StartDiscovery(stopDiscoveryChannel <-chan bool, uuids ...string) chan *Device {

	//Create the channel that will be used to return DBUS signal events to the caller
	//This channel is close when the Discover method ends
	deviceDiscoveredChannel := make(chan *Device)

	//Retrieve the device ble adapter from DBUS
	adapter, err := conn.GetAdapter()
	if err != nil {
		log.Printf("Error retrieving device ble adapter %s", err)
		return nil
	}

	go adapter.Discover(deviceDiscoveredChannel, stopDiscoveryChannel, uuids...)
	return deviceDiscoveredChannel
}

// Discover puts the adapter in discovery mode, waits for the specified amount of time to discover
// one of the given UUIDs, and then stops discovery mode.
func (adapter *blob) Discover(deviceChannel chan<- *Device, stopDiscoveryChannel <-chan bool, uuids ...string) {

	conn := adapter.conn
	signals := make(chan *dbus.Signal)
	conn.bus.Signal(signals)

	//Declare deferreds so that we don't leave anything hanging around
	defer conn.bus.RemoveSignal(signals)

	defer close(deviceChannel)
	defer close(signals)

	var err error

	log.Printf("Setting discovery filter")
	if err = adapter.SetDiscoveryFilter(uuids...); err != nil {
		log.Printf("Error setting discovery filter: %s", err.Error())
		return
	}

	log.Printf("Starting device discovery")
	if err = adapter.StartDiscovery(); err != nil {
		log.Printf("Error starting device discovery: %s", err.Error())
		return
	}

	log.Printf("Starting discoverDevicesLoop")
	if err = adapter.discoverLoop(deviceChannel, uuids, signals, stopDiscoveryChannel); err != nil {
		log.Printf("Error returned from discoverDevicesLoop: %s", err.Error())
		return
	}

	return
}

func (adapter *blob) discoverLoop(deviceChannel chan<- *Device, uuids []string, signals <-chan *dbus.Signal, stopDiscoveryChannel <-chan bool) error {
	for {
		select {
		case s := <-signals:
			switch s.Name {
			case interfacesAdded:
				//Refresh the list of managed objects
				if err := adapter.conn.Update(); err != nil {
					log.Printf("Error updating object cache for added devices: %#v", err)
					return err
				}

				//Get the interface added properties
				props := interfaceProperties(s)

				if theDevice, err := adapter.Conn().GetDeviceByAddress(props["Address"].Value().(string)); err == nil {
					deviceChannel <- &theDevice
				} else {
					log.Printf(err.Error())
				}
			case interfacesRemoved:
				//&dbus.Signal{
				//	Sender:":1.7",
				//	Path:"/",
				//	Name:"org.freedesktop.DBus.ObjectManager.InterfacesRemoved",
				//	Body:[]interface {}{
				//		"/org/bluez/hci0/dev_A0_E6_F8_8B_A8_6F",
				//		[]string{
				//			"org.freedesktop.DBus.Properties",
				//			"org.freedesktop.DBus.Introspectable",
				//			"org.bluez.Device1"
				//		}
				//	}
				//}
				//Get the interface removed properties
				props := interfaceProperties(s)
				if props[deviceInterface].Value() != nil {
					//Create a ble.Device object and write it to the channel
					removedDevice := new(Device)
					removedDeviceBlob := new(blob)
					removedDeviceBlob.properties = make(map[string]dbus.Variant)
					removedDeviceBlob.properties["Address"] = props[deviceInterface]

					*removedDevice = removedDeviceBlob
					deviceChannel <- removedDevice
				}
			case propertiesChanged:
				if props := interfaceProperties(s); props != nil {
					//Refresh the list of managed objects
					if err := adapter.conn.Update(); err != nil {
						log.Printf("Error updating object cache for properties changed: %v", err)
						return err
					}

					if address := parseAddressFromPath(string(s.Path)); address != "" {
						if theDevice, err := adapter.Conn().GetDeviceByAddress(address); err == nil {
							deviceChannel <- &theDevice
						} else {
							log.Printf(err.Error())
						}
					}
				}
			default:
				log.Printf("%s: unexpected signal %s", adapter.Name(), s.Name)
			}
		case stopChannel := <-stopDiscoveryChannel:
			if stopChannel {
				adapter.StopDiscovery()

				return nil
			}
		}
	}
}

// If the signal contains deviceInterface,
// return the corresponding properties, otherwise nil.
// See http://dbus.freedesktop.org/doc/dbus-specification.html#standard-interfaces-objectmanager
func interfaceProperties(s *dbus.Signal) properties {
	switch s.Name {
	case interfacesAdded:
		var dict map[string]map[string]dbus.Variant
		//&dbus.Signal{Sender:":1.3", Path:"/", Name:"org.freedesktop.DBus.ObjectManager.InterfacesAdded",
		//Body:[]interface {}{"/org/bluez/hci0/dev_A0_E6_F8_8A_4D_5C",
		//map[string]map[string]dbus.Variant{"org.bluez.Device1":map[string]dbus.Variant{"Trusted":dbus.Variant{sig:dbus.Signature{str:"b"},
		//value:false}, "Blocked":dbus.Variant{sig:dbus.Signature{str:"b"}, value:false},
		//"LegacyPairing":dbus.Variant{sig:dbus.Signature{str:"b"}, value:false},
		//"RSSI":dbus.Variant{sig:dbus.Signature{str:"n"}, value:-59}, "UUIDs":dbus.Variant{sig:dbus.Signature{str:"as"},
		//value:[]string{"32f9169f-4feb-4883-ade6-1f0127018db3"}}, "Adapter":dbus.Variant{sig:dbus.Signature{str:"o"},
		//value:"/org/bluez/hci0"}, "ManufacturerData":dbus.Variant{sig:dbus.Signature{str:"a{qv}"},
		//value:map[uint16]dbus.Variant{0x5c60:dbus.Variant{sig:dbus.Signature{str:"ay"},
		//value:[]uint8{0x4d, 0x8a, 0xf8, 0xe6, 0xa0, 0x0}}}}, "Alias":dbus.Variant{sig:dbus.Signature{str:"s"},
		//value:"A0-E6-F8-8A-4D-5C"}, "AdvertisingFlags":dbus.Variant{sig:dbus.Signature{str:"ay"},
		//value:[]uint8{0x5}}, "Paired":dbus.Variant{sig:dbus.Signature{str:"b"}, value:false},
		//"Connected":dbus.Variant{sig:dbus.Signature{str:"b"}, value:false},
		//"ServicesResolved":dbus.Variant{sig:dbus.Signature{str:"b"}, value:false},
		//"Address":dbus.Variant{sig:dbus.Signature{str:"s"}, value:"A0:E6:F8:8A:4D:5C"}},
		//"org.freedesktop.DBus.Properties":map[string]dbus.Variant{},
		//"org.freedesktop.DBus.Introspectable":map[string]dbus.Variant{}}}}

		err := dbus.Store(s.Body[1:2], &dict)
		if err != nil {
			log.Print(err)
			return nil
		}
		return dict[deviceInterface]
	case interfacesRemoved:
		var dict = make(map[string]dbus.Variant)
		//&dbus.Signal{Sender:":1.3", Path:"/", Name:"org.freedesktop.DBus.ObjectManager.InterfacesRemoved",
		// Body:[]interface {}{"/org/bluez/hci0/dev_A0_E6_F8_8A_4D_5C",
		// []string{"org.freedesktop.DBus.Properties", "org.freedesktop.DBus.Introspectable",
		// "org.bluez.Device1"}}}
		dict[deviceInterface] = dbus.MakeVariant(parseAddressFromPath(string(s.Body[0].(dbus.ObjectPath))))
		return dict

	case propertiesChanged:
		var dict map[string]dbus.Variant
		//&dbus.Signal{Sender:":1.3", Path:"/org/bluez/hci0/dev_A0_E6_F8_8A_4D_5C",
		//Name:"org.freedesktop.DBus.Properties.PropertiesChanged",
		//Body:[]interface {}{"org.bluez.Device1",
		//map[string]dbus.Variant{"RSSI":dbus.Variant{sig:dbus.Signature{str:"n"}, value:-59}}, []string{}}}
		if s.Body[0].(string) == deviceInterface {
			err := dbus.Store(s.Body[1:2], &dict)
			if err != nil {
				log.Printf(err.Error())
				return nil
			}
			return dict
		}
		return nil
	}
	return nil
}

func parseAddressFromPath(path string) string {
	ndx := strings.LastIndex(path, "dev_")
	if ndx >= 0 {
		//Replace the underscores with colons
		return strings.Replace(path[ndx+4:len(path)], "_", ":", -1)
	}
	return ""
}

package ble

import (
	"log"

	"github.com/godbus/dbus"
)

func (conn *Connection) addMatch(rule string) error {
	return conn.bus.BusObject().Call(
		"org.freedesktop.DBus.AddMatch",
		0,
		rule,
	).Err
}

func (conn *Connection) removeMatch(rule string) error {
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
		log.Println("Error retrieving device ble adapter %s", err)
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

	//Define the dbus signals to listen for
	addrule := "type='signal',interface='org.freedesktop.DBus.ObjectManager',member='InterfacesAdded'"
	removerule := "type='signal',interface='org.freedesktop.DBus.ObjectManager',member='InterfacesRemoved'"
	//propertiesrule := "type='signal',interface='org.freedesktop.DBus.Properties',member='PropertiesChanged'"

	//Declare deferreds so that we don't leave anything hanging around
	//defer conn.removeMatch(propertiesrule) //This causes problems
	defer conn.removeMatch(addrule)
	defer conn.removeMatch(removerule)
	defer conn.bus.RemoveSignal(signals)

	defer close(deviceChannel)
	defer close(signals)

	var err error
	if err = conn.addMatch(addrule); err != nil {
		log.Println("Error adding InterfacesAdded match: %s", err.Error())
		return
	}

	if err = conn.addMatch(removerule); err != nil {
		log.Println("Error adding InterfacesRemoved match %s", err.Error())
		return
	}

	// if err = conn.addMatch(propertiesrule); err != nil {
	// 	log.Println("Error adding PropertiesChanged match")
	// 	return
	// }

	log.Println("Setting discovery filter")
	if err = adapter.SetDiscoveryFilter(uuids...); err != nil {
		log.Println("Error setting discovery filter: %s", err.Error())
		return
	}

	log.Println("Starting device discovery")
	if err = adapter.StartDiscovery(); err != nil {
		log.Println("Error starting device discovery: %s", err.Error())
		return
	}

	log.Println("Starting discoverDevicesLoop")
	if err = adapter.discoverLoop(deviceChannel, uuids, signals, stopDiscoveryChannel); err != nil {
		log.Println("Error returned from discoverDevicesLoop: %s", err.Error())
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
					log.Println("Error updating object cache for added devices: %v", err)
					return err
				}

				//Get the interface added properties
				props := interfaceProperties(s)
				if theDevice, err := adapter.Conn().GetDeviceByAddress(props["Address"].Value().(string)); err == nil {
					deviceChannel <- &theDevice
				} else {
					log.Println(err)
				}

			//No need to account for PropertiesChanged signal
			//Don't think we need to account for interfaces removed, but leaving here just in case
			case interfacesRemoved:
				log.Printf("interface removed signal = %#v", s)

				// The interfaces removed signal contains only the device address.
				// The address would need to be parsed from the path.
				//TODO - Determine if this is needed
				//deviceChannel <- &theDevice
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

			//case propertiesChanged:
			//log.Printf("properties changed signal = %#v", s)

			//&dbus.Signal{
			//	Sender:":1.7",
			//	Path:"/org/bluez/hci0/dev_A0_E6_F8_8A_57_C7",
			//	Name:"org.freedesktop.DBus.Properties.PropertiesChanged",
			//	Body:[]interface {}{
			//		"org.bluez.Device1",
			//		map[string]dbus.Variant{
			//			"RSSI":dbus.Variant{sig:dbus.Signature{str:"n"}, value:-97}
			//		},
			//		[]string{}
			//	}
			//}

			//Need to filter only on devices
			//TODO - Need to determine if object cache device needs to be updated
			//TODO - Need to determine if we need to respond to this signal
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

	var dict map[string]map[string]dbus.Variant
	err := dbus.Store(s.Body[1:2], &dict)
	if err != nil {
		log.Print(err)
		return nil
	}
	return dict[deviceInterface]
}

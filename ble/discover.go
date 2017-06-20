package ble

import (
	"fmt"
	"log"
	"time"

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

//TODO Delete if possible
// DiscoveryTimeoutError indicates that discovery has timed out.
type DiscoveryTimeoutError []string

//TODO Delete if possible
func (e DiscoveryTimeoutError) Error() string {
	return fmt.Sprintf("discovery timeout %v", []string(e))
}

//TODO Delete if possible
// Discover puts the adapter in discovery mode,
// waits for the specified timeout to discover one of the given UUIDs,
// and then stops discovery mode.
func (adapter *blob) Discover(timeout time.Duration, uuids ...string) error {
	conn := adapter.conn
	signals := make(chan *dbus.Signal)
	defer close(signals)
	conn.bus.Signal(signals)
	defer conn.bus.RemoveSignal(signals)
	addrule := "type='signal',interface='org.freedesktop.DBus.ObjectManager',member='InterfacesAdded'"
	err := adapter.conn.addMatch(addrule)
	if err != nil {
		return err
	}

	// removerule := "type='signal',interface='org.freedesktop.DBus.ObjectManager',member='InterfacesRemoved'"
	// err = adapter.conn.addMatch(removerule)
	// if err != nil {
	// 	return err
	// }

	defer conn.removeMatch(addrule) // nolint
	//defer conn.removeMatch(removerule) // nolint

	err = adapter.SetDiscoveryFilter(uuids...)
	if err != nil {
		return err
	}
	err = adapter.StartDiscovery()
	if err != nil {
		return err
	}
	defer adapter.StopDiscovery() // nolint
	var t <-chan time.Time
	if timeout != 0 {
		t = time.After(timeout)
	}

	return adapter.discoverLoop(uuids, signals, t)
}

//TODO Delete if possible
// Discover initiates discovery for a LE peripheral with the given UUIDs.
// It waits for at most the specified timeout, or indefinitely if timeout = 0.
//func (conn *Connection) Discover(timeout time.Duration, uuids ...string) (Device, error) {
func (conn *Connection) Discover(timeout time.Duration, uuids ...string) ([]Device, error) {
	adapter, err := conn.GetAdapter()
	if err != nil {
		return nil, err
	}
	err = adapter.Discover(timeout, uuids...)
	if err != nil {
		return nil, err
	}
	err = conn.Update()
	if err != nil {
		return nil, err
	}

	return conn.GetDevices(uuids...)
	//return device, nil
}

//TODO Delete if possible
func (adapter *blob) discoverLoop(uuids []string, signals <-chan *dbus.Signal, timeout <-chan time.Time) error {
	for {
		select {
		case s := <-signals:
			switch s.Name {
			case interfacesAdded:
				if adapter.discoveryComplete(s, uuids) {
					return nil
				}

				props := interfaceProperties(s)
				log.Println("Interface properties = %#v", props)

			//No need to account for PropertiesChanged signal
			//Don't think we need to account for interfaces removed, but leaving here just in case
			// case interfacesRemoved:
			// 	log.Printf("Interface removed")
			// 	log.Printf("interface removed signal = %s", s)
			default:
				log.Printf("%s: unexpected signal %s", adapter.Name(), s.Name)
			}
		case <-timeout:
			return DiscoveryTimeoutError(uuids)
		}
	}
}

//This function is invoked when discovery of an individual device is completed
func (adapter *blob) discoveryComplete(s *dbus.Signal, uuids []string) bool {
	//Properties is of type "properties map[string]dbus.Variant""
	//Each property in props is of the form map[string]dbus.Variant
	props := interfaceProperties(s)

	if props == nil {
		log.Printf("%s: skipping signal with no device interface", adapter.Name())
		return false
	}

	var name string
	if props["Name"].Value() != nil {
		name = props["Name"].Value().(string)
	} else {
		name = props["Alias"].Value().(string)
	}

	services := props["UUIDs"].Value().([]string)
	if uuidsInclude(services, uuids) {
		log.Printf("%s: discovered %s", adapter.Name(), name)
		return true
	}
	log.Printf("%s: skipping signal for device %s", adapter.Name(), name)
	return false
}

//TODO Rename if possible
// InitiateDiscovery - Initiates discovery of LE peripherals with the given UUIDs.
func (conn *Connection) InitiateDiscovery(uuids ...string) (chan *Device, error) {

	deviceDiscoveredChannel := make(chan *Device)

	log.Println("In Initiate Discovery, Retrieving device ble adapter from DBUS")
	adapter, err := conn.GetAdapter()
	if err != nil {
		log.Println("Error retrieving device ble adapter %s", err)
		return nil, err
	}

	go adapter.DiscoverDevices(deviceDiscoveredChannel, uuids...)
	if err != nil {
		log.Println("Error while discovering devices %s", err)
		return nil, err
	}

	return deviceDiscoveredChannel, nil
}

//TODO Rename if possible
// Discover puts the adapter in discovery mode,
// waits for the specified timeout to discover one of the given UUIDs,
// and then stops discovery mode.
func (adapter *blob) DiscoverDevices(deviceChannel chan<- *Device, uuids ...string) error {
	conn := adapter.conn

	signals := make(chan *dbus.Signal)
	conn.bus.Signal(signals)

	addrule := "type='signal',interface='org.freedesktop.DBus.ObjectManager',member='InterfacesAdded'"
	err := adapter.conn.addMatch(addrule)
	if err != nil {
		log.Println("Error adding InterfacesAdded match")
		return err
	}

	removerule := "type='signal',interface='org.freedesktop.DBus.ObjectManager',member='InterfacesRemoved'"
	err = adapter.conn.addMatch(removerule)
	if err != nil {
		log.Println("Error adding InterfacesRemoved match")
		return err
	}

	//Deferreds
	defer close(signals)
	defer conn.bus.RemoveSignal(signals)
	defer close(deviceChannel)
	defer conn.removeMatch(addrule)
	defer conn.removeMatch(removerule)

	log.Println("Setting discovery filter")
	err = adapter.SetDiscoveryFilter(uuids...)
	if err != nil {
		log.Println("Error setting discovery filter: %s", err)
		return err
	}

	log.Println("Starting device discovery")
	err = adapter.StartDiscovery()
	if err != nil {
		log.Println("Error starting device discovery: %s", err)
		return err
	}

	log.Println("Starting discoverDevicesLoop")
	err = adapter.discoverDevicesLoop(deviceChannel, uuids, signals)
	if err != nil {
		log.Println("Error returned from discoverDevicesLoop: %s", err)
		return err
	}
	return err
}

//TODO Rename if possible
func (adapter *blob) discoverDevicesLoop(deviceChannel chan<- *Device, uuids []string, signals <-chan *dbus.Signal) error {
	log.Println("In discoverDevicesLoop")
	defer adapter.StopDiscovery() // nolint

	for {
		select {
		case s := <-signals:
			switch s.Name {
			case interfacesAdded:
				//Refresh the list of managed objects
				err := adapter.conn.Update()
				if err != nil {
					log.Println("Error updating object cache: %v", err)
					return err
				}

				//if adapter.discoveryComplete(s, uuids) {
				props := interfaceProperties(s)
				theDevice, err := adapter.Conn().GetDeviceByAddress(props["Address"].Value().(string))
				if err == nil {
					deviceChannel <- &theDevice
				} else {
					log.Println(err)
				}
				//}

			//No need to account for PropertiesChanged signal
			//Don't think we need to account for interfaces removed, but leaving here just in case
			case interfacesRemoved:
				log.Printf("interface removed signal = %s", s)
			default:
				log.Printf("%s: unexpected signal %s", adapter.Name(), s.Name)
			}
		}
	}
}

// If the InterfacesAdded signal contains deviceInterface,
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

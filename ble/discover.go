package ble

import (
	"log"

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
func (conn *Connection) StartDiscovery(stopDiscoveryChannel <-chan bool, uuids ...string) chan *dbus.Signal {

	//Create the channel that will be used to return DBUS signal events to the caller
	//This channel is closed when the Discover method ends
	deviceDiscoveredChannel := make(chan *dbus.Signal)

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
func (adapter *blob) Discover(deviceChannel chan<- *dbus.Signal, stopDiscoveryChannel <-chan bool, uuids ...string) {

	conn := adapter.conn
	signals := make(chan *dbus.Signal)
	conn.bus.Signal(signals)

	//Declare deferreds so that we don't leave anything hanging around
	defer conn.bus.RemoveSignal(signals)

	defer close(deviceChannel)
	defer close(signals)

	var err error

	if len(uuids) > 0 {
		log.Printf("Setting discovery filter")
		if err = adapter.SetDiscoveryFilter(uuids...); err != nil {
			log.Printf("Error setting discovery filter: %s", err.Error())
			return
		}
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

func (adapter *blob) discoverLoop(deviceChannel chan<- *dbus.Signal, uuids []string, signals <-chan *dbus.Signal, stopDiscoveryChannel <-chan bool) error {
	for {
		select {
		case s := <-signals:
			log.Printf("Signal received: %#v)", s)
			switch s.Name {
			case InterfacesAdded:
				deviceChannel <- s
			case InterfacesRemoved:
				deviceChannel <- s
			case PropertiesChanged:
				deviceChannel <- s
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

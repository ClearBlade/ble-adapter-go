package ble

import (
	"log"

	"github.com/godbus/dbus"
)

// The Adapter type corresponds to the org.bluez.Adapter1 interface.
// See bluez/doc/adapter-api.txt
//
// StartDiscovery starts discovery on the adapter.
//
// StopDiscovery stops discovery on the adapter.
//
// RemoveDevice removes the specified device and its pairing information.
//
// SetDiscoveryFilter sets the discovery filter to require
// LE transport and the given UUIDs.
//
// Discover performs discovery for a device with the given UUIDs,
// for at most the specified timeout, or indefinitely if timeout is 0.
// See also the Discover method of the ObjectCache type.
type Adapter interface {
	BaseObject

	StartDiscovery() error
	StopDiscovery() error
	RemoveDevice(*Device) error
	SetDiscoveryFilter(uuids ...string) error
	Discover(sigChannel chan<- *dbus.Signal, stopDiscoveryChannel <-chan bool, uuids ...string)

	Address() string //The Bluetooth device address - readonly
	Alias() string   //The Bluetooth friendly name - readwrite
	SetAlias(string)
	Class() uint32 //The Bluetooth class of device - readonly
	Powered() bool //Switch an adapter on or off - readwrite
	SetPowered(bool)
	Discoverable() bool //Switch an adapter to discoverable or non-discoverable - readwrite
	SetDiscoverable(bool)
	Pairable() bool //Switch an adapter to pairable or non-pairable - readwrite
	SetPairable(bool)
	PairableTimeout() uint32 //The pairable timeout in seconds - readwrite
	SetPairableTimeout(uint32)
	DiscoverableTimeout() uint32 //The discoverable timeout in seconds - readwrite
	SetDiscoverableTimeout(uint32)
	Discovering() bool //Indicates that a device discovery procedure is active - readonly
	UUIDs() []string   //List of 128-bit UUIDs that represents the available local services - readonly
	Modalias() string  //Local Device ID information in modalias format used by the kernel and udev - readonly, optional
}

// GetAdapter finds an Adapter in the object cache and returns it.
func (conn *Connection) GetAdapter() (Adapter, error) {
	return conn.findObject(AdapterInterface, func(_ *blob) bool { return true })
}

func (adapter *blob) StartDiscovery() error {
	log.Printf("%s: starting discovery", adapter.Name())
	return adapter.call("StartDiscovery")
}

func (adapter *blob) StopDiscovery() error {
	log.Printf("%s: stopping discovery", adapter.Name())
	return adapter.call("StopDiscovery")
}

func (adapter *blob) RemoveDevice(device *Device) error {
	log.Printf("%s: removing device %s", adapter.Name(), (*device).Name())
	return adapter.call("RemoveDevice", (*device).Path())
}

func (adapter *blob) Powered() bool {
	return adapter.properties[BluezPowered].Value().(bool)
}

func (adapter *blob) SetPowered(powered bool) {
	adapter.properties[BluezPowered] = dbus.MakeVariant(powered)
}

func (adapter *blob) Discoverable() bool {
	return adapter.properties[BluezDiscoverable].Value().(bool)
}

func (adapter *blob) SetDiscoverable(discoverable bool) {
	adapter.properties[BluezDiscoverable] = dbus.MakeVariant(discoverable)
}

func (adapter *blob) Pairable() bool {
	return adapter.properties[BluezPairable].Value().(bool)
}

func (adapter *blob) SetPairable(pairable bool) {
	adapter.properties[BluezPairable] = dbus.MakeVariant(pairable)
}

func (adapter *blob) PairableTimeout() uint32 {
	return adapter.properties[BluezPairableTimeout].Value().(uint32)
}

func (adapter *blob) SetPairableTimeout(pairableTimeout uint32) {
	if pairableTimeout > 0 {
		adapter.properties[BluezPairableTimeout] = dbus.MakeVariant(pairableTimeout)
	}
}

func (adapter *blob) DiscoverableTimeout() uint32 {
	return adapter.properties[BluezDiscoverableTimeout].Value().(uint32)
}

func (adapter *blob) SetDiscoverableTimeout(discoverableTimeout uint32) {
	if discoverableTimeout > 0 {
		adapter.properties[BluezDiscoverableTimeout] = dbus.MakeVariant(discoverableTimeout)
	}
}

func (adapter *blob) Discovering() bool {
	return adapter.properties[BluezDiscovering].Value().(bool)
}

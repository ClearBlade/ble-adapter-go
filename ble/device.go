package ble

import (
	"fmt"
	"log"
	"strings"

	"reflect"

	"github.com/godbus/dbus"
)

const (
	deviceInterface   = "org.bluez.Device1"
	interfacesAdded   = "org.freedesktop.DBus.ObjectManager.InterfacesAdded"
	interfacesRemoved = "org.freedesktop.DBus.ObjectManager.InterfacesRemoved"
)

// The Device type corresponds to the org.bluez.Device1 interface.
// See bluez/doc/devicet-api.txt
type Device interface {
	BaseObject

	UUIDs() []string
	Address() string
	Alias() string
	ManufacturerData() map[string]interface{}
	RSSI() int16
	Connected() bool
	Paired() bool

	Connect() error
	Pair() error
}

type JSONableSlice []uint8

func (conn *Connection) matchDevice(matching predicate) (Device, error) {
	return conn.findObject(deviceInterface, matching)
}

func (conn *Connection) matchDevices(matching predicate) ([]Device, error) {
	objects, err := conn.findObjects(deviceInterface, matching)

	devices := make([]Device, len(objects))
	for i := range devices {
		devices[i] = objects[i]
	}

	return devices, err
}

// GetDevice finds a Device in the object cache matching the given UUIDs.
func (conn *Connection) GetDevice(uuids ...string) (Device, error) {
	return conn.matchDevice(func(device *blob) bool {
		return uuidsInclude(device.UUIDs(), uuids)
	})
}

// GetDevices finds all Devices in the object cache matching the given UUIDs.
func (conn *Connection) GetDevices(uuids ...string) ([]Device, error) {
	return conn.matchDevices(func(device *blob) bool {
		return uuidsInclude(device.UUIDs(), uuids)
	})
}

func uuidsInclude(advertised []string, uuids []string) bool {
	for _, u := range uuids {
		if !ValidUUID(strings.ToLower(u)) {
			log.Printf("invalid UUID %s", u)
			return false
		}
		if !stringArrayContains(advertised, u) {
			return false
		}
	}
	return true
}

// GetDeviceByName finds a Device in the object cache with the given name.
func (conn *Connection) GetDeviceByName(name string) (Device, error) {
	return conn.matchDevice(func(device *blob) bool {
		return device.Name() == name
	})
}

// GetDeviceByName finds a Device in the object cache with the given name.
func (conn *Connection) GetDeviceByAddress(address string) (Device, error) {
	return conn.matchDevice(func(device *blob) bool {
		return device.Address() == address
	})
}

func (device *blob) UUIDs() []string {
	return device.properties["UUIDs"].Value().([]string)
}

func (device *blob) ManufacturerData() map[string]interface{} {

	manDataMap := make(map[string]interface{})

	if manufacturer, ok := device.properties["ManufacturerData"]; ok {
		manData := manufacturer.Value().(map[uint16]dbus.Variant)
		for key, value := range manData {
			manDataMap["id"] = key

			if reflect.TypeOf(value.Value()) == reflect.TypeOf([]byte(nil)) {
				//Golang converts byte arrays to strings when marshalling as json
				//We need to use JSONableSlice so that they remain as byte arrays
				manDataMap["data"] = JSONableSlice(value.Value().([]uint8))
			} else {
				manDataMap["data"] = value.Value()
			}
		}
	}

	return manDataMap
}

func (device *blob) Address() string {
	return device.properties["Address"].Value().(string)
}

func (device *blob) Alias() string {
	return device.properties["Alias"].Value().(string)
}

func (device *blob) RSSI() int16 {
	return device.properties["RSSI"].Value().(int16)
}

func (device *blob) Connected() bool {
	return device.properties["Connected"].Value().(bool)
}

func (device *blob) Paired() bool {
	return device.properties["Paired"].Value().(bool)
}

func (device *blob) Connect() error {
	log.Printf("%s: connecting", device.Name())
	return device.call("Connect")
}

func (device *blob) Pair() error {
	log.Printf("%s: pairing", device.Name())
	return device.call("Pair")
}

func stringArrayContains(a []string, str string) bool {
	for _, s := range a {
		if strings.ToLower(s) == strings.ToLower(str) {
			return true
		}
	}
	return false
}

//MarshalJSON - Allows a []byte to be marshalled as an array rather than a string
func (u JSONableSlice) MarshalJSON() ([]byte, error) {
	var result string
	if u == nil {
		result = "null"
	} else {
		result = strings.Join(strings.Fields(fmt.Sprintf("%d", u)), ",")
	}
	return []byte(result), nil
}

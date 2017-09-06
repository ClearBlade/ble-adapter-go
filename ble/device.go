package ble

import (
	"fmt"
	"log"
	"strings"

	"reflect"

	"github.com/godbus/dbus"
)

// The Device type corresponds to the org.bluez.Device1 interface.
// See bluez/doc/devicet-api.txt
type Device interface {
	BaseObject

	Connect() error
	Disconnect() error
	ConnectProfile(string) error
	DisconnectProfile(string) error
	Pair() error
	CancelPairing() error

	Address() string                          //The Bluetooth device address of the remote device - readonly
	Name() string                             //The Bluetooth remote name - readonly, optional
	Icon() string                             //Proposed icon name according to the freedesktop.org icon naming specification - readonly, optional
	Class() uint32                            //The Bluetooth class of device - readonly, optional
	Appearance() uint16                       //External appearance of device, as found on GAP service - readonly, optional
	UUIDs() []string                          //List of 128-bit UUIDs that represents the available remote services - readonly, optional
	Paired() bool                             //Indicates if the remote device is paired - readonly
	Connected() bool                          //Indicates if the remote device is currently connec - readonly
	Trusted() bool                            //Indicates if the remote is seen as trusted - readwrite
	SetTrusted(bool)                          //Sets the trusted value
	Blocked() bool                            //If set to true any incoming connections from the device will be immediately rejected - readwrite
	SetBlocked(bool)                          //Sets the blocked value
	Alias() string                            //The name alias for the remote device - readwrite
	SetAlias(string)                          //Sets the device alias
	Adapter() dbus.ObjectPath                 //The object path of the adapter the device belongs to - readonly
	LegacyPairing() bool                      //Set to true if the device only supports the pre-2.1 pairing mechanism
	Modalias() string                         //Remote Device ID information in modalias format used by the kernel and udev - readonly, optional
	RSSI() int16                              //Received Signal Strength Indicator of the remote device - readonly, optional
	TxPower() int16                           //Advertised transmitted power level - readonly, optional
	ManufacturerData() map[string]interface{} //Manufacturer specific advertisement data - readonly, optional
	ServiceData() map[string]interface{}      //Service advertisement data - readonly, optional
	ServicesResolved() bool                   //Indicate whether or not service discovery has been resolved - readonly
	AdvertisingFlags() []byte                 //The Advertising Data Flags of the remote device - readonly, experimental
}

type JSONableSlice []uint8

func (conn *Connection) matchDevice(matching predicate) (Device, error) {
	return conn.findObject(DeviceInterface, matching)
}

func (conn *Connection) matchDevices(matching predicate) ([]Device, error) {
	objects, err := conn.findObjects(DeviceInterface, matching)

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

// Name returns the object's name.
func (device *blob) Name() string {
	name, ok := device.properties[BluezName].Value().(string)
	if !ok {
		return ""
	}
	return name
}

func (device *blob) Appearance() uint16 {
	appearance, ok := device.properties[BluezAppearance].Value().(uint16)
	if !ok {
		return 0
	}
	return appearance
}

func (device *blob) Icon() string {
	icon, ok := device.properties[BluezIcon].Value().(string)
	if !ok {
		return ""
	}
	return icon
}

func (device *blob) RSSI() int16 {
	rssi, ok := device.properties[BluezRSSI].Value().(int16)
	if !ok {
		return -1
	}
	return rssi
}

func (device *blob) TxPower() int16 {
	txPower, ok := device.properties[BluezTxPower].Value().(int16)
	if !ok {
		return -1
	}
	return txPower
}

func (device *blob) ManufacturerData() map[string]interface{} {

	manDataMap := make(map[string]interface{})

	if manufacturer, ok := device.properties[BluezManufacturerData]; ok {
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

func (device *blob) ServiceData() map[string]interface{} {

	servDataMap := make(map[string]interface{})

	if service, ok := device.properties[BluezServiceData]; ok {
		servData := service.Value().(map[uint16]dbus.Variant)
		for key, value := range servData {
			servDataMap["id"] = key

			if reflect.TypeOf(value.Value()) == reflect.TypeOf([]byte(nil)) {
				//Golang converts byte arrays to strings when marshalling as json
				//We need to use JSONableSlice so that they remain as byte arrays
				servDataMap["data"] = JSONableSlice(value.Value().([]uint8))
			} else {
				servDataMap["data"] = value.Value()
			}
		}
	}

	return servDataMap
}

func (device *blob) AdvertisingFlags() []byte {
	var val = device.properties[BluezAdvertisingFlags].Value()
	if val == nil {
		return []byte{}
	}
	return val.([]byte)
}

func (device *blob) Connected() bool {
	return device.properties[BluezConnected].Value().(bool)
}

func (device *blob) Paired() bool {
	return device.properties[BluezPaired].Value().(bool)
}

func (device *blob) LegacyPairing() bool {
	return device.properties[BluezLegacyPairing].Value().(bool)
}

func (device *blob) Trusted() bool {
	return device.properties[BluezTrusted].Value().(bool)
}

func (device *blob) SetTrusted(trusted bool) {
	device.properties[BluezTrusted] = dbus.MakeVariant(trusted)
}

func (device *blob) ServicesResolved() bool {
	return device.properties[BluezServicesResolved].Value().(bool)
}

func (device *blob) Blocked() bool {
	return device.properties[BluezBlocked].Value().(bool)
}

func (device *blob) SetBlocked(blocked bool) {
	device.properties[BluezBlocked] = dbus.MakeVariant(blocked)
}

func (device *blob) Adapter() dbus.ObjectPath {
	return device.properties[BluezAdapter].Value().(dbus.ObjectPath)
}

func (device *blob) Connect() error {
	log.Printf("%s: connecting", device.Name())
	return device.call("Connect")
}

func (device *blob) Disconnect() error {
	log.Printf("%s: disconnecting", device.Name())
	return device.call("Disconnect")
}

func (device *blob) ConnectProfile(uuid string) error {
	log.Printf("%s: connecting profile %s", device.Name(), uuid)
	return device.call("ConnectProfile", uuid)
}

func (device *blob) DisconnectProfile(uuid string) error {
	log.Printf("%s: disconnecting profile %s", device.Name(), uuid)
	return device.call("DisconnectProfile", uuid)
}

func (device *blob) Pair() error {
	log.Printf("%s: pairing", device.Name())
	return device.call("Pair")
}

func (device *blob) CancelPairing() error {
	log.Printf("%s: cancelling pairing", device.Name())
	return device.call("CancelPairing")
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

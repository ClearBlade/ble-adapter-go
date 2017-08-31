package ble

import (
	"log"

	"github.com/godbus/dbus"
)

func (conn *Connection) findGattObject(iface string, uuid string) (*blob, error) {
	return conn.findObject(iface, func(desc *blob) bool {
		return desc.UUID() == uuid
	})
}

func (conn *Connection) findGattObjectByPath(iface string, path string) (*blob, error) {
	return conn.findObject(iface, func(desc *blob) bool {
		return string(desc.Path()) == path
	})
}

// GattHandle is the interface satisfied by GATT handles.
type GattHandle interface {
	BaseObject

	UUID() string
}

// UUID returns the handle's UUID
func (handle *blob) UUID() string {
	return handle.properties[BluezUUID].Value().(string)
}

// Service corresponds to the org.bluez.GattService1 interface.
// See bluez/doc/gatt-api.txt
type Service interface {
	GattHandle

	Primary() bool               //Indicates whether or not this GATT service is a primary service - readonly
	Device() dbus.ObjectPath     //Object path of the Bluetooth device the service belongs to - readonly, optional
	Includes() []dbus.ObjectPath //Array of object paths representing the included services of this service - readonly, Currently not implemented in BlueZ
}

func (service *blob) Primary() bool {
	return service.properties[BluezPrimary].Value().(bool)
}

func (service *blob) Device() dbus.ObjectPath {
	device, ok := service.properties[BluezDevice].Value().(dbus.ObjectPath)
	if !ok {
		return *(new(dbus.ObjectPath))
	}
	return device
}

func (service *blob) Includes() []dbus.ObjectPath {
	includes, ok := service.properties[BluezIncludes].Value().([]dbus.ObjectPath)
	if !ok {
		return []dbus.ObjectPath{}
	}
	return includes
}

// GetService finds a Service with the given UUID.
func (conn *Connection) GetService(uuid string) (Service, error) {
	return conn.findGattObject(ServiceInterface, uuid)
}

// ReadWriteHandle is the interface satisfied by GATT objects
// that provide ReadValue and WriteValue operations.
type ReadWriteHandle interface {
	GattHandle

	ReadValue() ([]byte, error)
	WriteValue([]byte) error
}

// ReadValue reads the handle's value.
func (handle *blob) ReadValue() ([]byte, error) {
	var data []byte
	err := handle.callv("ReadValue", Properties{}).Store(&data)
	return data, err
}

// WriteValue writes a value to the handle.
func (handle *blob) WriteValue(data []byte) error {
	log.Printf("In WriteValue")
	return handle.call("WriteValue", data, Properties{})
}

// NotifyHandler represents a function that handles notifications.
type NotifyHandler func([]byte)

// Characteristic corresponds to the org.bluez.GattCharacteristic1 interface.
// See bluez/doc/gatt-api.txt
type Characteristic interface {
	ReadWriteHandle

	StartNotify() error
	StopNotify() error
	HandleNotify(NotifyHandler) error

	Service() dbus.ObjectPath //Object path of the GATT service the characteristic belongs to - readonly
	Value() []byte            //The cached value of the characteristic - readonly, optional
	WriteAcquired() bool      //True, if this characteristic has been acquired by any client using AcquireWrite - readonly, optional
	NotifyAcquired() bool     //True, if this characteristic has been acquired by any client using AcquireNotify - readonly, optional
	Notifying() bool          //True, if notifications or indications on this characteristic are currently enabled - readonly, optional
	Flags() []string          //Defines how the characteristic value can be used - readonly
}

// GetCharacteristic finds a Characteristic with the given UUID.
func (conn *Connection) GetCharacteristic(uuid string) (Characteristic, error) {
	return conn.findGattObject(CharacteristicInterface, uuid)
}

// ReadCharacteristic reads a Characteristic with the given UUID.
func (conn *Connection) ReadCharacteristic(uuid string) ([]byte, error) {

	var char Characteristic
	var err error
	if char, err = conn.GetCharacteristic(uuid); err != nil {
		return nil, err
	}

	return char.ReadValue()
}

// WriteCharacteristic writes a Characteristic with the given UUID.
func (conn *Connection) WriteCharacteristic(uuid string, value []byte) error {

	log.Printf("In WriteCharacteristic")
	var char Characteristic
	var err error
	if char, err = conn.GetCharacteristic(uuid); err != nil {
		return err
	}

	log.Printf("Characteristic retrieved")

	//TODO see if flags allow value to be written

	return char.WriteValue(value)
}

// Service returns the object path of the GATT service the characteristic belongs tog.
func (handle *blob) Service() dbus.ObjectPath {
	return handle.properties[BluezService].Value().(dbus.ObjectPath)
}

// Value returns the cached value of the characteristic.
func (handle *blob) Value() []byte {
	value, ok := handle.properties[BluezValue].Value().([]byte)
	if !ok {
		return []byte{}
	}
	return value
}

func (handle *blob) WriteAcquired() bool {
	write, ok := handle.properties[BluezWriteAcquired].Value().(bool)
	if !ok {
		return false
	}
	return write
}

func (handle *blob) NotifyAcquired() bool {
	notify, ok := handle.properties[BluezNotifyAcquired].Value().(bool)
	if !ok {
		return false
	}
	return notify
}

// Notifying returns whether or not a Characteristic is notifying.
func (handle *blob) Notifying() bool {
	notifying, ok := handle.properties[BluezNotifying].Value().(bool)
	if !ok {
		return false
	}
	return notifying
}

func (handle *blob) Flags() []string {
	flags, ok := handle.properties[BluezFlags].Value().([]string)
	if !ok {
		return []string{}
	}
	return flags
}

// StartNotify starts notifying.
func (handle *blob) StartNotify() error {
	return handle.call("StartNotify")
}

// StartNotify stops notifying.
func (handle *blob) StopNotify() error {
	return handle.call("StopNotify")
}

// Descriptor corresponds to the org.bluez.GattDescriptor1 interface.
// See bluez/doc/gatt-api.txt
type Descriptor interface {
	ReadWriteHandle

	Characteristic() dbus.ObjectPath //Object path of the GATT characteristic the descriptor belongs to - readonly
	Value() []byte                   //The cached value of the descriptor - readonly, optional
	Flags() []string                 //Defines how the descriptor value can be used - readonly
}

// Characteristic returns the object path of the GATT characteristic the descriptor belongs tog.
func (handle *blob) Characteristic() dbus.ObjectPath {
	return handle.properties[BluezCharacteristic].Value().(dbus.ObjectPath)
}

// GetDescriptor finds a Descriptor with the given UUID.
func (conn *Connection) GetDescriptor(uuid string) (Descriptor, error) {
	return conn.findGattObject(DescriptorInterface, uuid)
}

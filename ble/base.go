/*
Package ble provides functions to discover, connect, pair,
and communicate with Bluetooth Low Energy peripheral devices.

This implementation uses the BlueZ D-Bus interface, rather than sockets.
It is similar to github.com/adafruit/Adafruit_Python_BluefruitLE
*/
package ble

import (
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/godbus/dbus"
)

// Connection represents a D-Bus connection.
type Connection struct {
	bus *dbus.Conn

	// It would be nice to factor out the subtypes here,
	// but then the reflection used by dbus.Store() wouldn't work.
	objects map[dbus.ObjectPath]map[string]map[string]dbus.Variant
}

// Open opens a connection to the system D-Bus
func Open() (*Connection, error) {
	bus, err := dbus.SystemBus()
	if err != nil {
		return nil, err
	}
	conn := Connection{bus: bus}
	err = conn.Update()
	if err != nil {
		conn.Close()
		return nil, err
	}
	return &conn, nil
}

// Close closes the D-Bus connection.
func (conn *Connection) Close() {
	conn.bus.Close() // nolint
}

// Update gets all objects and properties.
// See http://dbus.freedesktop.org/doc/dbus-specification.html#standard-interfaces-objectmanager
func (conn *Connection) Update() error {
	call := conn.bus.Object("org.bluez", "/").Call(
		dot(ObjectManager, "GetManagedObjects"),
		0,
	)
	return call.Store(&conn.objects)
}

type dbusInterfaces *map[string]map[string]dbus.Variant

// The iterObjects function applies a function of type objectProc to
// each object in the cache.  It should return true if the iteration
// should stop, false if it should continue.
type objectProc func(dbus.ObjectPath, dbusInterfaces) bool

func (conn *Connection) iterObjects(proc objectProc) {
	for path, dict := range conn.objects {
		if proc(path, &dict) {
			return
		}
	}
}

// Print prints the objects in the cache.
func (conn *Connection) Print(w *io.Writer) {
	printer := func(path dbus.ObjectPath, dict dbusInterfaces) bool {
		return printObject(w, path, dict)
	}
	conn.iterObjects(printer)
}

// nolint: errcheck, gas
func printObject(w *io.Writer, path dbus.ObjectPath, dict dbusInterfaces) bool {
	fmt.Fprintln(*w, path)
	for iface, props := range *dict {
		printProperties(w, iface, props)
	}
	fmt.Fprintln(*w)
	return false
}

// BaseObject is the interface satisfied by bluez D-Bus objects.
type BaseObject interface {
	Conn() *Connection
	Path() dbus.ObjectPath
	Interface() string
	Print(*io.Writer)
}

type Properties map[string]dbus.Variant

type blob struct {
	conn       *Connection
	path       dbus.ObjectPath
	iface      string
	properties Properties
	object     dbus.BusObject
}

// Conn returns the object's D-Bus connection.
func (obj *blob) Conn() *Connection {
	return obj.conn
}

// Path returns the object's D-Bus path.
func (obj *blob) Path() dbus.ObjectPath {
	return obj.path
}

// Interface returns the object's D-Bus interface name.
func (obj *blob) Interface() string {
	return obj.iface
}

func (obj *blob) Class() uint32 {
	class, ok := obj.properties["Class"].Value().(uint32)
	if !ok {
		return 0
	}
	return class
}

// UUIDs returns the object's UUIDs.
func (obj *blob) UUIDs() []string {
	uuids, ok := obj.properties["UUIDs"].Value().([]string)
	if !ok {
		return []string{}
	}
	return uuids
}

// Address returns the object's Address.
func (obj *blob) Address() string {
	return obj.properties["Address"].Value().(string)
}

// Alias returns the object's Alias.
func (obj *blob) Alias() string {
	return obj.properties["Alias"].Value().(string)
}

func (obj *blob) SetAlias(alias string) {
	if alias != "" {
		obj.properties["Alias"] = dbus.MakeVariant(alias)
	}
}

func (obj *blob) Modalias() string {
	modalias, ok := obj.properties["Modalias"].Value().(string)
	if !ok {
		return ""
	}
	return modalias
}

func (obj *blob) callv(method string, args ...interface{}) *dbus.Call {
	const callTimeout = 5 * time.Second
	c := obj.object.Go(dot(obj.iface, method), 0, nil, args...)
	if c.Err == nil {
		select {
		case <-c.Done:
		case <-time.After(callTimeout):
			c.Err = fmt.Errorf("BLE call timeout")
		}
	}
	return c
}

func (obj *blob) call(method string, args ...interface{}) error {
	return obj.callv(method, args...).Err
}

// Print prints the object.
func (obj *blob) Print(w *io.Writer) {
	fmt.Fprintf(*w, "%s [%s]\n", obj.path, obj.iface) // nolint
	printProperties(w, "", obj.properties)
}

// nolint: errcheck, gas
func printProperties(w *io.Writer, iface string, props Properties) {
	indent := "    "
	if iface != "" {
		fmt.Fprintf(*w, "%s%s\n", indent, iface)
		indent += indent
	}
	for key, val := range props {
		fmt.Fprintf(*w, "%s%s %s\n", indent, key, val.String())
	}
}

// The findObject function tests each object with functions of type predicate.
type predicate func(*blob) bool

// findObject finds an object satisfying the given predicate.
// If returns an error if zero or more than one is found.
func (conn *Connection) findObject(iface string, matching predicate) (*blob, error) {
	var found []*blob
	conn.iterObjects(func(path dbus.ObjectPath, dict dbusInterfaces) bool {
		_ = fmt.Errorf("Interfaces = %#v", dict)
		props := (*dict)[iface]
		if props == nil {
			return false
		}
		obj := &blob{
			conn:       conn,
			path:       path,
			iface:      iface,
			properties: props,
			object:     conn.bus.Object("org.bluez", path),
		}
		if matching(obj) {
			found = append(found, obj)
		}
		return false
	})
	switch len(found) {
	case 1:
		return found[0], nil
	case 0:
		return nil, fmt.Errorf("interface %s not found", iface)
	default:
		return nil, fmt.Errorf("found %d instances of interface %s", len(found), iface)
	}
}

// findObjects finds objects satisfying the given predicate.
// If returns an error if zero or more than one is found.
func (conn *Connection) findObjects(iface string, matching predicate) ([]*blob, error) {
	var found []*blob
	conn.iterObjects(func(path dbus.ObjectPath, dict dbusInterfaces) bool {
		props := (*dict)[iface]
		if props == nil {
			return false
		}
		obj := &blob{
			conn:       conn,
			path:       path,
			iface:      iface,
			properties: props,
			object:     conn.bus.Object("org.bluez", path),
		}
		if matching(obj) {
			found = append(found, obj)
		}
		return false
	})
	switch len(found) {
	case 1:
		return found, nil
	case 0:
		return nil, fmt.Errorf("interface %s not found", iface)
	default:
		return found, nil
	}
}

func dot(a, b string) string {
	return a + "." + b
}

//CreateNewBlob - Creates a new blob instance
func CreateNewBlob() *blob {
	return new(blob)
}

//GetInterfaceProperties
// If the signal contains deviceInterface,
// return the corresponding properties, otherwise nil.
// See http://dbus.freedesktop.org/doc/dbus-specification.html#standard-interfaces-objectmanager
func GetInterfaceProperties(s *dbus.Signal) Properties {
	switch s.Name {
	case InterfacesAdded:
		log.Printf("[DEBUG] Returning InterfacesAdded properties")
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

		if val, ok := dict[AdapterInterface]; ok {
			log.Printf("[DEBUG] Returning Adapter properties")
			return val
		}
		if val, ok := dict[DeviceInterface]; ok {
			log.Printf("[DEBUG] Returning Device properties")
			return val
		}
		if val, ok := dict[ServiceInterface]; ok {
			log.Printf("[DEBUG] Returning GATT Service properties")
			return val
		}
		if val, ok := dict[CharacteristicInterface]; ok {
			log.Printf("[DEBUG] Returning GATT Characteristic properties")
			return val
		}
		if val, ok := dict[DescriptorInterface]; ok {
			log.Printf("[DEBUG] Returning GATT Descriptor properties")
			return val
		}

		//Otherwise, return an empty properties struct
		return Properties(make(map[string]dbus.Variant))
	case InterfacesRemoved:
		log.Printf("[DEBUG] Returning InterfacesRemoved properties")
		var dict = make(map[string]dbus.Variant)
		//&dbus.Signal{Sender:":1.3", Path:"/", Name:"org.freedesktop.DBus.ObjectManager.InterfacesRemoved",
		// Body:[]interface {}{"/org/bluez/hci0/dev_A0_E6_F8_8A_4D_5C",
		// []string{"org.freedesktop.DBus.Properties", "org.freedesktop.DBus.Introspectable",
		// "org.bluez.Device1"}}}
		dict[DeviceInterface] = dbus.MakeVariant(ParseAddressFromPath(string(s.Body[0].(dbus.ObjectPath))))
		return dict

	case PropertiesChanged:
		log.Printf("[DEBUG] Returning PropertiesChanged properties")
		var dict map[string]dbus.Variant
		//&dbus.Signal{Sender:":1.3", Path:"/org/bluez/hci0/dev_A0_E6_F8_8A_4D_5C",
		//Name:"org.freedesktop.DBus.Properties.PropertiesChanged",
		//Body:[]interface {}{"org.bluez.Device1",
		//map[string]dbus.Variant{"RSSI":dbus.Variant{sig:dbus.Signature{str:"n"}, value:-59}}, []string{}}}
		if s.Body[0].(string) == DeviceInterface {
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

//ParseAddressFromPath - Extracts the device address from a device path
func ParseAddressFromPath(path string) string {
	ndx := strings.LastIndex(path, "dev_")
	if ndx >= 0 {
		//Replace the underscores with colons
		return strings.Replace(path[ndx+4:len(path)], "_", ":", -1)
	}
	return ""
}

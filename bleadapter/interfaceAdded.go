package bleadapter

import (
	"log"

	cbble "github.com/clearblade/ble-adapter-go/ble"
	"github.com/godbus/dbus"
)

//Helper methods related to DBUS InterfaceAdded signals

//HandleInterfaceAdded - Determine the interface the InterfaceAdded signal occured on and handle
// the signal accordingly
func HandleInterfaceAdded(adapt BleAdapter, signal *dbus.Signal) {
	//&dbus.Signal{
	//	Sender:":1.3",
	//	Path:"/",
	//	Name:"org.freedesktop.DBus.ObjectManager.InterfacesAdded",
	//	Body:[]interface {}{
	//		"/org/bluez/hci0/dev_A0_E6_F8_8A_4D_5C",
	//			map[string]map[string]dbus.Variant{"org.bluez.Device1":map[string]dbus.Variant{
	//				"Trusted":dbus.Variant{sig:dbus.Signature{str:"b"}, value:false},
	//				"Blocked":dbus.Variant{sig:dbus.Signature{str:"b"}, value:false},
	//				"LegacyPairing":dbus.Variant{sig:dbus.Signature{str:"b"}, value:false},
	//				"RSSI":dbus.Variant{sig:dbus.Signature{str:"n"}, value:-59},
	//				"UUIDs":dbus.Variant{sig:dbus.Signature{str:"as"},
	//					value:[]string{"32f9169f-4feb-4883-ade6-1f0127018db3"}},
	//				"Adapter":dbus.Variant{sig:dbus.Signature{str:"o"}, value:"/org/bluez/hci0"},
	//				"ManufacturerData":dbus.Variant{sig:dbus.Signature{str:"a{qv}"},
	//					value:map[uint16]dbus.Variant{0x5c60:dbus.Variant{sig:dbus.Signature{str:"ay"},
	//					value:[]uint8{0x4d, 0x8a, 0xf8, 0xe6, 0xa0, 0x0}}}},
	//				"Alias":dbus.Variant{sig:dbus.Signature{str:"s"}, value:"A0-E6-F8-8A-4D-5C"},
	//				"AdvertisingFlags":dbus.Variant{sig:dbus.Signature{str:"ay"}, value:[]uint8{0x5}},
	//				"Paired":dbus.Variant{sig:dbus.Signature{str:"b"}, value:false},
	//				"Connected":dbus.Variant{sig:dbus.Signature{str:"b"}, value:false},
	//				"ServicesResolved":dbus.Variant{sig:dbus.Signature{str:"b"}, value:false},
	//				"Address":dbus.Variant{sig:dbus.Signature{str:"s"}, value:"A0:E6:F8:8A:4D:5C"}
	//			},
	//			"org.freedesktop.DBus.Properties":map[string]dbus.Variant{},
	//			"org.freedesktop.DBus.Introspectable":map[string]dbus.Variant{}
	//		}
	//	}
	//}
	log.Printf("[DEBUG] In HandleInterfaceAdded")
	//The body of the interface added signal will always be a 2 element array with the following elements:
	// elem 1 - The path of the DBUS interface that was added
	// elem 2 - A map of the DBUS Interfaces that were added

	//Get the interface properties
	props := cbble.GetInterfaceProperties(signal)

	//Perform interface specific processing on the signal
	if (signal.Body[1].(map[string]map[string]dbus.Variant))[cbble.AdapterInterface] != nil {
		HandleAdapterAdded(adapt, props)
	} else if (signal.Body[1].(map[string]map[string]dbus.Variant))[cbble.DeviceInterface] != nil {
		HandleDeviceAdded(adapt, props)
	} else if (signal.Body[1].(map[string]map[string]dbus.Variant))[cbble.ServiceInterface] != nil {
		HandleGattServiceAdded(adapt, props)
	} else if (signal.Body[1].(map[string]map[string]dbus.Variant))[cbble.CharacteristicInterface] != nil {
		HandleGattCharacteristicAdded(adapt, props)
	} else if (signal.Body[1].(map[string]map[string]dbus.Variant))[cbble.DescriptorInterface] != nil {
		HandleGattDescriptorAdded(adapt, props)
	} else {
		log.Printf("[DEBUG] Unhandled signal interface %#v", signal.Body[1])
	}
}

//HandleAdapterAdded - Future development
func HandleAdapterAdded(adapt BleAdapter, properties cbble.Properties) {
	log.Printf("[DEBUG] Adapter interface added")
	//log.Printf("properties = %#v", properties)

	//No current need to process adapters
}

//HandleDeviceAdded - Publish new devices to the platform
func HandleDeviceAdded(adapt BleAdapter, properties cbble.Properties) {
	log.Printf("[DEBUG] Device interface added")
	//log.Printf("properties = %#v", properties)

	//Publish the device to the platform
	log.Printf("[DEBUG] Publishing device to platform")
	adapt.publishDevice(properties["Address"].Value().(string))
}

//HandleGattServiceAdded - Future development
func HandleGattServiceAdded(adapt BleAdapter, properties cbble.Properties) {
	log.Printf("[DEBUG] GATT Service interface added")
	//log.Printf("properties = %#v", properties)

	//No current need to process gatt services
}

//HandleGattCharacteristicAdded - Future development
func HandleGattCharacteristicAdded(adapt BleAdapter, properties cbble.Properties) {
	log.Printf("[DEBUG] GAT Characteristic interface added")
	//log.Printf("properties = %#v", properties)

	//No current need to process gatt characteristics
}

//HandleGattDescriptorAdded - Future development
func HandleGattDescriptorAdded(adapt BleAdapter, properties cbble.Properties) {
	log.Printf("[DEBUG] GATT Descriptor interface added")
	//log.Printf("properties = %#v", properties)

	//No current need to process gatt descriptors
}

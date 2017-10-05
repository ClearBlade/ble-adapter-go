package bleadapter

import (
	"log"

	cbble "github.com/clearblade/ble-adapter-go/ble"
	"github.com/godbus/dbus"
)

//Helper methods related to DBUS PropertiesChanged signals

//HandlePropertyChanged - Determine the interface the PropertyChanged signal occured on and handle
// the signal accordingly
func HandlePropertyChanged(adapt BleAdapter, signal *dbus.Signal) {
	//&dbus.Signal{
	//	Sender:":1.3",
	//	Path:"/org/bluez/hci0/dev_A0_E6_F8_8A_4D_5C",
	//	Name:"org.freedesktop.DBus.Properties.PropertiesChanged",
	//	Body:[]interface {}{
	//		"org.bluez.Device1",
	//		map[string]dbus.Variant{
	//			"RSSI":dbus.Variant{sig:dbus.Signature{str:"n"}, value:-59}
	//		},
	//		[]string{}
	//	}
	//}

	//The body of the properties changed signal will always be a 3 element array with the following elements:
	// elem 1 - The DBUS interface that generated the properties changed signal
	// elem 2 - A map containing the changed properties
	// elem 3 - An array containing the names of invalidated properties

	log.Printf("[DEBUG] In HandlePropertyChanged")

	//Perform interface specific processing on the signal
	switch signal.Body[0].(string) {
	case cbble.AdapterInterface:
		HandleAdapterPropertyChanged(adapt, signal)
	case cbble.DeviceInterface:
		HandleDevicePropertyChanged(adapt, signal)
	case cbble.ServiceInterface:
		HandleGattServicePropertyChanged(adapt, signal)
	case cbble.CharacteristicInterface:
		HandleGattCharacteristicPropertyChanged(adapt, signal)
	case cbble.DescriptorInterface:
		HandleGattDescriptorPropertyChanged(adapt, signal)
	default:
		log.Printf("[DEBUG] Unhandled signal interface %s", signal.Body[0].(string))
	}

}

//HandleAdapterPropertyChanged - Future development
func HandleAdapterPropertyChanged(adapt BleAdapter, signal *dbus.Signal) {
	//Adapter properties
	// Powered
	// Discoverable
	// Pairable
	// PairableTimeout
	// DiscoverableTimeout
	// Discovering

	log.Printf("[DEBUG] In HandleAdapterPropertyChanged")
}

//HandleDevicePropertyChanged - Future development
func HandleDevicePropertyChanged(adapt BleAdapter, signal *dbus.Signal) {
	//Device properties
	// Paired
	// Connected
	// Trusted
	// Blocked
	// RSSI
	// TxPower
	// ServicesResolved

	//May need to add property specific processing later

	//Publish the device to the platform
	//&dbus.Signal{
	//	Sender:":1.3",
	//	Path:"/org/bluez/hci0/dev_A0_E6_F8_8A_4D_5C",
	//	Name:"org.freedesktop.DBus.Properties.PropertiesChanged",
	//	Body:[]interface {}{
	//		"org.bluez.Device1",
	//		map[string]dbus.Variant{
	//			"RSSI":dbus.Variant{sig:dbus.Signature{str:"n"}, value:-59}
	//		},
	//		[]string{}
	//	}
	//}

	//Get the mac address from the device path
	log.Printf("[DEBUG] HandleDevicePropertyChanged - Publishing device to platform: %s", string(signal.Path))
	adapt.publishDevice(cbble.ParseAddressFromPath(string(signal.Path)))
}

//HandleGattServicePropertyChanged - Future development
func HandleGattServicePropertyChanged(adapt BleAdapter, signal *dbus.Signal) {
	log.Printf("[DEBUG] In HandleGattServicePropertyChanged. No implementation yet.")
}

//HandleGattCharacteristicPropertyChanged - Future development
func HandleGattCharacteristicPropertyChanged(adapt BleAdapter, signal *dbus.Signal) {
	log.Printf("[DEBUG] In HandleGattCharacteristicPropertyChanged. No implementation yet.")
}

//HandleGattDescriptorPropertyChanged - Future development
func HandleGattDescriptorPropertyChanged(adapt BleAdapter, signal *dbus.Signal) {
	log.Printf("[DEBUG] In HandleGattDescriptorPropertyChanged. No implementation yet.")
}

//HandleDevicePairedChange - Future development
func HandleDevicePairedChange(adapt BleAdapter, signal *dbus.Signal) {
	log.Printf("[DEBUG] In HandleDevicePairedChange. No implementation yet.")
}

//HandleDeviceConnectedChange - Future development
func HandleDeviceConnectedChange(adapt BleAdapter, signal *dbus.Signal) {
	//&dbus.Signal{
	//	Sender:":1.7",
	//	Path:"/org/bluez/hci0/dev_00_0B_57_36_73_9F",
	//	Name:"org.freedesktop.DBus.Properties.PropertiesChanged",
	//	Body:[]interface {}{
	//		"org.bluez.Device1",
	//		map[string]dbus.Variant{
	//			"Connected":dbus.Variant{sig:dbus.Signature{str:"b"}, value:true}
	//		},
	//		[]string{}
	//	}
	//}
	log.Printf("[DEBUG] In HandleDeviceConnectedChange. No implementation yet.")
}

//HandleDeviceTrustedChange - Future development
func HandleDeviceTrustedChange(adapt BleAdapter, signal *dbus.Signal) {
	log.Printf("[DEBUG] In HandleDeviceTrustedChange. No implementation yet.")
}

//HandleDeviceBlockedChange - Future development
func HandleDeviceBlockedChange(adapt BleAdapter, signal *dbus.Signal) {
	log.Printf("[DEBUG] In HandleDeviceBlockedChange. No implementation yet.")
}

//HandleDeviceRssiChange - Future development
func HandleDeviceRssiChange(adapt BleAdapter, signal *dbus.Signal) {
	log.Printf("[DEBUG] In HandleDeviceRssiChange. No implementation yet.")
}

//HandleDeviceTxPowerChange - Future development
func HandleDeviceTxPowerChange(adapt BleAdapter, signal *dbus.Signal) {
	log.Printf("[DEBUG] In HandleDeviceTxPowerChange. No implementation yet.")
}

//HandleDeviceServicesResolvedChange - Future development
func HandleDeviceServicesResolvedChange(adapt BleAdapter, signal *dbus.Signal) {
	//&dbus.Signal{
	//	Sender:":1.7",
	//	Path:"/org/bluez/hci0/dev_00_0B_57_36_73_9F",
	//	Name:"org.freedesktop.DBus.Properties.PropertiesChanged",
	//	Body:[]interface {}{
	//		"org.bluez.Device1",
	//		map[string]dbus.Variant{"ServicesResolved":dbus.Variant{sig:dbus.Signature{str:"b"}, value:true}}, []string{}}})
	log.Printf("[DEBUG] In HandleDeviceServicesResolvedChange. No implementation yet.")
}

//TODO - Determine if gatt property changes result in properties changed signals
//Gatt service properties
// Primary
// Device
// Includes

//Gatt characteristic properties
// Service
// Value
// WriteAcquired
// NotifyAcquired
// Notifying
// Flags

//Gatt descriptor properties
// Characteristic
// Value
// Flags

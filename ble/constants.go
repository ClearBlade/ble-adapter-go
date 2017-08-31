package ble

//Constants used to reference DBUS specific items
const (
	BluetoothBaseUUID = "00000000-0000-1000-8000-00805F9B34FB"
	//DBUS Interfaces
	ObjectManager           = "org.freedesktop.DBus.ObjectManager"
	AdapterInterface        = "org.bluez.Adapter1"
	DeviceInterface         = "org.bluez.Device1"
	ServiceInterface        = "org.bluez.GattService1"
	CharacteristicInterface = "org.bluez.GattCharacteristic1"
	DescriptorInterface     = "org.bluez.GattDescriptor1"
	DbusProperties          = "org.freedesktop.DBus.Properties"
	DbusIntrospectable      = "org.freedesktop.DBus.Introspectable"

	//DBUS signals
	InterfacesAdded   = "org.freedesktop.DBus.ObjectManager.InterfacesAdded"
	InterfacesRemoved = "org.freedesktop.DBus.ObjectManager.InterfacesRemoved"
	PropertiesChanged = "org.freedesktop.DBus.Properties.PropertiesChanged"

	//DBus signal rules
	AddRule        = "type='signal',interface='org.freedesktop.DBus.ObjectManager',member='InterfacesAdded'"
	RemoveRule     = "type='signal',interface='org.freedesktop.DBus.ObjectManager',member='InterfacesRemoved'"
	PropertiesRule = "type='signal',interface='org.freedesktop.DBus.Properties',member='PropertiesChanged'"

	//BlueZ DBUS Properties
	BluezAdapter             = "Adapter"
	BluezAddress             = "Address"
	BluezAdvertisingFlags    = "AdvertisingFlags"
	BluezAlias               = "Alias"
	BluezAppearance          = "Appearance"
	BluezBlocked             = "Blocked"
	BluezCharacteristic      = "Characteristic"
	BluezClass               = "Class"
	BluezConnected           = "Connected"
	BluezDevice              = "Device"
	BluezDiscoverable        = "Discoverable"
	BluezDiscoverableTimeout = "DiscoverableTimeout"
	BluezDiscovering         = "Discovering"
	BluezFlags               = "Flags"
	BluezIcon                = "Icon"
	BluezIncludes            = "Includes"
	BluezLegacyPairing       = "LegacyPairing"
	BluezManufacturerData    = "ManufacturerData"
	BluezModalias            = "Modalias"
	BluezName                = "Name"
	BluezNotifyAcquired      = "NotifyAcquired"
	BluezNotifying           = "Notifying"
	BluezPairable            = "Pairable"
	BluezPairableTimeout     = "PairableTimeout"
	BluezPaired              = "Paired"
	BluezPowered             = "Powered"
	BluezPrimary             = "Primary"
	BluezRSSI                = "RSSI"
	BluezService             = "Service"
	BluezServiceData         = "ServiceData"
	BluezServicesResolved    = "ServicesResolved"
	BluezTrusted             = "Trusted"
	BluezTxPower             = "TxPower"
	BluezUUID                = "UUID"
	BluezUUIDs               = "UUIDs"
	BluezValue               = "Value"
	BluezWriteAcquired       = "WriteAcquired"

	BleCommandRead  = "read"
	BleCommandWrite = "write"
)

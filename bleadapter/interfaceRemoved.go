package bleadapter

import (
	"log"

	cbble "github.com/clearblade/ble-adapter-go/ble"
	"github.com/godbus/dbus"
)

//Helper methods related to DBUS InterfaceRemoved signals

//HandleInterfaceRemoved - Future development
//
//Determine the interface the InterfaceRemoved signal occured on and handle the signal accordingly
func HandleInterfaceRemoved(adapt BleAdapter, signal *dbus.Signal) {
	//&dbus.Signal{
	//	Sender:":1.7",
	//	Path:"/",
	//	Name:"org.freedesktop.DBus.ObjectManager.InterfacesRemoved",
	//	Body:[]interface {}{
	//		"/org/bluez/hci0/dev_A0_E6_F8_8B_A8_6F",
	//		[]string{
	//			"org.freedesktop.DBus.Properties",
	//			"org.freedesktop.DBus.Introspectable",
	//			"org.bluez.Device1"
	//		}
	//	}
	//}
	//Get the interface removed properties
	log.Printf("[DEBUG] In HandleInterfaceRemoved")

	props := cbble.GetInterfaceProperties(signal)
	if props[cbble.DeviceInterface].Value() != nil {
		//Determine what to do when devices are removed
		log.Printf("[DEBUG] Device removed = %#v", props[cbble.DeviceInterface].Value())
	}
}

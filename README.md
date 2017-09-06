# BLE-ADAPTER-GO

A GoLang bluetooth adapter implementation utilizing BlueZ and DBUS that allows BLE devices to interact with the ClearBlade Platform.

Much of the code has been based off of the implementation found at https://github.com/ecc1/ble

Additional information regarding BlueZ and DBUS can be found at the following links:
BlueZ - https://git.kernel.org/cgit/bluetooth/bluez.git/tree/doc  
DBUS - https://www.freedesktop.org/wiki/Software/dbus/

## Operating System Dependencies
The BLE Adapter is only supported on Linux operating systems with the following specifications:

  * Linux - Minimum kernel version of 3.5 - http://www.bluez.org/release-of-bluez-5-0/  
  * BlueZ - The minimum version of BlueZ with BLE support is 5.40. It is recommended to install at least BlueZ version 5.44 (5.46 is the most current version).

## Cross compile GoLang for Raspberry Pi
`GOOS=linux GOARCH=arm GOARM=6 go build`

## Status
---

The adapter currently supports the following:
  1. Discover BLE devices and forward device properties to the ClearBlade platform
  2. Pairing with BLE devices
  3. Cancelling pairing with BLE devices
  4. Removing (unpairing) BLE devices
  5. Connecting to BLE devices
  6. Disconnecting from BLE devices
  7. Reading characteristic values from BLE devices
  8. Writing characteristic values to BLE devices

## ClearBlade Platform Dependencies
The BLE Adapter was constructed to allow BLE devices to communicate with a _System_ defined in a ClearBlade Platform instance. Therefore, the BLE Adapter requires a _System_ to have been created within a ClearBlade Platform instance.

Once a System has been created, artifacts must be defined within the ClearBlade Platform system to allow the BLE Adapter to function properly. At a minimum, the following artifacts __MUST__ be created within a ClearBlade Platform System:

* Auth --> Devices
  * A row representing the physical gateway/device the BLE Adapter will be executing on
  * The value specified in the _name_ column will be used as the value of the __deviceName__ argument specified in the BLE Adapter start-up command
  * The value specified in the _active\_key_ column will be used as the value of the __password__ argument specified in the BLE Adapter start-up command

* BLE\_Adapter\_Config
  * A data collection containing a single row
  * Provides the ability to specify runtime configuration options


* BLE\_Device\_Filters
  * A data collection that provides the ability to dynamically pass BLE _service advertisement_ uuids into the device discovery process, via Adapter.setDiscoveryFilter
  * Discovery filters provide a mechanism to target specific BLE devices
  * __CAVEAT:__ Discovery filtering will only work if the BLE device specifies a service UUID in its advertisement payload.

###BLE\_Adapter\_Config Schema
Column Name | Column Data Type | Column Description
----------- | ---------------- | ------------------
publish\_topic | string | The MQTT topic to use when publishing to the platform (the specified topic will be prepended with _deviceName_/, where deviceName is the value of the __deviceName__ argument specified on the start-up command)
discovery\_scan\_seconds | integer | Specifies the length of time a BLE device discovery scan should run
discovery\_pause\_seconds | integer | Specifies the length of time to pause between BLE device discovery scans
handle\_removed | boolean | Specifies whether or not the BLE adapter should handle DBUS _InterfaceRemoved_ signals
handle\_changed | boolean | Specifies whether or not the BLE adapter should handle DBUS _PropertiesChanged_ signals

###BLE\_Device\_Filters Schema
Column Name | Column Data Type | Column Description
----------- | ---------------- | ------------------
ble\_uuid | string | The _service advertisement_ uuid, optionally broadcasted by a BLE device, to allow for limiting the BLE devices that are discovered by the BLE adapter
enabled | boolean | Specifies whether or not filtering should be enabled for the specified UUID

## Usage

### Starting the ble adapter

`ble-adapter-go -systemKey <PLATFORM SYSTEM KEY> -systemSecret <PLATFORM SYSTEM SECRET> -deviceName <AUTH DEVICE NAME> -password <AUTH DEVICE PASSWORD> -platformURL <CB PLATFORM URL> -messagingURL <CB PLATFORM MESSAGING URL>`

   *Where* 

   __systemKey__
  * REQUIRED
  * The system key of the ClearBLade Platform __System__ the BLE adapter connect to

   __systemSecret__
  * REQUIRED
  * The system secret of the ClearBLade Platform __System__ the BLE adapter connect to
   
   __deviceName__
  * REQUIRED
  * The device name the BLE adapter will use to authenticate to the ClearBlade Platform
  * Requires the device to have been defined in the _Auth - Devices_ collection within the ClearBlade Platform __System__
   
   __password__
  * REQUIRED
  * The password the BLE adapter will use to authenticate to the platform with.
  * Requires the device to have been defined in the _Auth - Devices_ collection within the ClearBlade Platform __System__
  * In most cases the __active_key__ of the device will be used as the password
   
   __platformURL__
  * The url of the ClearBlade Platform instance the BLE Adapter will connect to
  * OPTIONAL
  * Defaults to __http://localhost:9000__
   
   __messagingURL__
  * The MQTT url of the ClearBlade Platform instance the BLE Adapter will connect to
  * OPTIONAL
  * Defaults to __localhost:1883__

### Configuration
The BLE adapter can be configured by changing the values specified within the row contained in the BLE_Adapter_Config data collection within the ClearBlade Platform. Changes made to any values will be applied prior to the start of a subsequent _discovery_ scan.

## Interacting with BLE Devices
The BLE Adapter provides the ability to interact with BLE devices: connect, disconnect, pair, read data, write data, etc. In order to interact with a ble device, a JSON message containing command details must be published, via MQTT, to the ClearBlade Platform MQTT message broker (or a ClearBlade Edge message broker).

### JSON Message Format
The JSON message format expected by the BLE Adapter is as follows:

```json
{
	"command": "write",
	"deviceAddress": "00:0B:57:36:73:9F",
	"devicePath": "/org/bluez/hci0/dev_00_0B_57_36_73_9F",
	"gattCharacteristic": "FCB89C40-C603-59F3-7DC3-5ECE444A401B",
	"gattCharacteristicValue": [12, 23, 43, 45],
	"stayConnected": true
}
```

  command
   * The command to execute on the device. The command should be one of:
      * pair
      * remove
      * connect
      * disconnect
      * read
      * write
      * cancelPairing

  deviceAddress
   * The device MAC address
   * Contained within the _mac\_address_ column of the __BLE\_Devices__ data collection within the ClearBlade Platform  

  devicePath
   * The DBUS object path
   * Can be determined by utilizing a utility such as __D Feet__ 
   * Contained within the JSON stored in the _device\_json_ column of the __BLE\_Devices__ data collection within the ClearBlade Platform  

  gattCharacteristic
   * GATT Characteristic UUID

  gattCharacteristicValue
   * The value to of the specified _gattCharacteristic_
   * Must be specified as an array of 8-bit integers (translates to a byte array in lower level programming languages)
     * [12, 248, 255]
   * Returned in the response payload for __read__ commands
   * Required as input for __write__ commands

  stayConnected
   * Should the BLE adapter in the linux operating system remain connected to the BLE device after the command runs?
   * __true__|__false__
   * The default value, if not specified, is __false__

### Sending BLE Command Requests
To send a command request to a BLE device, JSON data in the format specified above should be published to the ClearBlade Platform or a ClearBlade Edge message broker. The MQTT topic name to publish to MUST be _**{Device Name}/bleadapter/bledevice/command**_, where _{Device Name}_ is the value of the __deviceName__ argument specified in the BLE Adapter start-up command.

#### BLE Command Responses
Responses to BLE commands will be published back to the ClearBlade Platform or ClearBlade Edge, thus allowing code triggers and code services to act on those responses. The responses will be published as a JSON payload. The JSON payload representing the response is a copy of the request payload with additional _err_ and _response_ members added. The _err_ member is a boolean indicator denoting whether an error occurred. The _response_ member is a string providing a description of the outcome of the command.

```json
{
	"command": "write",
	"deviceAddress": "00:0B:57:36:73:9F",
	"devicePath": "/org/bluez/hci0/dev_00_0B_57_36_73_9F",
	"gattCharacteristic": "FCB89C40-C603-59F3-7DC3-5ECE444A401B",
	"gattCharacteristicValue": [12, 23, 43, 45],
	"stayConnected": true,
  "err": false,
  "response": "BLE command write executed successfully"
}
```

## Setup
---
Tested with

- golang `1.8` (minimum `v1.6`)
- bluez bluetooth `v5.44` and `v5.45`
- raspbian and hypirot (debian 8) armv7 `4.4.x`  

### Upgrading BlueZ
These steps were compiled together from multiple sources obtained through numerous internet searches. The main source of information was taken from:

   https://www.element14.com/community/community/stem-academy/microbit/blog/2016/09/16/1-microbit-1-raspberry-pi-3-1-bluez-upgrade-1-huge-headache

#### Install BlueZ Dependencies
  * `sudo apt-get update`
  * `sudo apt-get install –y bluetooth bluez-tools build-essential autoconf glib2.0 libdbus-1-dev libudev-dev libical-dev libreadline-dev`

##### Potential Errors Encountered
```
   Error with "sudo apt-get install glib2.0"  
   Processing triggers for man-db (2.7.0.2-5) ...  
   /usr/bin/mandb: can't rename /var/cache/man/2111 to /var/cache/man/index.db: Read-only file system  
   /usr/bin/mandb: can't remove /var/cache/man/2111: Read-only file system  
   /usr/bin/mandb: can't chmod /var/cache/man/index.db: Read-only file system  
   /usr/bin/mandb: can't remove /var/cache/man/index.db: Read-only file system  
   /usr/bin/mandb: warning: can't update index cache /var/cache/man/index.db: Read-only file system  
   fopen: Read-only file system  
   dpkg: unrecoverable fatal error, aborting:  
    unable to flush updated status of 'man-db': Read-only file system  
   E: Sub-process /usr/bin/dpkg returned an error code (2)  
   E: Failed to write temporary StateFile /var/lib/apt/extended_states.tmp
```  

###### Resolution
Issue the following commands:
  * `sudo mandb`
```
   Purging old database entries in /usr/share/man...  
   Processing manual pages under /usr/share/man...  
   fopen: Read-only file system
```

  * `sudo apt-get install libglib2.0-dev`  
  **In the event of a `__E: dpkg was interrupted__` error, execute:
     `sudo dpkg –configure –a` to correct the problem**

#### Install BlueZ
Install the appropriate version of BlueZ (5.45 is currently the most recent version) by issuing the following commands at a terminal prompt:
  * `wget http://www.kernel.org/pub/linux/bluetooth/bluez-5.45.tar.xz`
  * `tar xf bluez-5.45.tar.xz`
  * `cd bluez-5.45`
  * `./configure --prefix=/usr --mandir=/usr/share/man --sysconfdir=/etc --localstatedir=/var --enable-experimental --enable-maintainer-mode`
  * `make`
  * `sudo make install`


#### Post Installation Configuration
Additional steps are needed in order to ensure BlueZ operates correctly when working with BLE devices and DBUS.
##### Enable Experimental Support
Enable experimental support in the bluetooth daemon (this enables BLE)
  * `sudo sed -i '/^ExecStart.*bluetoothd\s*$/ s/$/ --experimental/' /lib/systemd/system/bluetooth.service`
##### Link to Bluetooth Firmware
Raspbian installs Bluetooth firmware in a directory not recognized by Bluez. A symbolic link needs to be created so that BlueZ can accurately find the Bluetooth firmware.
  * `sudo ln -s /lib/firmware /etc/firmware`
##### Enable Bluetooth on Boot
Enable Bluetooth to load during system boot, then reload daemons to load it without rebooting, or just reboot after enabling it.
  * `sudo systemctl enable bluetooth`
  * `sudo systemctl daemon-reload`
##### Apply permissions
The following command will need to be executed for each user that will be accessing Bluetooth:  
  `sudo usermod -G bluetooth -a pi`
  * Where “pi” is the username in the example above. Replace “pi” with the username of any user who will need to be able to access Bluetooth.

When executing code (specifically GoLang), the following error was encountered when attempting to make a connection to DBUS:  
```
Rejected send message, 2 matched rules; type="method_call", sender=":1.6"  
(uid=1000 pid=1031 comm="./ble-adapter-go -activeKey 1234567890 -deviceName")  
interface="org.freedesktop.DBus.ObjectManager" member="GetManagedObjects" error name="(unset)"  
requested_reply="0" destination="org.bluez"  
(uid=0 pid=624 comm="/usr/libexec/bluetooth/bluetoothd --experimental ")  
```

In order to rectify the issue, the following steps must be performed:
1.	edit `/etc/dbus-1/system.d/bluetooth.conf`
2.	Take the section `<policy user="root">` and duplicate it for the user that the code is executing under (pi is assumed). Otherwise, all BLE code will need to be executed using sudo  
```
   <policy user="pi">  
     <allow own="org.bluez"/>  
     <allow send_destination="org.bluez"/>  
     <allow send_interface="org.bluez.Agent1"/>  
     <allow send_interface="org.bluez.MediaEndpoint1"/>  
     <allow send_interface="org.bluez.MediaPlayer1"/>  
     <allow send_interface="org.bluez.ThermometerWatcher1"/>  
     <allow send_interface="org.bluez.AlertAgent1"/>  
     <allow send_interface="org.bluez.Profile1"/>  
     <allow send_interface="org.bluez.HeartRateWatcher1"/>  
     <allow send_interface="org.bluez.CyclingSpeedWatcher1"/>  
     <allow send_interface="org.bluez.GattCharacteristic1"/>  
     <allow send_interface="org.bluez.GattDescriptor1"/>  
     <allow send_interface="org.freedesktop.DBus.ObjectManager"/>  
     <allow send_interface="org.freedesktop.DBus.Properties"/>  
   </policy>
```
##### Reboot
Reboot after making these changes.
`sudo reboot`

##### Other Errors Encountered
```
2017/06/13 21:48:49 pi: setting discovery filter [32F9169F-4FEB-4883-ADE6-1F0127018DB3]  
Error setting discovery filter: %s Method "SetDiscoveryFilter"   
with signature "a{sv}" on interface "org.bluez.Adapter1" doesn't exist
```

###### Possible Solution
Upgrade BlueZ to version 5.43 at the minimum


## Todo
---
 - Add unit tests
 - Add and generate docs with examples

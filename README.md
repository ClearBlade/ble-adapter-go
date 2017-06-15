# BLE-ADAPTER-GO

A Golang bluetooth adapter implementation utilizing BlueZ and DBUS that allows BLE devices to interact with the ClearBlade Platform.

BlueZ - https://git.kernel.org/cgit/bluetooth/bluez.git/tree/doc
DBUS - https://www.freedesktop.org/wiki/Software/dbus/

## Status
---

The adapter currently only supports the ability to discover BLE devices and subsequently forward BLE device details to the ClearBlade platform. Additional work will need to be done to provide the ability to read from and write to devices.

## Usage
---

`ble-adapter-go -systemKey <PLATFORM SYSTEM KEY> -systemSecret <PLATFORM SYSTEM SECRET> -deviceName <AUTH DEVICE NAME> -password <AUTH DEVICE PASSWORD> -platformURL <CB PLATFORM URL> -messagingURL <CB PLATFORM MESSAGING URL>`

   *Where* 

   __systemKey__
  * REQUIRED
  * The system key of the ClearBLade Platform __System__ the adapter will be connecting to

   __systemSecret__
  * REQUIRED
  * The system secret of the ClearBLade Platform __System__ the adapter will be connecting to
   
   __deviceName__
  * REQUIRED
  * The name of the device the adapter will authenticate to the platform with.
  * Requires the device to have been defined within the ClearBlade Platform __System__
   
   __password__
  * REQUIRED
  * The password the adapter will use to authenticate to the platform with.
  * Requires the device to have been defined within the ClearBlade Platform __System__
  * In most cases the __active_key__ of the device will be used as the password
   
   __platformURL__
  * OPTIONAL
  * Defaults to __http://localhost:9000__
   
   __messagingURL__
  * OPTIONAL
  * Defaults to __localhost:1883__

## Dependencies
  * BlueZ - The minimum version of BlueZ with BLE support is 5.33.


## Setup
---
Tested with

- golang `1.8` (minimum `v1.6`)
- bluez bluetooth `v5.44` and `v5.45`
- raspbian and hypirot (debian 8) armv7 `4.4.x`  

See in `scripts/` how to upgrade bluez to 5.43

Give access to `hciconfig` to any user (may have [security implications](https://www.insecure.ws/linux/getcap_setcap.html))

```
sudo setcap 'cap_net_raw,cap_net_admin+eip' `which hciconfig`
```

### Upgrading BlueZ
These steps were compiled together from multiple sources obtained through numerous internet searches. The main source of information was taken from:

   https://www.element14.com/community/community/stem-academy/microbit/blog/2016/09/16/1-microbit-1-raspberry-pi-3-1-bluez-upgrade-1-huge-headache

#### Install BlueZ Dependencies
  * `sudo apt-get update`
  * `sudo apt-get install –y bluetooth bluez-tools build-essential autoconf glib2.0 libdbus-1-dev libudev-dev libical-dev libreadline-dev`

##### Potential Errors Encountered
   `Error with "sudo apt-get install glib2.0"`
   `Processing triggers for man-db (2.7.0.2-5) ...`
   `/usr/bin/mandb: can't rename /var/cache/man/2111 to /var/cache/man/index.db: Read-only file system`
   `/usr/bin/mandb: can't remove /var/cache/man/2111: Read-only file system`
   `/usr/bin/mandb: can't chmod /var/cache/man/index.db: Read-only file system`
   `/usr/bin/mandb: can't remove /var/cache/man/index.db: Read-only file system`
   `/usr/bin/mandb: warning: can't update index cache /var/cache/man/index.db: Read-only file system`
   `fopen: Read-only file system`
   `dpkg: unrecoverable fatal error, aborting:`
   ` unable to flush updated status of 'man-db': Read-only file system`
   `E: Sub-process /usr/bin/dpkg returned an error code (2)`
   `E: Failed to write temporary StateFile /var/lib/apt/extended_states.tmp`

###### Resolution
Issue the following commands:
  * `sudo mandb`
   `Purging old database entries in /usr/share/man...`
   `Processing manual pages under /usr/share/man...`
   `fopen: Read-only file system`

  * `sudo apt-get install libglib2.0-dev`
   **In the event of a __E: dpkg was interrupted__ error, execute `sudo dpkg –configure –a` to correct the problem

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
Rejected send message, 2 matched rules; type="method_call", sender=":1.6" (uid=1000 pid=1031 comm="./ble-adapter-go -activeKey 1234567890 -deviceName") interface="org.freedesktop.DBus.ObjectManager" member="GetManagedObjects" error name="(unset)" requested_reply="0" destination="org.bluez" (uid=0 pid=624 comm="/usr/libexec/bluetooth/bluetoothd --experimental ")


In order to rectify the issue, the following steps must be performed:
1.	edit `/etc/dbus-1/system.d/bluetooth.conf`
2.	Take the section `<policy user="root">` and duplicate it for the user that the code is executing under (pi is assumed). Otherwise, all BLE code will need to be executed using sudo 
   `<policy user="pi">`
   `    <allow own="org.bluez"/>`
   `    <allow send_destination="org.bluez"/>`
   `    <allow send_interface="org.bluez.Agent1"/>`
   `    <allow send_interface="org.bluez.MediaEndpoint1"/>`
   `    <allow send_interface="org.bluez.MediaPlayer1"/>`
   `    <allow send_interface="org.bluez.ThermometerWatcher1"/>`
   `    <allow send_interface="org.bluez.AlertAgent1"/>`
   `    <allow send_interface="org.bluez.Profile1"/>`
   `    <allow send_interface="org.bluez.HeartRateWatcher1"/>`
   `    <allow send_interface="org.bluez.CyclingSpeedWatcher1"/>`
   `    <allow send_interface="org.bluez.GattCharacteristic1"/>`
   `    <allow send_interface="org.bluez.GattDescriptor1"/>`
   `    <allow send_interface="org.freedesktop.DBus.ObjectManager"/>`
   `    <allow send_interface="org.freedesktop.DBus.Properties"/>`
   `  </policy>`
##### Reboot
Reboot after making these changes.
`sudo reboot`

##### Other Errors Encountered
`2017/06/13 21:48:49 pi: setting discovery filter [32F9169F-4FEB-4883-ADE6-1F0127018DB3]`
`Error setting discovery filter: %s Method "SetDiscoveryFilter" with signature "a{sv}" on interface "org.bluez.Adapter1" doesn't exist`

###### Possible Solution
Upgrade BlueZ to version 5.43 at the minimum


## Todo
---
 - Add Device read / write
 - Add unit tests
 - Add and generate docs with examples

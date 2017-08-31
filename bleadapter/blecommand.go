package bleadapter

import (
	"encoding/json"
	"errors"
	"log"
	"strings"

	cbble "github.com/clearblade/ble-adapter-go/ble"
)

//Available BLE commands
//
//  Pair
//  Remove (unpair)
// 	Connect
//  Disconnect
// 	Read
// 	Write
//  CancelPairing

type commandProcessor interface {
	Process(*BLECommand) error
	Name() string
}

//Pair - A struct used to encapsulate a BLE device "pair" subcommand
type Pair struct{}

//CancelPairing - A struct used to encapsulate a BLE device "cancel pairing" subcommand
type CancelPairing struct{}

//Remove - A struct used to encapsulate a BLE device "remove" subcommand
type Remove struct{}

//Connect - A struct used to encapsulate a BLE device "connect" subcommand
type Connect struct{}

//Disconnect - A struct used to encapsulate a BLE device "disconnect" subcommand
type Disconnect struct{}

//Read - A struct used to encapsulate a BLE device "read" subcommand
type Read struct{}

//Write - A struct used to encapsulate a BLE device "write" subcommand
type Write struct{}

//BLECommand - A struct used to encapsulate a BLE command received from the platform
type BLECommand struct {
	adapter     *BleAdapter            //Provides access to the DBUS connection and CbClient
	command     map[string]interface{} //The command that was received, will have command, device address, device path,
	subCommands []commandProcessor
	device      *cbble.Device
}

var (
	pair          = Pair{}
	cancelPairing = CancelPairing{}
	remove        = Remove{}
	connect       = Connect{}
	disconnect    = Disconnect{}
	read          = Read{}
	write         = Write{}
)

func NewBLECommand(theBleAdapter *BleAdapter, jsoncommand map[string]interface{}) *BLECommand {

	bleCommand := &BLECommand{
		adapter:     theBleAdapter,
		command:     jsoncommand,
		subCommands: []commandProcessor{},
	}

	//Build the array of sub-commands that are needed to handle the entire ble command
	switch strings.ToLower(jsoncommand["command"].(string)) {
	case "pair":
		bleCommand.subCommands = append(bleCommand.subCommands, pair)
	case "remove":
		bleCommand.subCommands = append(bleCommand.subCommands, remove)
	case "connect":
		bleCommand.subCommands = append(bleCommand.subCommands, connect)
	case "disconnect":
		bleCommand.subCommands = append(bleCommand.subCommands, disconnect)
	case "read":
		bleCommand.subCommands = append(bleCommand.subCommands, connect, read)
	case "write":
		bleCommand.subCommands = append(bleCommand.subCommands, connect, write)
	case "cancelpairing":
		bleCommand.subCommands = append(bleCommand.subCommands, cancelPairing)
	}

	log.Printf("[DEBUG] bleCommand.subCommands: %#v", bleCommand.subCommands)

	if (jsoncommand["stayConnected"] == nil || jsoncommand["stayConnected"] != true) &&
		(strings.ToLower(jsoncommand["command"].(string)) != "disconnect" && strings.ToLower(jsoncommand["command"].(string)) != "remove") {
		log.Printf("[DEBUG] Adding disconnect command")
		bleCommand.subCommands = append(bleCommand.subCommands, disconnect)
	}

	return bleCommand
}

//Execute - Retrieve the BLE device from the object cache and execute the subcommands
//against the device
func (cmd BLECommand) Execute() error {

	dev, err := getDevice(&cmd)
	if err != nil {
		log.Printf("[ERROR] Unable to execute BLE command \"" + cmd.command["command"].(string) + "\". Error received when retrieving BLE device from DBUS object cache: " + err.Error())
		return errors.New("Unable to execute BLE command \"" + cmd.command["command"].(string) + "\". Error received when retrieving BLE device from DBUS object cache: " + err.Error())
	}

	cmd.device = &dev

	for _, subcmd := range cmd.subCommands {
		log.Printf("[DEBUG] Executing subcommand %s", subcmd.Name())
		err = subcmd.Process(&cmd)
		if err != nil {
			log.Printf("[ERROR] Error executing subcommand %s", subcmd.Name())
			break
		}
	}

	return err
}

//Name - Return the name of the subcommand
func (cmd Pair) Name() string {
	return "Pair"
}

//Process - Execute the subcommand
func (cmd Pair) Process(blecmd *BLECommand) error {
	if err := (*blecmd.device).Pair(); err != nil {
		log.Printf("[ERROR] Error while pairing: %s", err.Error())
		return errors.New(cmd.Name() + ":Process - Unable to pair with BLE device. Error received when attempting to pair with BLE device: " + err.Error())
	}
	log.Printf("[DEBUG] Subcommand %s complete", cmd.Name())
	return nil
}

//Name - Return the name of the subcommand
func (cmd CancelPairing) Name() string {
	return "CancelPairing"
}

//Process - Execute the subcommand
func (cmd CancelPairing) Process(blecmd *BLECommand) error {
	if err := (*blecmd.device).CancelPairing(); err != nil {
		log.Printf("[ERROR] Error while attempting to cancel pairing: %s", err.Error())
		return errors.New(cmd.Name() + ":Process - Unable to cancel pairing with BLE device. Error received when attempting to cancel pairing with BLE device: " + err.Error())
	}
	log.Printf("[DEBUG] Subcommand %s complete", cmd.Name())
	return nil
}

//Name - Return the name of the subcommand
func (cmd Remove) Name() string {
	return "Remove"
}

//Process - Execute the subcommand
func (cmd Remove) Process(blecmd *BLECommand) error {
	adapter, err := blecmd.adapter.connection.GetAdapter()
	if err != nil {
		log.Printf("[ERROR] Error while retrieving adapter: %s", err.Error())
		return errors.New(cmd.Name() + ":Process - Unable to remove BLE device. Error received when retrieving Bluetooth adapter: " + err.Error())
	}

	if err = adapter.RemoveDevice(blecmd.device); err != nil {
		log.Printf("[ERROR] Error while removing BLE device: %s", err.Error())
		return errors.New(cmd.Name() + ":Process - Unable to remove BLE device. Error received when attempting to remove the BLE device: " + err.Error())
	}
	log.Printf("[DEBUG] Subcommand %s complete", cmd.Name())
	return nil
}

//Name - Return the name of the subcommand
func (cmd Connect) Name() string {
	return "Connect"
}

//Process - Execute the subcommand
func (cmd Connect) Process(blecmd *BLECommand) error {
	if err := (*blecmd.device).Connect(); err != nil {
		log.Printf("[ERROR] Error while connecting to BLE device: %s", err.Error())
		return errors.New(cmd.Name() + ":Process - Unable to connect to BLE device. Error received when attempting to connect to the BLE device: " + err.Error())
	}
	log.Printf("[DEBUG] Subcommand %s complete", cmd.Name())
	return nil
}

//Name - Return the name of the subcommand
func (cmd Disconnect) Name() string {
	return "Disconnect"
}

//Process - Execute the subcommand
func (cmd Disconnect) Process(blecmd *BLECommand) error {
	if err := (*blecmd.device).Disconnect(); err != nil {
		log.Printf("[ERROR] Error while disconnecting from BLE device: %s", err.Error())
		return errors.New(cmd.Name() + ":Process - Unable to disconnect from BLE device. Error received when attempting to disconnect from the BLE device: " + err.Error())
	}
	log.Printf("[DEBUG] Subcommand %s complete", cmd.Name())
	return nil
}

//Name - Return the name of the subcommand
func (cmd Read) Name() string {
	return "Read"
}

//Process - Execute the subcommand
func (cmd Read) Process(blecmd *BLECommand) error {

	gattChar := strings.ToLower(blecmd.command["gattCharacteristic"].(string))
	if gattChar == "" {
		log.Printf("[ERROR] Unable to read BLE data. GATT characteristic UUID not provided.")
		return errors.New(cmd.Name() + ":Process - Unable to read BLE data. GATT characteristic UUID not provided.")
	}

	val, err := blecmd.adapter.connection.ReadCharacteristic(gattChar)
	if err != nil {
		log.Printf("[ERROR] Error while reading from BLE device: %s", err.Error())
		return errors.New(cmd.Name() + ":Process - Unable to read data from BLE device. Error received when attempting to read from the BLE device: " + err.Error())
	}
	if val != nil {
		log.Printf("[DEBUG] Value read from BLE device: %#v", val)
		blecmd.command["gattCharacteristicValue"] = val
	}
	log.Printf("[DEBUG] Subcommand %s complete", cmd.Name())
	return nil
}

//Name - Return the name of the subcommand
func (cmd Write) Name() string {
	return "Write"
}

//Process - Execute the subcommand
func (cmd Write) Process(blecmd *BLECommand) error {
	var gattValue = blecmd.command["gattCharacteristicValue"]
	var gattChar = strings.ToLower(blecmd.command["gattCharacteristic"].(string))

	if gattChar == "" {
		log.Printf("[ERROR] Unable to write BLE data to BLE device. GATT characteristic UUID not provided.")
		return errors.New(cmd.Name() + ":Process - Unable to write BLE data to BLE device. GATT characteristic UUID not provided.")
	}

	if gattValue == nil {
		log.Printf("[ERROR] Unable to write BLE data to BLE device. Gatt characteristic value not provided.")
		return errors.New(cmd.Name() + ":Process - Unable to write BLE data to BLE device. Gatt characteristic value not provided.")
	}

	//The array of bytes passed in json will be passed as []interface{float, float, ...}
	//We need to convert the json value to a byte array
	gattValueBytes := make([]byte, len(gattValue.([]interface{})))
	for i, elem := range gattValue.([]interface{}) {
		gattValueBytes[i] = byte(elem.(float64))
	}
	log.Printf("[DEBUG] gattValueBytes = %#v", gattValueBytes)

	if err := blecmd.adapter.connection.WriteCharacteristic(strings.ToLower(gattChar), gattValueBytes); err != nil {
		log.Printf("[ERROR] Error while writing: %s", err.Error())
		return errors.New(cmd.Name() + ":Process - Unable to write BLE data to BLE device. Error received when attempting to write to the BLE device: " + err.Error())
	}
	log.Printf("[DEBUG] Subcommand %s complete", cmd.Name())
	return nil
}

func getDevice(blecmd *BLECommand) (cbble.Device, error) {
	log.Printf("[DEBUG] Retrieving BLE Device from DBUS object cache. Device address = %s", blecmd.command["deviceAddress"].(string))
	return blecmd.adapter.connection.GetDeviceByAddress(blecmd.command["deviceAddress"].(string))
}

func getCharacteristic(blecmd *BLECommand) (cbble.Characteristic, error) {
	log.Printf("[DEBUG] Retrieving GATT characteristic from DBUS object cache. Characteristic uuid = %s", blecmd.command["gattCharacteristic"].(string))
	return blecmd.adapter.connection.GetCharacteristic(blecmd.command["gattCharacteristic"].(string))
}

func (cmd BLECommand) sendSuccess(msg string) {
	log.Printf("[DEBUG] Sending success response to platform")
	cmd.command["err"] = false
	cmd.command["response"] = msg

	cmd.endCommand()
}

func (cmd BLECommand) sendError(msg string) {
	log.Printf("[DEBUG] Sending error response to platform")
	cmd.command["err"] = true
	cmd.command["response"] = msg

	cmd.endCommand()
}

func (cmd BLECommand) endCommand() {
	//Publish a message to the platform
	sendCommandResponse(&cmd)
}

func sendCommandResponse(blecommand *BLECommand) {
	log.Printf("[DEBUG] Sending response to platform")
	//Publish the command response back to the platform
	resp, err := json.Marshal(blecommand.command)
	if err == nil {
		log.Printf("[DEBUG] Publishing response to platform")
		blecommand.adapter.cbDeviceClient.Publish(blecommand.adapter.cbDeviceClient.DeviceName+"/"+deviceSubscribeTopic+"/response", resp, messagingQos)
	} else {
		log.Printf("[ERROR] Error marshalling response to platform: %s", err.Error())
	}
}

package bleadapter

import (
	"crypto/tls"
	"log"
	"os"
	"strconv"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

var ()

const (
	MQTT_SERVER  = "tcp://127.0.0.1:1883"
	MSG_QOS      = 0
	MSG_RETAINED = false
)

func init() {
}

func clientID() string {
	hostname, _ := os.Hostname()
	return hostname + strconv.Itoa(time.Now().Second())
}

func authenticate() {
	//Authenticate to the MQTT broker
}

func Publish(topic string, message string) {

	//username := flag.String("username", "", "A username to authenticate to the MQTT server")
	//password := flag.String("password", "", "Password to match username")

	connOpts := MQTT.NewClientOptions().AddBroker(MQTT_SERVER).SetClientID(clientID()).SetCleanSession(true)
	// if *username != "" {
	// 	connOpts.SetUsername(*username)
	// 	if *password != "" {
	// 		connOpts.SetPassword(*password)
	// 	}
	// }
	tlsConfig := &tls.Config{InsecureSkipVerify: true, ClientAuth: tls.NoClientCert}
	connOpts.SetTLSConfig(tlsConfig)

	client := MQTT.NewClient(connOpts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
		return
	}
	log.Println("Connected to %s\n", MQTT_SERVER)

	log.Println("Publishing message %s to topic %s\n", message, topic)
	client.Publish(topic, byte(MSG_QOS), MSG_RETAINED, message)
}

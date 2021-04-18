package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	devices "github.com/kubeedge/kubeedge/cloud/pkg/apis/devices/v1alpha2"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var client mqtt.Client
var topic1_payload string
var topic2_payload string

const (
	mqttUrl      = "tcp://192.168.1.28:1883"
	user         = "zozo"
	passwd       = "1994Zozo"
	topic1       = "sensors/livingroom1"
	topic2       = "sensors/livingroom2"
	topic_device = "$hw/events/device/hudtemp1/twin/update"
)

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("Connected")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Connect lost: %v", err)
}

type SensorData struct {
	Temperature string `json:"temperature"`
	Humidity    string `json:"humidity"`
}

//BaseMessage the base struct of event message
type BaseMessage struct {
	EventID   string `json:"event_id"`
	Timestamp int64  `json:"timestamp"`
}

//TwinValue the struct of twin value
type TwinValue struct {
	Value    *string        `json:"value, omitempty"`
	Metadata *ValueMetadata `json:"metadata,omitempty"`
}

//ValueMetadata the meta of value
type ValueMetadata struct {
	Timestamp int64 `json:"timestamp, omitempty"`
}

//TypeMetadata the meta of value type
type TypeMetadata struct {
	Type string `json:"type,omitempty"`
}

//TwinVersion twin version
type TwinVersion struct {
	CloudVersion int64 `json:"cloud"`
	EdgeVersion  int64 `json:"edge"`
}

//MsgTwin the struct of device twin
type MsgTwin struct {
	Expected        *TwinValue    `json:"expected,omitempty"`
	Actual          *TwinValue    `json:"actual,omitempty"`
	Optional        *bool         `json:"optional,omitempty"`
	Metadata        *TypeMetadata `json:"metadata,omitempty"`
	ExpectedVersion *TwinVersion  `json:"expected_version,omitempty"`
	ActualVersion   *TwinVersion  `json:"actual_version,omitempty"`
}

//DeviceTwinUpdate the struct of device twin update
type DeviceTwinUpdate struct {
	BaseMessage
	Twin map[string]*MsgTwin `json:"twin"`
}

//createActualUpdateMessage function is used to create the device twin update message
func createActualUpdateMessage(tempValue string, hudValue string) DeviceTwinUpdate {
	var deviceTwinUpdateMessage DeviceTwinUpdate
	actualMap := map[string]*MsgTwin{"temperature": {Actual: &TwinValue{Value: &tempValue}, Metadata: &TypeMetadata{Type: "Updated"}}, "humidity": {Actual: &TwinValue{Value: &hudValue}, Metadata: &TypeMetadata{Type: "Updated"}}}
	deviceTwinUpdateMessage.Twin = actualMap
	return deviceTwinUpdateMessage
}

func publishToMqtt(temp int, hud int) {
	updateMessage := createActualUpdateMessage(strconv.Itoa(12), strconv.Itoa(13))
	twinUpdateBody, _ := json.Marshal(updateMessage)

	token := client.Publish(topic_device, 0, false, twinUpdateBody)

	if token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
	}
}

func connectToMqtt() mqtt.Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(mqttUrl)
	opts.SetClientID("mapper_mqtt_client")
	opts.SetUsername(user)
	opts.SetPassword(passwd)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler

	client = mqtt.NewClient(opts)

	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
	}

	return client
}

func subscribe(client mqtt.Client) {
	token := client.Subscribe(topic1, 1, func(client mqtt.Client, msg mqtt.Message) {
		fmt.Println(msg.Topic(), string(msg.Payload()))
		// fmt.Printf("* [%s] %s\n", msg.Topic(), string(msg.Payload()))
		if topic1_payload != string(msg.Payload()) {
			topic1_payload = string(msg.Payload())
			fmt.Println(topic1_payload)
			// json.unMarshal()
		}
	})
	token.Wait()
}

func main() {
	stopchan := make(chan os.Signal)
	signal.Notify(stopchan, syscall.SIGINT, syscall.SIGKILL)
	defer close(stopchan)

	client = connectToMqtt()
	subscribe(client)

	select {
	case <-stopchan:
		fmt.Printf("Interrupt, exit.\n")
		break
	}
}

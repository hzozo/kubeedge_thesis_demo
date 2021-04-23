package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var client mqtt.Client    // the mqtt client used to subscribe
var pubclient mqtt.Client // the mqtt client used to publish
var topic_payload []byte

const (
	mqttUrl      = "tcp://127.0.0.1:1883"
	topic_device = "<your_topic>"
	topic_edge   = "$hw/events/device/<your_device>/twin/update"
)

//DeviceStateUpdate is the structure used in updating the device state
type DeviceStateUpdate struct {
	State string `json:"state,omitempty"`
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("Connected")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Connect lost: %v", err)
}

type SensorData struct {
	Temperature float32 `json:"temperature"`
	Humidity    float32 `json:"humidity"`
	Battery     int     `json:"battery"`
	Average     int     `json:"average"`
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

func createActualUpdateMessage(tempValue string, hudValue string) DeviceTwinUpdate {
	var deviceTwinUpdateMessage DeviceTwinUpdate
	actualMap := map[string]*MsgTwin{"temperature": {Actual: &TwinValue{Value: &tempValue}, Metadata: &TypeMetadata{Type: "Updated"}}, "humidity": {Actual: &TwinValue{Value: &hudValue}, Metadata: &TypeMetadata{Type: "Updated"}}}
	deviceTwinUpdateMessage.Twin = actualMap
	return deviceTwinUpdateMessage
}

func publishToMqtt(temp float32, hud float32) {
	updateMessage := createActualUpdateMessage(fmt.Sprintf("%.1f", temp), fmt.Sprintf("%.1f", hud))
	twinUpdateBody, _ := json.Marshal(updateMessage)

	token := pubclient.Publish(topic_edge, 0, false, twinUpdateBody)

	if token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
	}
}

func connectToMqtt(clientID string) mqtt.Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(mqttUrl)
	opts.SetClientID(clientID)
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
	token := client.Subscribe(topic_device, 1, func(client mqtt.Client, msg mqtt.Message) {
		if string(topic_payload) != string(msg.Payload()) {
			topic_payload = msg.Payload()
			var sensorData SensorData
			err := json.Unmarshal([]byte(topic_payload), &sensorData)
			fmt.Println("error", err)
			publishToMqtt(sensorData.Temperature, sensorData.Humidity)
		}
	})
	token.Wait()
}

func main() {
	stopchan := make(chan os.Signal)
	signal.Notify(stopchan, syscall.SIGINT, syscall.SIGKILL)
	defer close(stopchan)

	client = connectToMqtt("subsciption_<device_number>")
	pubclient = connectToMqtt("publishing_<device_number>")
	subscribe(client)

	select {
	case <-stopchan:
		fmt.Printf("Interrupt, exit.\n")
		break
	}
}

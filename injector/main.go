package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var first_client mqtt.Client  // the mqtt client used to subscribe to first device
var second_client mqtt.Client // the mqtt client used to subscribe to second device
var pub_client mqtt.Client    // the mqtt client used to publish
var topic1_payload []byte
var topic2_payload []byte

const (
	mqttUrl             = "tcp://192.168.1.104:1883" //"tcp://127.0.0.1:1883"
	first_topic_device  = "sensors/livingroom1"
	second_topic_device = "sensors/livingroom2"
	topic_edge          = "$hw/events/device/hudtemp-aggregated/twin/update"
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
	updateMessage := createActualUpdateMessage(fmt.Sprintf("%.2f", temp), fmt.Sprintf("%.2f", hud))
	twinUpdateBody, _ := json.Marshal(updateMessage)

	token := pub_client.Publish(topic_edge, 0, false, twinUpdateBody)

	if token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
	}
}

func connectToMqtt(clientID string, client mqtt.Client) mqtt.Client {
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

func subscribe(client1 mqtt.Client, client2 mqtt.Client) {
	var sensorData1 SensorData
	var sensorData2 SensorData
	token1 := client1.Subscribe(first_topic_device, 1, func(client1 mqtt.Client, msg mqtt.Message) {
		fmt.Println(string(msg.Payload()))
		if string(topic1_payload) != string(msg.Payload()) {
			topic1_payload = msg.Payload()
			err := json.Unmarshal([]byte(topic1_payload), &sensorData1)
			fmt.Println("error", err)
			aggregate_hudtemp(sensorData1, sensorData2)
		}
	})
	token1.Wait()
	token2 := client2.Subscribe(second_topic_device, 1, func(client2 mqtt.Client, msg mqtt.Message) {
		fmt.Println(string(msg.Payload()))
		if string(topic2_payload) != string(msg.Payload()) {
			topic2_payload = msg.Payload()
			err := json.Unmarshal([]byte(topic2_payload), &sensorData2)
			fmt.Println("error", err)
			aggregate_hudtemp(sensorData1, sensorData2)
		}
	})
	token2.Wait()
}

func aggregate_hudtemp(sensorData1 SensorData, sensorData2 SensorData) {
	rand.Seed(time.Now().UnixNano())
	var final_temp float32
	var final_hud float32
	avg_temp := (sensorData1.Temperature + sensorData2.Temperature) / 2
	avg_hud := (sensorData1.Humidity + sensorData2.Humidity) / 2
	min_temp := avg_temp - 5
	min_hud := avg_hud - 5
	max_temp := min_temp + 10
	max_hud := min_hud + 10
	rand_temp := rand.Float32()*(max_temp-min_temp) + min_temp //rand.Intn(int(max_temp)-int(min_temp)) + int(min_temp)
	rand_hud := rand.Float32()*(max_hud-min_hud) + min_hud     //rand.Intn(int(max_hud)-int(min_hud)) + int(min_hud)
	fmt.Println(rand_temp)
	// now comes the implementation of the TMR
	if sensorData1.Temperature-sensorData2.Temperature < 1 {
		final_temp = avg_temp
	}
	if sensorData1.Humidity-sensorData2.Humidity < 1 {
		final_hud = avg_hud
	}
	if rand_temp-sensorData1.Temperature < 1 {
		final_temp = (rand_temp + sensorData1.Temperature) / 2
	}
	if rand_hud-sensorData2.Humidity < 1 {
		final_hud = avg_hud
	}
	fmt.Println("publishing")
	publishToMqtt(final_temp, final_hud)
}

func main() {
	stopchan := make(chan os.Signal)
	signal.Notify(stopchan, syscall.SIGINT, syscall.SIGKILL)
	defer close(stopchan)

	first_client = connectToMqtt("subsciption_3", first_client)
	second_client = connectToMqtt("subsciption_4", second_client)
	pub_client = connectToMqtt("publishing_3", pub_client)
	subscribe(first_client, second_client)

	select {
	case <-stopchan:
		fmt.Printf("Interrupt, exit.\n")
		break
	}
}

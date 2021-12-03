package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var firstClient mqtt.Client      // the mqtt client used to subscribe to first device
var secondClient mqtt.Client     // the mqtt client used to subscribe to second device
var publishingClient mqtt.Client // the mqtt client used to publish
var firstTopicPayload []byte
var secondTopicPayload []byte

const (
	mqttUrl           = "tcp://127.0.0.1:1883"
	firstDeviceTopic  = "sensors/<first_room>"
	secondDeviceTopic = "sensors/<second_room>"
	edgeTopic         = "$hw/events/device/hudtemp-aggregated/twin/update"
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

	token := publishingClient.Publish(edgeTopic, 0, false, twinUpdateBody)

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
	token1 := client1.Subscribe(firstDeviceTopic, 1, func(client1 mqtt.Client, msg mqtt.Message) {
		fmt.Println(string(msg.Payload()))
		if string(firstTopicPayload) != string(msg.Payload()) {
			firstTopicPayload = msg.Payload()
			err := json.Unmarshal([]byte(firstTopicPayload), &sensorData1)
			fmt.Println("error", err)
			aggregateHudTemp(sensorData1, sensorData2, generateThirdSignal(sensorData1, sensorData2))
		}
	})
	token1.Wait()
	token2 := client2.Subscribe(secondDeviceTopic, 1, func(client2 mqtt.Client, msg mqtt.Message) {
		fmt.Println(string(msg.Payload()))
		if string(secondTopicPayload) != string(msg.Payload()) {
			secondTopicPayload = msg.Payload()
			err := json.Unmarshal([]byte(secondTopicPayload), &sensorData2)
			fmt.Println("error", err)
			aggregateHudTemp(sensorData1, sensorData2, generateThirdSignal(sensorData1, sensorData2))
		}
	})
	token2.Wait()
}

func generateThirdSignal(sensorData1 SensorData, sensorData2 SensorData) SensorData {
	// in this section, we produce the 3rd signal required for TMR
	var sensorData3 SensorData
	rand.Seed(time.Now().UnixNano())
	avg_temp := (sensorData1.Temperature + sensorData2.Temperature) / 2
	avg_hud := (sensorData1.Humidity + sensorData2.Humidity) / 2
	min_temp := avg_temp - 5
	min_hud := avg_hud - 5
	max_temp := min_temp + 10
	max_hud := min_hud + 10
	sensorData3.Temperature = rand.Float32()*(max_temp-min_temp) + min_temp
	sensorData3.Humidity = rand.Float32()*(max_hud-min_hud) + min_hud
	return sensorData3
}

func getVote(sensorReading float32, average [3]float32) int {
	var vote int = 0
	for i := 0; i < 3; i++ {
		if math.Abs(float64(sensorReading-average[i])) < math.Abs(float64(sensorReading-average[vote])) {
			vote = i
		}
	}
	return vote
}

func aggregateHudTemp(sensorData1 SensorData, sensorData2 SensorData, sensorData3 SensorData) {
	// now comes the implementation of the TMR
	var finalTemp float32
	var finalHud float32
	var tempAverages [3]float32
	var hudAverages [3]float32
	tempAverages[0] = (sensorData1.Temperature + sensorData2.Temperature) / 2
	tempAverages[1] = (sensorData1.Temperature + sensorData3.Temperature) / 2
	tempAverages[2] = (sensorData2.Temperature + sensorData3.Temperature) / 2
	hudAverages[0] = (sensorData1.Humidity + sensorData2.Humidity) / 2
	hudAverages[1] = (sensorData1.Humidity + sensorData3.Humidity) / 2
	hudAverages[2] = (sensorData2.Humidity + sensorData3.Humidity) / 2

	var temp_votes [3]int
	temp_votes[0] = getVote(sensorData1.Temperature, tempAverages)
	temp_votes[1] = getVote(sensorData2.Temperature, tempAverages)
	temp_votes[2] = getVote(sensorData3.Temperature, tempAverages)

	var hud_votes [3]int
	hud_votes[0] = getVote(sensorData1.Humidity, hudAverages)
	hud_votes[1] = getVote(sensorData2.Humidity, hudAverages)
	hud_votes[2] = getVote(sensorData3.Humidity, hudAverages)

	if temp_votes[0] == temp_votes[1] {
		finalTemp = tempAverages[temp_votes[0]]
	} else if temp_votes[0] == temp_votes[2] {
		finalTemp = tempAverages[temp_votes[0]]
	} else if temp_votes[1] == temp_votes[2] {
		finalTemp = tempAverages[temp_votes[1]]
	} else {
		return
	}

	if hud_votes[0] == hud_votes[1] {
		finalHud = hudAverages[hud_votes[0]]
	} else if hud_votes[0] == hud_votes[2] {
		finalHud = hudAverages[hud_votes[0]]
	} else if hud_votes[1] == hud_votes[2] {
		finalHud = tempAverages[hud_votes[1]]
	} else {
		return
	}

	publishToMqtt(finalTemp, finalHud)
}

func main() {
	stopchan := make(chan os.Signal)
	signal.Notify(stopchan, syscall.SIGINT, syscall.SIGKILL)
	defer close(stopchan)

	firstClient = connectToMqtt("subsciption_3", firstClient)
	secondClient = connectToMqtt("subsciption_4", secondClient)
	publishingClient = connectToMqtt("publishing_3", publishingClient)
	subscribe(firstClient, secondClient)

	select {
	case <-stopchan:
		fmt.Printf("Interrupt, exit.\n")
		break
	}
}

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/hzozo/kubeedge_thesis_demo/mapper/utils"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"k8s.io/client-go/rest"
)

var client mqtt.Client
var pubclient mqtt.Client
var topic1_payload []byte
var topic2_payload string

const (
	mqttUrl      = "tcp://192.168.1.28:1883"
	user         = "zozo"
	passwd       = "1994Zozo"
	topic1       = "sensors/livingroom1"
	topic2       = "sensors/livingroom2"
	topic_device = "$hw/events/device/hudtemp1/twin/update"
)

//DeviceStateUpdate is the structure used in updating the device state
type DeviceStateUpdate struct {
	State string `json:"state,omitempty"`
}

// The device id of the counter
var deviceID = "hudtemp1"

// The default namespace in which the counter device instance resides
var namespace = "default"

// The CRD client used to patch the device instance.
var crdClient *rest.RESTClient

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

func initialize() {
	// Create a client to talk to the K8S API server to patch the device CRDs
	kubeConfig, err := utils.KubeConfig()
	if err != nil {
		log.Fatalf("Failed to create KubeConfig, error : %v", err)
	}
	log.Println("Get kubeConfig successfully")

	crdClient, err = utils.NewCRDClient(kubeConfig)
	if err != nil {
		log.Fatalf("Failed to create device crd client , error : %v", err)
	}
	log.Println("Get crdClient successfully")
}

/* func UpdateStatus() map[string]string {
	result := DeviceStatus{}
	raw, _ := crdClient.Get().Namespace(namespace).Resource(utils.ResourceTypeDevices).Name(deviceID).DoRaw(context.TODO())

	status := map[string]string{
		"status": "OFF",
		"value":  "0",
	}

	_ = json.Unmarshal(raw, &result)
	for _, twin := range result.Status.Twins {
		status["status"] = twin.Desired.Value
		status["value"] = twin.Reported.Value
	}

	return status
}

// UpdateDeviceTwinWithDesiredTrack patches the desired state of
// the device twin with the command.
func UpdateDeviceTwinWithDesiredTrack(cmd string) bool {
	if cmd == originCmd {
		return true
	}

	status := buildStatusWithDesiredTrack(cmd)
	deviceStatus := &DeviceStatus{Status: status}
	body, err := json.Marshal(deviceStatus)
	if err != nil {
		log.Printf("Failed to marshal device status %v", deviceStatus)
		return false
	}
	result := crdClient.Patch(utils.MergePatchType).Namespace(namespace).Resource(utils.ResourceTypeDevices).Name(deviceID).Body(body).Do(context.TODO())
	if result.Error() != nil {
		log.Printf("Failed to patch device status %v of device %v in namespace %v \n error:%+v", deviceStatus, deviceID, namespace, result.Error())
		return false
	} else {
		log.Printf("Turn %s %s", cmd, deviceID)
	}
	originCmd = cmd

	return true
}
*/
/* func publishToMqtt(cli *client.Client, temperature float32) {
	deviceTwinUpdate := "$hw/events/device/" + "temperature" + "/twin/update"

	updateMessage := createActualUpdateMessage(strconv.Itoa(int(temperature)) + "C")
	twinUpdateBody, _ := json.Marshal(updateMessage)

	cli.Publish(&client.PublishOptions{
		TopicName: []byte(deviceTwinUpdate),
		QoS:       mqtt.QoS0,
		Message:   twinUpdateBody,
	})
 }
*/

/* func buildStatusWithDesiredTrack(cmd string) devices.DeviceStatus {
	metadata := map[string]string{
		"timestamp": strconv.FormatInt(time.Now().Unix()/1e6, 10),
		"type":      "string",
	}
	twins := []devices.Twin{{PropertyName: "status", Desired: devices.TwinProperty{Value: cmd, Metadata: metadata}, Reported: devices.TwinProperty{Value: statusMap[cmd], Metadata: metadata}}}
	devicestatus := devices.DeviceStatus{Twins: twins}
	return devicestatus
}
*/
//createActualUpdateMessage function is used to create the device twin update message
func createActualUpdateMessage(tempValue string, hudValue string) DeviceTwinUpdate {
	var deviceTwinUpdateMessage DeviceTwinUpdate
	actualMap := map[string]*MsgTwin{"temperature": {Actual: &TwinValue{Value: &tempValue}, Metadata: &TypeMetadata{Type: "Updated"}}, "humidity": {Actual: &TwinValue{Value: &hudValue}, Metadata: &TypeMetadata{Type: "Updated"}}}
	deviceTwinUpdateMessage.Twin = actualMap
	return deviceTwinUpdateMessage
}

func publishToMqtt(temp float32, hud float32) {
	updateMessage := createActualUpdateMessage(fmt.Sprintf("%f", temp), fmt.Sprintf("%f", hud))
	twinUpdateBody, _ := json.Marshal(updateMessage)

	token := pubclient.Publish(topic_device, 0, false, twinUpdateBody)

	if token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
	}
}

func connectToMqtt(clientID string) mqtt.Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(mqttUrl)
	opts.SetClientID(clientID)
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
		if string(topic1_payload) != string(msg.Payload()) {
			topic1_payload = msg.Payload()
			var sensorData SensorData
			err := json.Unmarshal([]byte(topic1_payload), &sensorData)
			fmt.Println("error", err)
			// fmt.Println("unmarshalled", sensorData.Temperature)
			publishToMqtt(sensorData.Temperature, sensorData.Humidity)
		}
	})
	token.Wait()
}

func main() {
	stopchan := make(chan os.Signal)
	signal.Notify(stopchan, syscall.SIGINT, syscall.SIGKILL)
	defer close(stopchan)
	// initialize()
	client = connectToMqtt("subsciption")
	pubclient = connectToMqtt("publishing")
	subscribe(client)

	select {
	case <-stopchan:
		fmt.Printf("Interrupt, exit.\n")
		break
	}
}

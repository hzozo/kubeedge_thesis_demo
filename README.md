# KubeEdge Temperatue and Humidity Sensor Demo

## Description

This demo requires the user to have two 

Counter run at edge side, and user can control it in web from cloud side, also can get counter value in web from cloud side.

![function model](./images/function-level_model.jpg)


## Prerequisites

### Hardware Prerequisites

* RaspBerry PI (RaspBerry PI 3 Model B+ has been used for this demo). Bluetooth connection is a must.

### Software Prerequisites

* A running Kubernetes cluster.

* KubeEdge v1.8.1+

* MQTT Broker is running on Raspi.

## Steps to run the demo

### Create the device model and device instances in Kubernetes for the temperature and humidity sensor

In this step, we create the device model and the instances for the temperature and humidity sensor using the yaml files.
Execute the below commands in the directory the repository was cloned to.

```console
$ sed -i "s#edge-node#<your edge node name>#" crds/hudtemp-*.yaml
$ kubectl create -f crds/hudtemp-model.yaml
$ kubectl create -f crds/hudtemp-instance1.yaml
$ kubectl create -f crds/hudtemp-instance2.yaml
$ kubectl create -f crds/hudtemp-aggregated.yaml
```

### Prepare Bluetooth Gateway application

The Bluetooth Gateway application connects to the sensor and publishes the measurements to MQTT.
For this to work, the user has to set up the Bluetooth Gateway so it knows what to connect to.

# The user should execute the below step twice

```console
$ cd kubeedge_thesis_demo
$ cd xiaomi-ble-mqtt
$ vim devices.ini
$ vim mqtt.ini
$ cd kubeedge_thesis_demo
$ docker build -t kubeedge-pi-hudtemp<number_of_device>:v1.0
```

**Note: instance must be created after model and deleted before model.**

### Run KubeEdge Pi Counter App

The KubeEdge Counter App run in raspi.

```console
$ cd $GOPATH/src/github.com/kubeedge/examples/kubeedge-counter-demo/counter-mapper
$ make
$ make docker
$ cd $GOPATH/src/github.com/kubeedge/examples/kubeedge-counter-demo/templates
$ kubectl create -f kubeedge-pi-counter-app.yaml
```

The App will subscribe to the `$hw/events/device/counter/twin/update/document` topic, and when it receives the expected control command on the topic, it will turn on/off the counter, also it will fresh counter value and publish value to `$hw/events/device/counter/twin/update` topic, then the latest counter status will be sychronized between edge and cloud.

At last, user can get the counter status at cloud side.


### Control counter by visiting Web App Page

* Visit web app page by the web app link `MASTER_NODE_IP:80`.

  ![web ui](./images/web-ui.png)

* Choose `ON` option, and click `Execute`, then user can see counter start to count by `docker logs -f counter-container-id` at edge side.

* Choose `STATUS` option, then click `Execute` to get the counter status, finally counter status and current counter value will display in web.

  also you can watch counter status by `kubectl get device counter -o yaml -w` at cloud side.

* Choose `OFF` option, and click `Execute`, counter stop work at edge side.

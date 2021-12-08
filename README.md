# KubeEdge Temperatue and Humidity Sensor Demo

## Purpose of the project

This project was established in order to demonstrate the possibility of the use of cheap technological devices in edge computing. This project uses 3 cheap sensors - capable of reading temperature and humidity levels - in order to provide one reliable signal utilizing the capabilities of edge computing. The goal is to prove that one can make reliable measurements using cheaper devices at a possibly lower price, even when more of these devices are required.

## Description

This demo requires the user to have two Xiaomi MiJia Bluetooth Temperature and Humidity sensors physically, which devices' readings will be taken and processed as part of this demo.
The above readings will then be supplemented by a third signal that is generated randomly by the signal aggregator application.
To save important memory resources (as the Raspberry Pi 3 has a very limited amount), the generation of the 3rd signal and the aggregation of the signals using a triple modular redundant algorithm, as well as the publishing of the aggregated signal all takes place in the signal aggregator application.

![function model](./images/function-level_model.jpg)


## Prerequisites

### Hardware Prerequisites

* RaspBerry PI (RaspBerry PI 3 Model B+ has been used for this demo). Bluetooth connection is a must.

* Xiaomi MiJia LYWSDCGQ Temperature and Humidity Sensor

### Software Prerequisites

* A running Kubernetes cluster.

* KubeEdge v1.8.1+

* MQTT Broker is running on Raspi.

## Steps to run the demo

**Note: instances must be created after model and deleted before model.**

### Create the device model and device instances in Kubernetes for the temperature and humidity sensor

In this step, we create the device model and the instances for the temperature and humidity sensor using the yaml files.
Execute the below commands in the directory the repository was cloned to.

##### Execute below commands on the cloud node

```console
$ sed -i 's/<edge_node>/<name_of_edge_node>/g' crds/hudtemp-*.yaml
$ kubectl create -f crds/hudtemp-model.yaml
$ kubectl create -f crds/hudtemp-instance1.yaml
$ kubectl create -f crds/hudtemp-instance2.yaml
$ kubectl create -f crds/hudtemp-aggregated.yaml
```

### Preparation of Bluetooth Gateway application

The Bluetooth Gateway application connects to the sensor and publishes the measurements via MQTT.
For this to work, the user has to create the Docker images of the Bluetooth Gateway so it knows what to connect to (aka. has the correct configuration).

##### Execute the below commands on the edge node

```console
$ cd kubeedge_thesis_demo
$ cd xiaomi-ble-mqtt
$ cp devices.ini.sample devices.ini
//The user should set the mac address of the device and specify the correct rooms in the section headers and in the topic and availability topic as well in the devices.ini file
$ vim devices.ini
$ cp mqtt.ini.sample mqtt.ini
//The user should not change the client name and must remove the username and password fields in the mqtt.ini file
$ vim mqtt.ini
$ cd kubeedge_thesis_demo
$ docker build -t kubeedge-bl-gw:v1.0 .
```

### Instantiation of the Bluetooth Gateway application

This is the instantiation of the just configured Bluetooth Gateway docker image.

##### Execute below commands on the cloud node

```console
$ cd kubeedge_thesis_demo
$ sed -i 's/<edge_node>/<name_of_edge_node>/g' crds/kubeedge-bl-gw.yaml
$ kubectl create -f crds/kubeedge-bl-gw.yaml
```

At this point, the information published to MQTT which needs to be processed and passed onto KubeEdge.

### Preparation of the mapper applications

The mapper application is written in Golang and its purpose is to process the previously published data via MQTT and publish it via MQTT once again, this time in a standard format defined by KubeEdge.

#### Below commands must be executed on the edge node

```console
$ cd kubeedge_thesis_demo/mapper
$ cp main.go main.go.bak
// We'll build the first mapper application at this point, with the user defined rooms
$ sed -i 's/<your_topic>/<first_previously_defined_room>/g' main.go
$ sed -i 's/<device_number>/1/g' main.go
$ sed -i 's/<your_device>/hudtemp1/g' main.go
$ go build -o sensor-app .
$ docker build -t kubeedge-sensor1-mapper .
// And now we'll build the second mapper application
$ cp main.go.bak main.go
$ sed -i 's/<your_topic>/<second_previously_defined_room>/g' main.go
$ sed -i 's/<device_number>/2/g' main.go
$ sed -i 's/<your_device>/hudtemp2/g' main.go
$ go build -o sensor-app .
$ docker build -t kubeedge-sensor2-mapper .
```

Now we have the Docker images ready on the edge node.

### Instantiation of the mapper applications

In this step we'll instantiate the mapper applications we've previously prepared. This is done using Kubernetes.

##### Execute below commands on the cloud node

```console
$ cd kubeedge_thesis_demo
$ sed -i 's/<edge_node>/<name_of_edge_node>/g' crds/kubeedge-hudtemp*.yaml
$ kubectl create -f crds/kubeedge-hudtemp-mapper1.yaml
$ kubectl create -f crds/kubeedge-hudtemp-mapper2.yaml
```

### Publish the signal aggregator application

The signal aggregator application is written in go and its purpose is to introduce a third temperature and humidity measurement, so we have 3 signals to run triple modular redundancy on and to implement a triple module redundant algorithm to produce the aggregated signal using that.

#### Below commands must be executed on the edge node

```console
$ cd kubeedge_thesis_demo/signal_aggregator
// We'll build the injector application at this point, with the user defined rooms
$ sed -i 's/<first_room>/<first_previously_defined_room>/g' main.go
$ sed -i 's/<second_room>/<second_previously_defined_room>/g' main.go
$ go build -o aggregator .
$ docker build -t aggregator:v1.0 .
```

Now we have the Docker image ready on the edge node.

### Instantiation of the signal aggregator application

In this step we'll instantiate the mapper applications we've previously prepared. This is done using Kubernetes.

##### Execute below commands on the cloud node

```console
$ cd kubeedge_thesis_demo
$ kubectl create -f crds/kubeedge-hudtemp-aggregated-signal.yaml
```

After the above commands have been executed, the Docker containers will soon be instantiated on the edge node and the measurements made by the temperature and humidity sensors will be published to Kubernetes, too. You can check it by executing the 'kubectl get device hudtemp-aggregated'.
FROM hzozo/kubeedge-pi-hudtemp:v0.5.2
  
COPY xiaomi-ble-mqtt/mqtt.ini /root/xiaomi/mqtt.ini
COPY xiaomi-ble-mqtt/devices.ini /root/xiaomi/devices.ini

CMD cron && tail -f /var/log/cron.log
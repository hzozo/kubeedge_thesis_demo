# Based on ubuntu
FROM ubuntu:20.04
LABEL maintainers="KubeEdge Community Developer"
LABEL description="KubeEdge Raspi Counter App"

# Copy from build directory
COPY sensor-app /sensor-app

# Define default command
ENTRYPOINT ["/sensor-app"]

# Run the executable
CMD ["sensor-app"]

# Use an Ubuntu base image
FROM ubuntu:22.04

# Set environment variables
ENV DEBIAN_FRONTEND=noninteractive

# Install necessary packages
# Tagging on linux requires the attr library
RUN apt-get update && apt-get install -y \
    git \
    wget \
    curl \
    build-essential \
    attr \ 
    && rm -rf /var/lib/apt/lists/*

# Install Go 1.23
RUN wget https://go.dev/dl/go1.23.0.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go1.23.0.linux-amd64.tar.gz && \
    rm go1.23.0.linux-amd64.tar.gz

# Set up Go environment
ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH="/go"

# Create app directory
WORKDIR /app

# Copy your Go project into the container
COPY . .

# Install dependencies
RUN go mod tidy

# Set entrypoint to bash, so we can pass commands dynamically
ENTRYPOINT ["bash", "-c"]

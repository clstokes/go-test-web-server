package main

import (
  "fmt"
  "log"
  "net/http"
  "os"
  "os/signal"
  "time"

  "github.com/garyburd/redigo/redis"
)

/*
 * Environment variables:
 * - NODE_DATACENTER - datacenter for metric keys
 * - REDIS_ADDRESS - address of the Redis server - ie. redis.service.consul:6379
 */

var NODE_NAME = "-node-server-count"
var listenPort = "8000"

// ab -c 20 -n 10000 http://localhost:8000/ seems to exhaust open connections
// for some reason

func main() {
  redisConn := getRedisConnection()
  if redisConn != nil {
    redisConn.Do("INCR", getNodeMetricKey())
    redisConn.Close()
  }

  makeShutdownChannel()

  fmt.Println("Listening on port " + listenPort)

  http.HandleFunc("/", handleRequest)
  http.HandleFunc("/health", handleHealthCheckRequest)
  http.ListenAndServe(":"+listenPort, nil)
}

func makeShutdownChannel() {
  sigch := make(chan os.Signal, 1)
  signal.Notify(sigch, os.Interrupt)

  go func() {
    <-sigch
    redisConn := getRedisConnection()
    if redisConn != nil {
      redisConn.Do("DECR", getNodeMetricKey())
    }
    redisConn.Close()
    fmt.Println("Exiting...")
    os.Exit(0)
  }()
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
  logRequest(r)

  redisConn := getRedisConnection()
  defer redisConn.Close()

  if redisConn != nil {
    redisConn.Do("INCR", getRequestMetricKey())
  }

  // artificial delay to catch in-flight request metrics more easily
  time.Sleep(1000 * time.Millisecond)

  fmt.Fprintf(w, time.Now().UTC().Format(time.RFC3339))

  if redisConn != nil {
    redisConn.Do("DECR", getRequestMetricKey())
  }
}

func handleHealthCheckRequest(w http.ResponseWriter, r *http.Request) {
  logRequest(r)
  fmt.Fprintf(w, "{\"status\": \"ok\"}")
}

func logRequest(r *http.Request) {
  fmt.Println("Handling request for %s from %s.", r.URL, r.RemoteAddr)
}

func getRequestMetricKey() string {
  return getDatacenterKey() + "-server-request-count"
}

func getNodeMetricKey() string {
  return getDatacenterKey() + NODE_NAME
}

func getDatacenterKey() string {
  key := os.Getenv("NODE_DATACENTER")
  if key == "" {
    key = "default"
  }
  return key
}

func getRedisConnection() redis.Conn {
  redisAddr := os.Getenv("REDIS_ADDRESS")
  if redisAddr == "" {
    redisAddr = "localhost:6379"
  }

  redisConn, err := redis.Dial("tcp", redisAddr)
  if err != nil {
    log.Fatalf("error connecting to redis: %v", err)
    return nil
  }

  return redisConn
}

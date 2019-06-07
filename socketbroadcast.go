package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var channelNumber int
var broadcastMessageChan = make(chan []string, 4)

func uptimeServer(serverChan chan chan string) {
	var clients []chan string
	// uptimeChan := make(chan int, 1)
	// // This goroutine will count our uptime in the background, and write
	// // updates to uptimeChan:
	// go func(target chan int) {
	// 	i := 0
	// 	for {
	// 		time.Sleep(time.Second)
	// 		i++
	// 		target <- i
	// 	}
	// }(uptimeChan)
	// And now we listen to new clients and new uptime messages:
	for {
		select {
		case client, _ := <-serverChan:
			clients = append(clients, client)
		case message, _ := <-broadcastMessageChan:
			// Send the uptime to all connected clients:
			for i := 0; i < len(clients); i++ { // handle disconnection from clients
				// for i, c := range clients {
				select {
				case <-clients[i]:
					log.Printf("A channel closed!\n")
					clients = append(clients[:i], clients[i+1:]...)
					i--
					continue
				default:
					clients[i] <- fmt.Sprintf("New message received from %v : %v", message[0], message[1])
				}

			}
		}
	}
}

func sendMessageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	message := make([]string, 2)
	message[0] = "curl"
	message[1] = vars["url"]
	log.Println("request with message: ", message)
	broadcastMessageChan <- message
}

func main() {
	// This upgrader is needed for WebSocket connections later:
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // Disable CORS for testing
		},
	}

	// Start the server and keep track of the channel that it receives
	// new clients on:

	serverChan := make(chan chan string, 4)
	go uptimeServer(serverChan)
	r := mux.NewRouter()
	// Define a HTTP handler function for the /status endpoint, that can receive
	// WebSocket-connections only... so note that browsing it with your browser will fail.
	r.HandleFunc("/broadcast", func(w http.ResponseWriter, r *http.Request) {
		// Upgrade this HTTP connection to a WS connection:
		ws, _ := upgrader.Upgrade(w, r, nil)
		// And register a client for this connection with the uptimeServer:
		channelNumber++
		myChannelNumber := channelNumber
		client := make(chan string, 1)
		serverChan <- client
		// notify := w.(http.CloseNotifier).CloseNotify()
		// And now check for uptimes written to the client indefinitely.
		// Yes, we are lacking proper error and disconnect checking here, too:
		log.Printf("channel %v joined\n", channelNumber)
		messageWsChan := make(chan []string, 1)
		go func(messageChan chan []string, myChannelNumber int) {
			for {
				_, message, err := ws.ReadMessage()
				if err != nil {
					log.Println("read:", err)
					break
				}
				log.Printf("recv: %v %s\n", myChannelNumber, message)
				messageWsChan <- []string{strconv.Itoa(myChannelNumber), string(message)}
			}
		}(messageWsChan, myChannelNumber)
	loop:
		for {

			select {
			// case <-notify:
			// 	log.Printf("channel %v disconnected\n", channelNumber)
			//  close(client)
			//  break loop
			case text, _ := <-client:
				writer, err := ws.NextWriter(websocket.TextMessage)
				if err != nil {
					log.Printf("channel %v disconnected\n", myChannelNumber)
					break loop
				}
				if _, err := writer.Write([]byte(text)); err != nil {
					log.Printf("channel %v disconnected\n", myChannelNumber)
					break loop
				}
				writer.Close()
			case message, _ := <-messageWsChan:
				broadcastMessageChan <- message
			}
		}
	})

	r.HandleFunc("/{url}", sendMessageHandler).Methods("GET")
	log.Println("serving 8080")
	http.ListenAndServe(":8080", r)
}

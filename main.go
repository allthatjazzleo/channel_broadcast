package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

var newMessageChan = make(chan string, 4)

func main() {
	serverChan := make(chan chan string, 4)
	go server(serverChan)

	// Connect the clients:
	client1Chan := make(chan string, 4)
	client2Chan := make(chan string, 4)
	fmt.Println(client1Chan)
	fmt.Println(client2Chan)
	serverChan <- client2Chan
	serverChan <- client1Chan

	// Notice that we have to start the clients in their own goroutine,// because we would have a deadlock otherwise:
	go client("Client 1", client1Chan)
	go client("Client 2", client2Chan)

	r := mux.NewRouter()
	r.HandleFunc("/{url}", getHandler).Methods("GET")
	log.Fatal(http.ListenAndServe(":3000", r))
	// Just a dirty hack to wait for everything to finish up.
	// A clean and safe approach would have been too much boilerplate code
	// for this blog-post
	// time.Sleep(2 * time.Second)
}

func server(serverChan chan chan string) {
	var clients []chan string
	for {
		select {
		case client, _ := <-serverChan:
			client <- fmt.Sprintf("chan %v is newly join", client)
			clients = append(clients, client)
			// fmt.Println(len(clients))
			// Broadcast the number of clients to all clients:
		case m, _ := <-newMessageChan:
			for _, c := range clients {
				// fmt.Println(ii)
				c <- fmt.Sprintf("broadcast message: %v", m)
			}
		}
	}
}

func client(clientName string, clientChan chan string) {
	for {
		text, _ := <-clientChan
		log.Printf("%s: %s\n", clientName, text)
	}
}

func getHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	message := vars["url"]
	log.Println("request with message: ", message)
	newMessageChan <- message
}

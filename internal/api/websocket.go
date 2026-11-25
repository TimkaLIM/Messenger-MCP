package api

import (
	"log"

	"github.com/gorilla/websocket"
)

//Client - один подключённый пользователь

type Client struct {
	Conn   *websocket.Conn
	ChatID int
}

type Broadcast struct {
	ChatID          int
	ReceivedMessage []byte
}

type Hub struct {
	//Мапа комнат: chat_id -> список клиентов
	rooms map[int]map[*Client]bool

	register   chan *Client
	unregister chan *Client
	broadcast  chan Broadcast // сюда кидаем готовый JSON для рассылки
}

func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[int]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan Broadcast),
	}
}

// Run - главный цикл хаба, который будет крутиться в отдельной горутине
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register: //клиент хочет подключиться к команте
			_, ok := h.rooms[client.ChatID]
			if !ok { //если комната не существует
				h.rooms[client.ChatID] = make(map[*Client]bool)
			}

			h.rooms[client.ChatID][client] = true
			// в конце добавляем клиента обязательно

		case client := <-h.unregister: //клиент хочет уйти из комнаты
			_, ok := h.rooms[client.ChatID][client]
			if ok { //если клиент есть в комнате - удаляем
				delete(h.rooms[client.ChatID], client)
				err := client.Conn.Close()
				if err != nil {
					log.Println("Ошибка закрытия соединения клиента: ", err)
				}
				if len(h.rooms[client.ChatID]) == 0 { //если комната пустая, удаляем её
					delete(h.rooms, client.ChatID)
				}
			}
		case message := <-h.broadcast: //пришло сообщение, рассылаем всем клиентам
			for client := range h.rooms[message.ChatID] { //проходимся по всем клиентам
				err := client.Conn.WriteMessage(websocket.TextMessage, message.ReceivedMessage)
				if err != nil {
					log.Println("Ошибка рассылки сообщения: ", err)
					h.unregister <- client
				}
			}

		}
	}
}

package api

import (
	"MCPmessenger/internal/DataBase"
	"MCPmessenger/internal/models"
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"

	"github.com/gorilla/websocket"

	"github.com/golang-jwt/jwt/v5"
)

//STRUCTS BLOCK START

type contextKey string

const userIDKey contextKey = "user_id"

type Application struct {
	DB        models.DBInterface
	SecretKey string
	Hub       *Hub
}

type ResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *ResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, errors.New("underlying ResponseWriter is not a Hijacker")
}

//STRUCTS BLOCK END

// HANDLERS BLOCK START

func (a *Application) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := a.DB.PingContext(r.Context()); err != nil {
		log.Println("Ошибка пинга в healthHandler:", err)
		w.WriteHeader(http.StatusInternalServerError)
		if _, err = w.Write([]byte(`{"status":"db_error"}`)); err != nil {
			log.Println("Ошибка записи w.Write: ", err)
		}
		return
	}
	if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
		log.Println("Ошибка записи w.Write: ", err)
	}

}

func echoHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1048576)
	var book map[string]any
	if jsonErr := json.NewDecoder(r.Body).Decode(&book); jsonErr != nil {
		log.Println("Ошибка декодинга echo:", jsonErr)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if jsonErr2 := json.NewEncoder(w).Encode(book); jsonErr2 != nil {
		log.Println("Ошибка энкодинга echo: ", jsonErr2)
		return
	}
}

func (a *Application) RegisterUser(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := r.Body.Close(); err != nil {
			log.Println("Ошибка закрытия r.Body: ", err)
		}
	}()
	w.Header().Set("Content-Type", "application/json")
	var NewUser models.User
	err := json.NewDecoder(r.Body).Decode(&NewUser)
	if err != nil {
		log.Println("Ошибка регистрации пользователя: ", err)
		w.WriteHeader(400)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(NewUser.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Println("Ошибка создания хэша пароля: ", err)
		w.WriteHeader(500)
		return
	}
	var pgErr *pgconn.PgError
	_, err = a.DB.ExecContext(r.Context(), "INSERT INTO users (username,password_hash) VALUES($1,$2)",
		NewUser.Username, hash)
	if err != nil {
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				log.Println("Ошибка регистрации пользователя в БД(дубликат): ", err)
				w.WriteHeader(409)
				if _, err = w.Write([]byte(`{"status":"name is occupied"}`)); err != nil {
					log.Println("Ошибка записи w.Write: ", err)
				}
				return
			}
		} else {
			log.Println("Ошибка регистрации пользователя в БД: ", err)
			w.WriteHeader(500)
			if _, err = w.Write([]byte(`{"status":"register_error"}`)); err != nil {
				log.Println("Ошибка записи w.Write: ", err)
			}
			return
		}
	}
	w.WriteHeader(http.StatusCreated)
	if _, err = w.Write([]byte(`{"status":"user_is_registred"}`)); err != nil {
		log.Println("Ошибка записи w.Write: ", err)
	}
}

func (a *Application) LoginUser(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := r.Body.Close(); err != nil {
			log.Println("Ошибка закрытия r.Body: ", err)
		}
	}()
	w.Header().Set("Content-Type", "application/json")
	var LoginUser models.User
	jsonErr := json.NewDecoder(r.Body).Decode(&LoginUser)
	if jsonErr != nil {
		log.Println("Ошибка декодирования логина/пароля: ", jsonErr)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	//Проверка пароля и логина
	user, err := DataBase.GetUserByUsername(r.Context(), a.DB, LoginUser.Username, LoginUser.Password)
	if err != nil {
		log.Println("Неверный логин или пароль", err)
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"status":"Bad Login or Password"}`))
		return
	}

	claims := jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	}
	//Создаем токен с алгоритмом HS256
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	secretKey := []byte(a.SecretKey)
	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		log.Println("Ошибка создания токена: ", err)
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`{"status":"login false"}`))
		return
	}
	_, _ = fmt.Fprintf(w, `{"status":"correct login and password","token":"%s"}`, tokenString)

}

func (a *Application) protectedHandler(w http.ResponseWriter, r *http.Request) {
	userName := r.Context().Value("username").(string)

	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(w, `{"status":"ok", "message":"Доступ разрешен", "username": %s}`, userName)
}

//CHATS

//CREATE CHAT

func (a *Application) CreateChatHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	//Принимаем на вход JSON, название чата
	var ChatInfo models.CreateChatRequest
	err := json.NewDecoder(r.Body).Decode(&ChatInfo)
	if err != nil {
		log.Println("Ошибка декодирования информации о чате: ", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	//У нас есть структура с чатом, создадим запись о чате в БД
	//Но перед этим создадим транзакцию, потому что если упадет добавление пользователя в чат, то пустая БД никому не нужна
	ctx := r.Context()
	tx, err := a.DB.BeginTx(ctx, nil)
	if err != nil {
		log.Println("Ошибка создания BEGIN при создании чата: ", err)
		w.WriteHeader(500)
		return
	}
	defer func() {
		if err = tx.Rollback(); err != nil {
			log.Println("Ошибка закрытия tx.Rollback: ", err)
		}
	}()
	userID := r.Context().Value("user_id").(int)

	var ChatID int
	invite_code, err := DataBase.GenerateInviteCode(6)
	if err != nil {
		log.Println("Ошибка генерации кода приглашения: ", err)
		w.WriteHeader(500)
		return
	}
	err = tx.QueryRowContext(ctx, "INSERT INTO chats (title,invite_code) VALUES($1,$2) RETURNING id", ChatInfo.Title, invite_code).Scan(&ChatID)
	if err != nil {
		log.Println("Ошибка создания чата в БД: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO chat_members (chat_id,user_id) VALUES($1,$2)", ChatID, userID)
	if err != nil {
		log.Println("Ошибка добавления пользователя в chat_members: ", err)
		w.WriteHeader(500)
		return
	}
	err = tx.Commit()
	if err != nil {
		log.Println("Ошибка коммита к БД(создание чата): ", err)
		w.WriteHeader(500)
		return
	}
	_, _ = w.Write([]byte(`{"status":"Chat is created!"}`))
}

func (a *Application) SelectAllChatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ctx := r.Context()
	userID := r.Context().Value("user_id").(int)
	rows, err := a.DB.QueryContext(ctx, "SELECT cm.chat_id,c.title FROM chat_members cm JOIN chats c ON cm.chat_id = c.id\n"+
		"WHERE cm.user_id = $1", userID)
	if err != nil {
		log.Println("Ошибка вывода всех чатов из БД: ", err)
		w.WriteHeader(500)
		return
	}
	defer func() {
		if err = rows.Close(); err != nil {
			log.Println("Ошибка закрытия rows: ", err)
		}
	}()
	var chats_list []models.ChatResponse
	for rows.Next() {
		var chat models.ChatResponse
		if err = rows.Scan(&chat.ID, &chat.Title); err != nil {
			log.Println("Ошибка сканирования id&title чата: ", err)
			continue
		}
		chats_list = append(chats_list, chat)
	}
	if err = rows.Err(); err != nil {
		log.Println("Ошибка при чтении списка чатов:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(chats_list)
	if err != nil {
		log.Println("Ошибка энкодирования чатов в JSON: ", err)
		w.WriteHeader(500)
		return
	}
}

func (a *Application) JoinToChatHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "application/json")
	var Request models.JoinChat
	err := json.NewDecoder(r.Body).Decode(&Request)
	if err != nil {
		log.Println("Ошибка декодирования заявки в чат: ", err)
		w.WriteHeader(400)
		return
	}
	ctx := r.Context()
	var chatID int
	err = a.DB.QueryRowContext(ctx, "SELECT id FROM chats WHERE invite_code = $1", Request.InviteCode).Scan(&chatID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"invite code not found"}`))
			return
		}
		log.Println("Ошибка БД:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	userID := r.Context().Value("user_id").(int)
	_, err = a.DB.ExecContext(ctx, "INSERT INTO chat_members (chat_id,user_id) VALUES($1,$2)", chatID, userID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			w.WriteHeader(http.StatusConflict) // 409
			_, _ = w.Write([]byte(`{"error":"you are already in this chat"}`))
			return
		}
		// Если другая ошибка — 500
		log.Println("Ошибка добавления в чат:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, _ = w.Write([]byte(`{"status":"joined_success"}`))

}

func (a *Application) SendMessageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	chatIDStr := r.PathValue("id")
	chatID, err := strconv.Atoi(chatIDStr)
	if err != nil {
		log.Println("Ошибка создания id чата(отправка сообщения): ", err)
		w.WriteHeader(400)
		return
	}
	var message models.ReceivedMessage
	err = json.NewDecoder(r.Body).Decode(&message)
	if err != nil {
		log.Println("Ошибка принятия сообщения: ", err)
		w.WriteHeader(400)
		return
	}
	senderID := r.Context().Value(userIDKey).(int)
	ctx := r.Context()
	var exists int
	err = a.DB.QueryRowContext(ctx, "SELECT 1 FROM chat_members WHERE user_id = $1 AND chat_id = $2", senderID, chatID).Scan(&exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Println("Пользователь пытается писать в чужой чат!")
			w.WriteHeader(http.StatusForbidden) // 403
			_, _ = w.Write([]byte(`{"error":"access denied"}`))
			return
		}
		// 5. Если это другая ошибка (база упала) — возвращаем 500
		log.Println("Ошибка БД при проверке участника:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var messageID int
	var createdAT time.Time
	err = a.DB.QueryRowContext(ctx, "INSERT INTO messages (chat_id,sender_id,text) VALUES($1,$2,$3) RETURNING id,created_at",
		chatID, senderID, message.Text).Scan(&messageID, &createdAT)
	if err != nil {
		log.Println("Ошибка добавления сообщения в messages: ", err)
		w.WriteHeader(500)
		return
	}

	//send message on websocket channel
	JSONMessage, err := json.Marshal(map[string]any{
		"id":         messageID,
		"sender_id":  senderID,
		"text":       message.Text,
		"created_at": createdAT,
	})
	if err != nil {
		log.Println("Ошибка создания JSON для передачи"+
			"сообщения в Websocket канал: ", err)
		w.WriteHeader(500)
		return
	}

	BroadCastMessage := Broadcast{
		ChatID:          chatID,
		ReceivedMessage: JSONMessage,
	}
	a.Hub.broadcast <- BroadCastMessage

	w.WriteHeader(http.StatusCreated)
	if err = json.NewEncoder(w).Encode(map[string]any{
		"id":         messageID,
		"created_at": createdAT,
	}); err != nil {
		log.Println("Ошибка энкодинга сообщения: ", err)
		w.WriteHeader(500)
		return
	}

}

func (a *Application) GetMessages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "application/json")
	var (
		beforeStr string
		limitStr  string
		err       error
		before    int
		limit     int
	)
	//BEFORE
	beforeStr = r.URL.Query().Get("before")
	if beforeStr == "" {
		before = 0
	} else {
		before, err = strconv.Atoi(beforeStr)
		if err != nil {
			log.Println("Ошибка присвоения before(int): ", err)
			before = 0
		}
	}
	//LIMIT
	limitStr = r.URL.Query().Get("limit")
	if limitStr == "" {
		limit = 50
	} else {
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			log.Println("Ошибка присвоения limit(str): ", err)
			limit = 50
		}
	}

	ctx := r.Context()
	chatIDStr := r.PathValue("id")
	chatID, err := strconv.Atoi(chatIDStr)
	if err != nil {
		log.Println("Ошибка превращения id чата из строки в число: ", err)
		w.WriteHeader(500)
		return
	}
	query := "SELECT id,sender_id,text,created_at FROM messages WHERE chat_id = $1"
	args := []any{chatID}
	if before > 0 {
		query += " AND id <= $2"
		args = append(args, before)
	}
	query += " ORDER BY id ASC LIMIT $" + strconv.Itoa(len(args)+1)
	args = append(args, limit)
	rows, err := a.DB.QueryContext(ctx, query, args...)
	if err != nil {
		log.Println("Ошибка выгрузки сообщений из БД: ", err)
		w.WriteHeader(500)
		return
	}
	defer func() {
		if err = rows.Close(); err != nil {
			log.Println("Ошибка закрытия rows: ", err)
		}
	}()
	var messages []models.MessagesResponse
	for rows.Next() {
		var message models.MessagesResponse
		err = rows.Scan(&message.ID, &message.SenderID, &message.Text, &message.CreatedAT)
		if err != nil {
			log.Println("Ошибка выгрузки сообщения(1): ", err)
			w.WriteHeader(500)
			return
		}
		messages = append(messages, message)
	}
	if rows.Err() != nil {
		log.Println("Ошибка при чтении результатов из БД: ", rows.Err())
		w.WriteHeader(500)
		return

	}
	err = json.NewEncoder(w).Encode(messages)
	if err != nil {
		log.Println("Ошибка, Энкодинга сообщений: ", err)
		w.WriteHeader(500)
		return
	}
}

//создаём глобально, чтобы каждый раз не создавать заново

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true //разрешаем подключение с любых доменов (для разработки)
	},
}

func (a *Application) WebSocketHandler(w http.ResponseWriter, r *http.Request) {

	userID := r.Context().Value(userIDKey).(int)

	chat_id_str := r.PathValue("chat_id")
	chat_id, err := strconv.Atoi(chat_id_str)
	if err != nil {
		log.Println("Ошибка перевода id чата из строки в число: ", err)
		w.WriteHeader(400)
		return
	}
	//check user is chat_member
	var exists int
	err = a.DB.QueryRowContext(r.Context(), "SELECT 1 FROM chat_members WHERE chat_id = $1 AND user_id = $2", chat_id, userID).Scan(&exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Пользователя НЕТ в этом чате
			log.Println("Попытка подключения к чужому чату:", userID, "->", chat_id)
			w.WriteHeader(403)
			return
		}
		// Другая ошибка БД (например, база упала)
		log.Println("Ошибка БД при проверке участия в чате:", err)
		w.WriteHeader(500)
		return
	}
	//создаём канал для клиента
	clientConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Ошибка обновления соединения: ", err)
		w.WriteHeader(500)
		return
	}
	client := &Client{
		Conn:   clientConn,
		ChatID: chat_id,
	}
	//если пользователь состоит в чате
	//отправляем клиента в канал добавления в комнату
	a.Hub.register <- client

	//запускаем чтение
	go func() {
		defer func() {
			a.Hub.unregister <- client
			err = client.Conn.Close()
			if err != nil {
				log.Println("Ошибка закрытия канала клиента: ", err)
			}
		}()

		for {
			//Читаем сообщения от клиента
			_, _, err = client.Conn.ReadMessage()
			if err != nil {
				log.Println("Ошибка чтения сообщения пользователя: ", err)
				break
			}
		}
	}()
	//Когда клиент закрывает соединение(например закрыл браузер) - ошибку соединения ловит ReadMessage()
	//И когда err!=nil -> срабатывает break, цикл закрывается и дальше срабатывает defer func()
	//Он удаляет клиента из комнаты и закрывает его канал

}

// HANDLERS BLOCK END

//Middleware BLOCK START

func (a *Application) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"no token"}`))
			return
		}

		//отрезаем Bearer
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(a.SecretKey), nil
		})

		if err != nil || !token.Valid {
			// Токен протух или подпись не сошлась
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		//если дошли сюда -> токен валиден, все работает, claims заполнен
		userID := int(claims["user_id"].(float64))
		log.Println("Авторизован пользователь с id: ", userID)

		//кладём id в context

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

func (rw *ResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		//Этот код выполняется до того, как попадет Handler(любой)
		start := time.Now()
		method := r.Method
		path := r.URL.Path
		w2 := &ResponseWriter{ResponseWriter: w, statusCode: 200}

		//Передаем управление дальше в хэндлер
		next.ServeHTTP(w2, r)

		status := w2.statusCode

		//Этот код выполнится после того как хэндлер завершится
		duration := time.Since(start)
		fmt.Printf("[LOG] %s %s %d| Время: %v\n", method, path, status, duration)
	})
}

// Функция для обертки хэндлэра в логгер, а потом сразу отправить в mux
func registerRoute(mux *http.ServeMux, methodAndPath string,
	handler func(w http.ResponseWriter, r *http.Request)) {

	logger := loggingMiddleware(http.HandlerFunc(handler))
	mux.Handle(methodAndPath, logger)
}

//MIDDLEWARE BLOCK END

func ServerStart(database models.DBInterface) {
	secret := os.Getenv("token_key")
	mux := http.NewServeMux()
	hub := NewHub()
	go hub.Run()
	app := Application{
		DB:        database,
		SecretKey: secret,
		Hub:       hub,
	}

	//РЕГИСТРАЦИЯ ХЭНДЛЕРОВ
	registerRoute(mux, "GET /health", app.healthHandler)
	registerRoute(mux, "POST /echo", echoHandler)
	registerRoute(mux, "GET /boom", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	registerRoute(mux, "POST /register", app.RegisterUser)
	registerRoute(mux, "POST /login", app.LoginUser)
	protectedHandler := loggingMiddleware(app.AuthMiddleware(http.HandlerFunc(app.protectedHandler)))
	mux.Handle("POST /protected", protectedHandler)
	craftChatHandler := loggingMiddleware(app.AuthMiddleware(http.HandlerFunc(app.CreateChatHandler)))
	mux.Handle("POST /chats/createchat", craftChatHandler)
	SelectChatsHandler := loggingMiddleware(app.AuthMiddleware(http.HandlerFunc(app.SelectAllChatsHandler)))
	mux.Handle("GET /chats/mychats", SelectChatsHandler)
	joinChatHandler := loggingMiddleware(app.AuthMiddleware(http.HandlerFunc(app.JoinToChatHandler)))
	mux.Handle("POST /chats/jointochat", joinChatHandler)
	MessageHandler := loggingMiddleware(app.AuthMiddleware(http.HandlerFunc(app.SendMessageHandler)))
	mux.Handle("POST /chats/{id}/message", MessageHandler)
	GetMessagesHandler := loggingMiddleware(app.AuthMiddleware(http.HandlerFunc(app.GetMessages)))
	mux.Handle("GET /chats/{id}/messages", GetMessagesHandler)
	WebJuiceHandler := loggingMiddleware(app.AuthMiddleware(http.HandlerFunc(app.WebSocketHandler)))
	mux.Handle("GET /ws/{chat_id}", WebJuiceHandler)

	//ОТКРЫТИЕ СЕРВЕРА
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,   //максимум 5 сек на чтение запроса
		WriteTimeout: 10 * time.Second,  //Максимум 10 сек на отправку ответа
		IdleTimeout:  120 * time.Second, //Время жизни keep-alive соединения
	}
	//в горутине, для shutdown
	go func() {
		fmt.Println("Сервер запущен на :8080")
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed { //Проверка сразу 2 ошибок, просто ошибка, если правда что-то не так
			log.Fatalf("Ошибка сервера: %v", err) //Чтобы эта ошибка не выводилась при намеренном закрытии сервера череза ctrl C
		}
	}()
	//ждем сигнал Ctrl+C
	quit := make(chan os.Signal, 1)   //канал для сигналов ОС
	signal.Notify(quit, os.Interrupt) //если придет сигнал ctrl+C(interrupt) положи его в канал quit
	<-quit                            //программа замирает здесь, пока ты не нажмешь Ctrl+C

	log.Println("Сервер выключается")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil { //принудительное заверешние сервера, запрещает новые сообщения
		log.Fatal("Принудительное завершение: ", err) //+ ждет контекст 5 секунд
	}
	log.Println("Сервер успешно остановлен")
}

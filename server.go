package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"

	"github.com/golang-jwt/jwt/v5"
)

// HANDLERS BLOCK START
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

func (a *Application) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := a.DB.Ping(); err != nil {
		log.Println("Ошибка пинга в healthHandler:", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"status":"db_error"}`))
		return
	}
	w.Write([]byte(`{"status":"ok"}`))
}

func (a *Application) RegisterUser(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	var NewUser User
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
	_, err = a.DB.Exec("INSERT INTO users (username,password_hash) VALUES($1,$2)",
		NewUser.Username, hash)
	if err != nil {
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				log.Println("Ошибка регистрации пользователя в БД(дубликат): ", err)
				w.WriteHeader(409)
				w.Write([]byte(`{"status":"name is occupied"}`))
				return
			}
		} else {
			log.Println("Ошибка регистрации пользователя в БД: ", err)
			w.Write([]byte(`{"status":"register_error"}`))
			return
		}
	}
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"status":"user_is_registred"}`))
}

func (a *Application) LoginUser(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	var LoginUser User
	jsonErr := json.NewDecoder(r.Body).Decode(&LoginUser)
	if jsonErr != nil {
		log.Println("Ошибка декодирования логика/пароля: ", jsonErr)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	//Проверка пароля и логина
	err := GetUserByUsername(a.DB, LoginUser.Username, LoginUser.Password)
	if err != nil {
		log.Println("Неверный логин или пароль", err)
		w.WriteHeader(400)
		w.Write([]byte(`{"status":"Bad Login or Password"}`))
		return
	}
	w.Write([]byte(`{"status":"correct login and password"}`))

	claims := jwt.MapClaims{
		"username": LoginUser.Username,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	}
	//Создаем токен с алгоритмом HS256
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	secretKey := []byte(a.SecretKey)
	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		log.Println("Ошибка создания токена: ", err)
		w.WriteHeader(500)
		w.Write([]byte(`{"status":"login false"}`))
		return
	}
	fmt.Fprintf(w, `{"token":"%s"}`, tokenString)

}

func (a *Application) protectedHandler(w http.ResponseWriter, r *http.Request) {
	userName := r.Context().Value("username").(string)

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"ok", "message":"Доступ разрешен", "username": %s}`, userName)
}

//CHATS

//CREATE CHAT

func (a *Application) CreateChatHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	//Принимаем на вход JSON, название чата
	var ChatInfo Chat
	err := json.NewDecoder(r.Body).Decode(&ChatInfo)
	if err != nil {
		log.Println("Ошибка декодирования информации о чате: ", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	//У нас есть структура с чатом, создадим запись о чате в БД
	//Но перед этим создадим транзакцию, потому что если упадет добавление пользователя в чат, то пустая БД никому не нужна
	_, err = a.DB.Exec("INSERT INTO chats (title) VALUES($1)", ChatInfo.Title)
	if err != nil {
		log.Println("Ошибка создания чата в БД: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write([]byte(`{"status":"Chat is created!"}`))
}

// HANDLERS BLOCK END

//Middleware BLOCK START

func (a *Application) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"no token"}`))
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
		userName := claims["username"].(string)
		log.Println("Авторизован пользователь с Username: ", userName)

		//кладём id в context

		ctx := context.WithValue(r.Context(), "username", userName)
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

func ServerStart(database *sql.DB) {

	secret := os.Getenv("token_key")
	mux := http.NewServeMux()
	app := Application{
		DB:        database,
		SecretKey: secret,
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
	mux.Handle("POST /createchat", craftChatHandler)

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

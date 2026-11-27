package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ReadMessagesInput struct {
	ChatID int `json:"chat_id" jsonschema:"ID чата для чтения сообщений"`
	Before int `json:"before,omitempty" jsonschema:"Сообщения до которого будут выгружены сообщения"`
	Limit  int `json:"limit" jsonschema:"То, сколько будет получено сообщений"`
}

type MessageResponse struct {
	ID        int       `json:"id"`
	SenderID  int       `json:"sender_id"`
	Text      string    `json:"text"`
	CreatedAT time.Time `json:"created_at"`
}

//AddTool автоматически сериализует его в JSON

type ReadMessagesOutput struct {
	Messages []MessageResponse `json:"messages"`
	Total    int               `json:"total"`
}

func ReadMessagesHandler(ctx context.Context, req *mcp.CallToolRequest, input ReadMessagesInput) (*mcp.CallToolResult,
	ReadMessagesOutput, error) {

	if input.ChatID <= 0 {
		return nil, ReadMessagesOutput{}, fmt.Errorf("chat_id должен быть больше 0")
	}
	if input.Limit <= 0 {
		input.Limit = 50
	}
	if input.Limit > 100 {
		input.Limit = 100
	}
	query := fmt.Sprintf(
		"http://localhost:8080/chats/%d/messages?before=%d&limit=%d",
		input.ChatID,
		input.Before,
		input.Limit)

	//создаем http запрос с контекстом
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, query, nil)
	if err != nil {
		return nil, ReadMessagesOutput{}, fmt.Errorf("создание HTTP-запроса: %w", err)
	}
	//выполняем запрос
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, ReadMessagesOutput{}, fmt.Errorf("HTTP GET запрос: %w", err)
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			log.Println("Ошибка закрытия тела ответа: ", err)
		}
	}()
	//проверяем статус
	if resp.StatusCode != http.StatusOK {
		return nil, ReadMessagesOutput{}, fmt.Errorf("неожиданный статус ответа: %d", resp.StatusCode)
	}

	//декодируем ответ
	var messages []MessageResponse
	err = json.NewDecoder(resp.Body).Decode(&messages)
	if err != nil {
		log.Println("Ошибка докедирования тела ответа: ", err)
		return nil, ReadMessagesOutput{}, fmt.Errorf("декодирование JSON: %w", err)
	}

	output := ReadMessagesOutput{
		Messages: messages,
		Total:    len(messages),
	}
	callResult := &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: fmt.Sprintf("Found %d messages in chat %d", len(messages), input.ChatID),
			},
		},
	}

	return callResult, output, nil
}

func main() {
	//создаём сервер
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "messenger-mcp",
		Version: "1.0"},
		nil,
	)

	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        "read_messages",
			Description: "Читает сообщения из указанного чата",
		},
		ReadMessagesHandler,
	)

	//создаем HTTP handler для MCP
	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, nil,
	)

	//настраиваем HTTP сервер
	http.Handle("/mcp", handler)

	fmt.Println("🚀 MCP server listening on :9090")
	fmt.Println("📡 Endpoint: http://localhost:9090/mcp")

	log.Fatal(http.ListenAndServe(":9090", nil))
}

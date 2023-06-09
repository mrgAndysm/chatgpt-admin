package routes

import (
	"chatgpt-go/global"
	"chatgpt-go/model"
	"chatgpt-go/service"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
	"io"
	"net/http"
	"net/url"
)

func VerifyEndpoint(c *gin.Context) {
	var req model.VerifyRequest
	if err := c.BindJSON(&req); err != nil || !isValidToken(req.Token) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"status":  "Error",
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "Success",
		"message": "Verify successfully",
		"data":    nil,
	})
}

func isValidToken(token string) bool {
	return token != "" && global.Config.System.AuthSecretKey == token
}

func SessionEndpoint(c *gin.Context) {
	authorizationHeader := "Bearer " + global.OpenAIKey
	c.Request.Header.Set("Authorization", authorizationHeader)

	authSecretKey := global.Config.System.AuthSecretKey
	isAuthenticated := authSecretKey != ""

	response := createResponse(isAuthenticated)
	c.JSON(http.StatusOK, response)
}

func createResponse(isAuthenticated bool) gin.H {
	return gin.H{
		"status":  "Success",
		"message": "",
		"data": gin.H{
			"auth":  isAuthenticated,
			"model": "ChatGPTAPI",
		},
	}
}

func GetConfig(c *gin.Context) {
	response, err := service.ChatConfig()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, response)
}

//func ChatProcess(chatStorage *ChatStorage) gin.HandlerFunc {
//	return func(c *gin.Context) {
//		// 设置响应头的 Content-Type 为 application/octet-stream
//		c.Header("Content-Type", "application/octet-stream")
//
//		// 获取响应写入器对象，并判断是否支持刷新缓冲区
//		flusher, ok := c.Writer.(http.Flusher)
//		if !ok {
//			c.AbortWithError(http.StatusInternalServerError, errors.New("Streaming not supported"))
//			return
//		}
//
//		// 解析请求参数
//		var req model.ChatRequest
//		err := c.BindJSON(&req)
//		if err != nil {
//			c.AbortWithError(http.StatusBadRequest, err)
//			return
//		}
//
//		if global.Config.System.OpenAIKey == "" {
//			panic(errors.New("Missing OPENAI_API_KEY environment variable"))
//		}
//
//		config := openai.DefaultConfig(global.Config.System.OpenAIKey)
//		socksHost := global.Config.System.SocksHost
//		socksPort := global.Config.System.SocksPort
//		httpsProxy := global.Config.System.HttpsProxy
//
//		if socksHost != "" && socksPort != "" {
//			proxyUrl, err := url.Parse("socks5://" + socksHost + ":" + socksPort)
//			if err != nil {
//				panic(err)
//			}
//			transport := &http.Transport{
//				Proxy: http.ProxyURL(proxyUrl),
//			}
//			config.HTTPClient = &http.Client{
//				Transport: transport,
//			}
//		} else if httpsProxy != "" {
//			proxyUrl, err := url.Parse("http://" + httpsProxy)
//			if err != nil {
//				panic(err)
//			}
//			transport := &http.Transport{
//				Proxy: http.ProxyURL(proxyUrl),
//			}
//			config.HTTPClient = &http.Client{
//				Transport: transport,
//			}
//		}
//
//		client := openai.NewClientWithConfig(config)
//
//		if req.Options.ParentMessageId == "" {
//			req.Options.ParentMessageId = uuid.NewString()
//		}
//		newMessageId := uuid.NewString()
//		chatStorage.AddMessage(newMessageId, req.Options.ParentMessageId, openai.ChatCompletionMessage{
//			Role:    openai.ChatMessageRoleUser,
//			Content: req.Prompt,
//		})
//		messages, err := chatStorage.GetMessages(newMessageId)
//		reqData := openai.ChatCompletionRequest{
//			Model:    openai.GPT3Dot5Turbo,
//			Messages: messages,
//			Stream:   true,
//		}
//
//		fmt.Printf("Request data: %v\n", reqData)
//		stream, err := client.CreateChatCompletionStream(c, reqData)
//		if err != nil {
//			fmt.Printf("CompletionStream error: %v\n", err)
//			return
//		}
//		defer stream.Close()
//
//		text := ""
//		messageId := ""
//		for {
//			response, err := stream.Recv()
//
//			if errors.Is(err, io.EOF) {
//				if messageId != "" {
//					chatStorage.AddMessage(messageId, newMessageId, openai.ChatCompletionMessage{
//						Role:    openai.ChatMessageRoleAssistant,
//						Content: text,
//					})
//				}
//				fmt.Println("Stream finished")
//				return
//			}
//
//			if err != nil {
//				fmt.Printf("Stream error: %v\n", err)
//				return
//			}
//
//			fmt.Printf("		Stream response: %v\n", response)
//
//			messageId = response.ID
//			text = text + response.Choices[0].Delta.Content
//			resp := model.ChatResponse{
//				Role:            openai.ChatMessageRoleAssistant,
//				Id:              response.ID,
//				ParentMessageId: newMessageId,
//				Text:            text,
//				Delta:           response.Choices[0].Delta.Content,
//				Detail:          response,
//			}
//			jsonResp, err := json.Marshal(resp)
//			if err != nil {
//				fmt.Printf("JSON marshaling error: %v\n", err)
//				return
//			}
//
//			_, err = c.Writer.Write(jsonResp)
//			if err != nil {
//				fmt.Printf("Writing response error: %v\n", err)
//				return
//			}
//
//			// 刷新缓冲区，发送数据
//			flusher.Flush()
//
//			// 在 response 结构体后面添加换行符，以便进行流式传输
//			_, err = c.Writer.Write([]byte("\n"))
//			if err != nil {
//				fmt.Printf("Writing newline error: %v\n", err)
//				return
//			}
//		}
//	}
//}

func ChatProcess(chatStorage *ChatStorage) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 设置响应头的 Content-Type 为 application/octet-stream
		c.Header("Content-Type", "application/octet-stream")

		// 获取响应写入器对象，并判断是否支持刷新缓冲区
		//flusher, ok := c.Writer.(http.Flusher)
		//if !ok {
		//	c.AbortWithError(http.StatusInternalServerError, errors.New("Streaming not supported"))
		//	return
		//}

		// 解析请求参数
		var req model.ChatRequest
		err := c.BindJSON(&req)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		if global.Config.System.OpenAIKey == "" {
			panic(errors.New("Missing OPENAI_API_KEY environment variable"))
		}

		config := openai.DefaultConfig(global.Config.System.OpenAIKey)
		socksHost := global.Config.System.SocksHost
		socksPort := global.Config.System.SocksPort
		httpsProxy := global.Config.System.HttpsProxy

		if socksHost != "" && socksPort != "" {
			proxyUrl, err := url.Parse("socks5://" + socksHost + ":" + socksPort)
			if err != nil {
				panic(err)
			}
			transport := &http.Transport{
				Proxy: http.ProxyURL(proxyUrl),
			}
			config.HTTPClient = &http.Client{
				Transport: transport,
			}
		} else if httpsProxy != "" {
			proxyUrl, err := url.Parse("http://" + httpsProxy)
			if err != nil {
				panic(err)
			}
			transport := &http.Transport{
				Proxy: http.ProxyURL(proxyUrl),
			}
			config.HTTPClient = &http.Client{
				Transport: transport,
			}
		}

		client := openai.NewClientWithConfig(config)

		if req.Options.ParentMessageId == "" {
			req.Options.ParentMessageId = uuid.NewString()
		}
		newMessageId := uuid.NewString()
		chatStorage.AddMessage(newMessageId, req.Options.ParentMessageId, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: req.Prompt,
		})
		messages, err := chatStorage.GetMessages(newMessageId)
		reqData := openai.ChatCompletionRequest{
			Model:    openai.GPT3Dot5Turbo,
			Messages: messages,
			Stream:   true,
		}

		fmt.Printf("Request data: %v\n", reqData)
		stream, err := client.CreateChatCompletionStream(c, reqData)
		if err != nil {
			fmt.Printf("CompletionStream error: %v\n", err)
			return
		}
		defer stream.Close()

		text := ""
		messageId := ""
		firstChunk := true
		for {
			response, err := stream.Recv()

			if errors.Is(err, io.EOF) {
				if messageId != "" {
					chatStorage.AddMessage(messageId, newMessageId, openai.ChatCompletionMessage{
						Role:    openai.ChatMessageRoleAssistant,
						Content: text,
					})
				}
				fmt.Println("Stream finished")
				return
			}

			if err != nil {
				fmt.Printf("Stream error: %v\n", err)
				return
			}

			fmt.Printf("		Stream response: %v\n", response)

			messageId = response.ID
			text = text + response.Choices[0].Delta.Content
			resp := model.ChatResponse{
				Role:            openai.ChatMessageRoleAssistant,
				Id:              response.ID,
				ParentMessageId: newMessageId,
				Text:            text,
				Delta:           response.Choices[0].Delta.Content,
				Detail:          response,
			}
			jsonResp, err := json.Marshal(resp)
			if err != nil {
				fmt.Printf("JSON marshaling error: %v\n", err)
				return
			}

			if firstChunk {
				//fmt.Fprintf(c.Writer, "%s", jsonResp)
				c.Writer.Write(jsonResp)
				firstChunk = false
			} else {
				c.Writer.Write([]byte("\n"))
				c.Writer.Write(jsonResp)
				//fmt.Fprintf(c.Writer, "\n%s", jsonResp)
			}
			c.Writer.Flush()
			//c.Writer.(http.Flusher).Flush()
		}
	}
}

//func chatProcessHandler(chatStorage *ChatStorage) http.HandlerFunc {
//	return func(c *gin.Context, w http.ResponseWriter, r *http.Request) {
//		// 设置响应头的 Content-type 为 application/octet-stream
//		w.Header().Set("Content-type", "application/octet-stream")
//
//		// 解析请求参数
//		var requestProps struct {
//			Prompt  string `json:"prompt"`
//			Options struct {
//				ConversationId  string `json:"conversationId,omitempty"`
//				ParentMessageId string `json:"parentMessageId,omitempty"`
//			} `json:"options,omitempty"`
//			SystemMessage string  `json:"systemMessage,omitempty"`
//			Temperature   float32 `json:"temperature,omitempty"`
//			TopP          float32 `json:"top_p,omitempty"`
//		}
//		if err := json.NewDecoder(r.Body).Decode(&requestProps); err != nil {
//			http.Error(w, err.Error(), http.StatusBadRequest)
//			return
//		}
//
//		// 解析请求参数
//		var req model.ChatRequest
//		err := c.BindJSON(&req)
//		if err != nil {
//			c.AbortWithError(http.StatusBadRequest, err)
//			return
//		}
//		// 检查全局配置中是否存在 API 密钥
//		// 这里假设密钥已经存储在系统环境变量中或者其他安全地位置
//		//apiKey := global.Config.System.OpenAIKey
//
//		newMessageId := uuid.NewString()
//
//		messages, err := chatStorage.GetMessages(newMessageId)
//		// 创建聊天过程
//		messageID := uuid.NewString()
//		chatStorage.AddMessage(newMessageId, req.Options.ParentMessageId, openai.ChatCompletionMessage{
//			Role:    openai.ChatMessageRoleUser,
//			Content: req.Prompt,
//		})
//		completionRequest := openai.ChatCompletionRequest{
//			Model:       openai.GPT3Dot5Turbo,
//			MaxTokens:   150,
//			Temperature: requestProps.Temperature,
//			TopP:        requestProps.TopP,
//			Stream:      true,
//			Messages:    messages,
//		}
//
//		config := openai.DefaultConfig(global.Config.System.OpenAIKey)
//		client := openai.NewClientWithConfig(config)
//		stream, err := client.CreateChatCompletionStream(c, completionRequest)
//		if err != nil {
//			http.Error(w, err.Error(), http.StatusInternalServerError)
//			return
//		}
//
//		// 处理聊天过程
//		defer stream.Close()
//		firstChunk := true
//		for {
//			completionResponse, err := stream.Recv()
//			if err == io.EOF {
//				break
//			}
//			if err != nil {
//				http.Error(w, err, http.StatusInternalServerError)
//				return
//			}
//
//			messageId = response.ID
//			text = text + response.Choices[0].Delta.Content
//			resp := model.ChatResponse{
//				Role:            openai.ChatMessageRoleAssistant,
//				Id:              response.ID,
//				ParentMessageId: newMessageId,
//				Text:            text,
//				Delta:           response.Choices[0].Delta.Content,
//				Detail:          response,
//			}
//			chatMessage := model.ChatResponse{
//				ID:    generateMessageID(),
//				Text:  completionResponse.Choices[0].Delta.Content,
//				IsBot: true,
//			}
//			chatStorage.Add(messageID, chatMessage.Text)
//			responseJSON, err := json.Marshal(chatMessage)
//			if err != nil {
//				http.Error(w, err.Error(), http.StatusInternalServerError)
//				return
//			}
//			if firstChunk {
//				fmt.Fprintf(w, "%s", responseJSON)
//				firstChunk = false
//			} else {
//				fmt.Fprintf(w, "\n%s", responseJSON)
//			}
//			w.(http.Flusher).Flush()
//		}
//	}
//}

//func ChatProcessaa(chatStorage *ChatStorage) gin.HandlerFunc {
//
//	return func(c *gin.Context) {
//		if firstChunk {
//			fmt.Fprintf(w, "%s", responseJSON)
//			firstChunk = false
//		} else {
//			fmt.Fprintf(w, "\n%s", responseJSON)
//		}
//	}
//}

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/viper"
)

const (
	tokenUrl    = "https://ngw.devices.sberbank.ru:9443/api/v2/oauth"
	tokenMethod = "POST"
	scope       = "scope=GIGACHAT_API_PERS"
)

// getConfig получает переменные из файла конфигураци
func getConfig() (string, string, string, int, error) {
	viper.SetConfigFile("config.env")

	if err := viper.ReadInConfig(); err != nil {
		err2 := fmt.Errorf("logs not read")
		return "", "", "", 0, err2
	}

	token := viper.Get("AUTH_KEY")
	token2 := fmt.Sprintf("%v", token)
	ip := viper.Get("IP")
	ip2 := fmt.Sprintf("%v", ip)
	port := viper.Get("PORT")
	port2 := fmt.Sprintf("%v", port)
	timeout := viper.Get("TIMEOUT")
	timeout2 := fmt.Sprintf("%v", timeout)
	timeout3, err := strconv.Atoi(timeout2)
	if err != nil {
		return "", "", "", 0, err
	}
	return token2, ip2, port2, timeout3, nil
}

// getToken получает acessToken на основе нашего ключа
func getToken(authKey string) (string, error) {
	//готовим данные для запроса
	url := tokenUrl
	method := tokenMethod
	payload := strings.NewReader(scope)
	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)
	if err != nil {

		return "", err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("RqUID", uuid.New().String())
	req.Header.Add("Authorization", "Basic "+authKey)
	// делаем сам запрос на сревера Сбербанка для получения токена
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	//разбираем полученные данные
	var tok Tok
	err = json.Unmarshal(body, &tok)
	if err != nil {
		return "", err
	}
	return tok.AcessToken, nil
}

// описание структуры токена
type Tok struct {
	AcessToken string `json:"access_token" db:"access_token"`
	ExpiresAt  int64  `json:"expires_at" db:"expires_at"`
}

package main

import (
	"fmt"
	"log"
	"time"

	"github.com/Rhymen/go-whatsapp"
)

var userData = make(map[string]time.Time)

func main() {
	// Подключение к WhatsApp
	wac, err := whatsapp.NewConn(5 * time.Second)
	if err != nil {
		log.Fatalf("Ошибка при создании соединения: %v", err)
	}

	// Авторизация в WhatsApp
	// Необходимо отсканировать QR-код, который появится в терминале
	qr := make(chan string)
	go func() {
		for msg := range qr {
			fmt.Println(msg)
		}
	}()
	wac.SetClient(qr)

	// Ожидание сканирования QR-кода
	<-wac.Wait
	log.Println("Вход выполнен!")

	// Начало прослушивания сообщений
	for {
		msg, err := wac.Read()
		if err != nil {
			log.Println("Ошибка чтения сообщения:", err)
			continue
		}
		handleMessage(msg, wac)
	}
}

func handleMessage(msg whatsapp.Message, wac *whatsapp.Conn) {
	if msg.Info.IsGroup() {
		return // Игнорируем групповые сообщения для простоты
	}

	if msg.Type == whatsapp.Text {
		switch msg.Text {
		case "!start":
			sendMessage(wac, msg.From, "Привет! Установите время для ежедневного опроса. Введите время в формате ЧЧ:ММ.")
		default:
			setTime(msg, wac)
		}
	}
}

func setTime(msg whatsapp.Message, wac *whatsapp.Conn) {
	userID := msg.From
	timeStr := msg.Text

	parsedTime, err := time.Parse("15:04", timeStr)
	if err != nil {
		sendMessage(wac, userID, "Неверный формат времени. Попробуйте снова.")
		return
	}

	userData[userID] = parsedTime
	sendMessage(wac, userID, fmt.Sprintf("Вы установили время опроса на %s.", timeStr))

	schedulePolling(userID, parsedTime, wac)
}

func schedulePolling(userID string, timeObj time.Time, wac *whatsapp.Conn) {
	now := time.Now()
	todayPoll := time.Date(now.Year(), now.Month(), now.Day(), timeObj.Hour(), timeObj.Minute(), 0, 0, now.Location())

	var nextPoll time.Time
	if now.Before(todayPoll) {
		nextPoll = todayPoll
	} else {
		nextPoll = todayPoll.Add(24 * time.Hour)
	}

	delay := nextPoll.Sub(now)

	log.Printf("Запланирован опрос для пользователя %s на %s (через %d секунд)", userID, nextPoll, int(delay.Seconds()))
	time.AfterFunc(delay, func() {
		pollUser(userID, wac)
	})
}

func pollUser(userID string, wac *whatsapp.Conn) {
	log.Printf("Отправка опроса пользователю %s", userID)
	sendMessage(wac, userID, "Ты поставил(а) инсулин? Ответь 'Да' или 'Нет'.")
}

func sendMessage(wac *whatsapp.Conn, to, text string) {
	msg := whatsapp.TextMessage{
		Info: whatsapp.MessageInfo{
			RemoteJid: to,
		},
		Text: text,
	}
	_, err := wac.Send(msg)
	if err != nil {
		log.Printf("Ошибка отправки сообщения пользователю %s: %v", to, err)
	}
}

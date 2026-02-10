package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/joho/godotenv"
)

// Config хранит конфигурацию из переменных окружения
type Config struct {
	Authorization string
	OrgID         string
}

// PageResponse структура для ответа API
type PageResponse struct {
	ID       int    `json:"id"`
	Slug     string `json:"slug"`
	Title    string `json:"title"`
	Content  string `json:"content"`
	PageType string `json:"page_type"`
}

func init() {
	// loads values from .env into the system
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
}

func main() {
	showWelcomeMessage()

	// Загружаем конфигурацию из переменных окружения
	config := loadConfig()
	if config.Authorization == "" {
		fmt.Println("Внимание: переменная окружения API_AUTH_TOKEN не установлена")
		fmt.Println("Для API запросов будет использоваться анонимный доступ")
	}
	if config.OrgID == "" {
		fmt.Println("Внимание: переменная окружения API_ORG_ID не установлена")
		fmt.Println("Для некоторых API запросов может потребоваться этот заголовок")
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		url := getInput(reader, "Введите URL для парсинга (или 'exit' для выхода): ")

		if shouldExit(url) {
			fmt.Println("Выход из программы...")
			break
		}

		if isHelpCommand(url) {
			showHelp()
			continue
		}

		validatedURL := validateURL(url)
		if validatedURL == "" {
			continue
		}

		parseURL(validatedURL, config)

		fmt.Println("\n" + strings.Repeat("-", 50) + "\n")
	}

	fmt.Println("Программа завершена. До свидания!")
}

func loadConfig() Config {
	apiAuthToken, existAuth := os.LookupEnv("API_AUTH_TOKEN")
	apiOrgId, existOrg := os.LookupEnv("API_ORG_ID")

	if !existAuth || !existOrg {
		panic("variables not finded!")
	}

	return Config{
		Authorization: apiAuthToken,
		OrgID:         apiOrgId,
	}
}

func showWelcomeMessage() {
	fmt.Println("=== ПАРСЕР API И ВЕБ-СТРАНИЦ ===")
	fmt.Println("Поддерживает API Wiki и обычные веб-страницы")
	fmt.Println("Требуемые переменные окружения:")
	fmt.Println("  API_AUTH_TOKEN - токен авторизации (Bearer token)")
	fmt.Println("  API_ORG_ID     - идентификатор организации")
	fmt.Println()
	fmt.Println("Доступные команды:")
	fmt.Println("  exit, quit - выход из программы")
	fmt.Println("  help, ?    - справка")
	fmt.Println(strings.Repeat("=", 50))
}

func getInput(reader *bufio.Reader, prompt string) string {
	fmt.Print(prompt)
	input, err := reader.ReadString('\n')
	if err != nil {
		log.Println("Ошибка чтения ввода:", err)
		return ""
	}
	return strings.TrimSpace(input)
}

func shouldExit(input string) bool {
	exitCommands := []string{"exit", "quit", "q", "выход"}
	inputLower := strings.ToLower(input)

	for _, cmd := range exitCommands {
		if inputLower == cmd {
			return true
		}
	}
	return false
}

func isHelpCommand(input string) bool {
	helpCommands := []string{"help", "?", "справка", "помощь"}
	inputLower := strings.ToLower(input)

	for _, cmd := range helpCommands {
		if inputLower == cmd {
			return true
		}
	}
	return false
}

func showHelp() {
	fmt.Println("\n=== СПРАВКА ===")
	fmt.Println("Как использовать парсер:")
	fmt.Println("1. Введите URL API или обычной страницы")
	fmt.Println("2. Для API URL должны начинаться с https://")
	fmt.Println("3. Для обычных сайтов можно вводить без протокола")
	fmt.Println("4. Переменные окружения загружаются автоматически")
	fmt.Println("5. Для выхода введите: exit, quit, q")
	fmt.Println("6. Для справки: help, ?")
	fmt.Println("\nПримеры API URL:")
	fmt.Println("  https://api.wiki.yandex.net/v1/pages?slug=...")
	fmt.Println("  https://api.example.com/data")
	fmt.Println("\nПримеры обычных URL:")
	fmt.Println("  google.com")
	fmt.Println("  https://github.com")
	fmt.Println(strings.Repeat("-", 50))
}

func validateURL(input string) string {
	if input == "" {
		fmt.Println("Ошибка: URL не может быть пустым")
		return ""
	}

	// Для API URL всегда требуется HTTPS
	if strings.Contains(input, "api.") && !strings.HasPrefix(input, "http") {
		fmt.Println("API URL требует протокол HTTPS")
		input = "https://" + input
		fmt.Println("Используем URL:", input)
		return input
	}

	// Для обычных URL спрашиваем протокол
	if !strings.HasPrefix(input, "http://") && !strings.HasPrefix(input, "https://") {
		fmt.Print("Протокол не указан. Использовать https://? (y/n): ")

		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))

		if answer == "y" || answer == "yes" || answer == "да" {
			input = "https://" + input
			fmt.Println("Используем URL:", input)
		} else {
			fmt.Println("Используйте полный URL с протоколом (https://...)")
			return ""
		}
	}

	if !strings.Contains(input, ".") {
		fmt.Println("Ошибка: URL должен содержать доменное имя")
		return ""
	}

	return input
}

func parseURL(url string, config Config) {
	fmt.Printf("\n🔍 Парсим: %s\n", url)
	fmt.Printf("⏰ Время: %s\n", time.Now().Format("15:04:05"))

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println("Ошибка создания запроса:", err)
		return
	}

	// Устанавливаем заголовки из конфигурации
	if config.Authorization != "" {
		req.Header.Set("Authorization", "OAuth " + config.Authorization)
		fmt.Println("✅ Используется Authorization заголовок")
	}

	if config.OrgID != "" {
		req.Header.Set("X-Org-Id", config.OrgID)
		fmt.Println("✅ Используется X-Org-Id заголовок")
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; MyParser/1.0)")
	req.Header.Set("Accept", "application/json, text/html, */*")

	startTime := time.Now()
	resp, err := client.Do(req)
	requestTime := time.Since(startTime)

	if err != nil {
		log.Println("❌ Ошибка HTTP запроса:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("📊 Статус: %d %s\n", resp.StatusCode, resp.Status)
	fmt.Printf("⏱️  Время выполнения запроса: %v\n", requestTime)
	fmt.Printf("📝 Content-Type: %s\n", resp.Header.Get("Content-Type"))
	fmt.Printf("📦 Content-Length: %d байт\n", resp.ContentLength)

	// Читаем весь ответ в буфер для многократного использования
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("❌ Ошибка чтения ответа:", err)
		return
	}

	// Определяем тип контента
	contentType := resp.Header.Get("Content-Type")

	if strings.Contains(contentType, "application/json") {
		parseJSONResponse(bodyBytes)
	} else if strings.Contains(contentType, "text/html") {
		parseHTMLResponse(bodyBytes, url)
	} else {
		parseGenericResponse(bodyBytes, contentType)
	}
}

func parseJSONResponse(body []byte) {
	fmt.Println("\n📋 Получен JSON ответ:")
	fmt.Println(strings.Repeat("=", 60))

	// Пробуем декодировать как PageResponse
	var page PageResponse
	if err := json.Unmarshal(body, &page); err == nil && page.ID != 0 {
		// Успешно распарсили как PageResponse
		displayPageResponse(page)
		return
	}

	// Пробуем как generic JSON
	displayGenericJSON(body)
}

func displayPageResponse(page PageResponse) {
	fmt.Printf("🆔 ID: %d\n", page.ID)
	fmt.Printf("🔗 Slug: %s\n", page.Slug)
	fmt.Printf("📝 Заголовок: %s\n", page.Title)
	fmt.Printf("📄 Тип страницы: %s\n", page.PageType)

	if page.Content != "" {
		fmt.Println("\n📖 Содержимое:")
		fmt.Println(strings.Repeat("-", 60))
		displayContent(page.Content)
	}
}

func displayContent(content string) {
	// Очищаем Markdown разметку для лучшего отображения
	content = strings.ReplaceAll(content, "**", "")
	content = strings.ReplaceAll(content, "#", "")
	content = strings.ReplaceAll(content, "&nbsp;", " ")

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			fmt.Printf("%3d: %s\n", i+1, line)
		}
	}
}

func displayGenericJSON(body []byte) {
	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		fmt.Println("❌ Ошибка парсинга JSON:", err)
		// Выводим сырой текст
		fmt.Println("\n📄 Сырой ответ:")
		fmt.Println(strings.Repeat("-", 60))
		fmt.Println(string(body))
		return
	}

	// Форматируем и выводим JSON
	formatted, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Println("❌ Ошибка форматирования JSON:", err)
		fmt.Println(string(body))
		return
	}

	// Ограничиваем вывод для больших JSON
	output := string(formatted)
	if len(output) > 2000 {
		fmt.Println("📄 JSON (первые 2000 символов):")
		output = output[:2000] + "\n... [вывод сокращен]"
	} else {
		fmt.Println("📄 JSON:")
	}
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println(output)

	// Если это объект, показываем ключи
	if obj, ok := data.(map[string]interface{}); ok {
		fmt.Println("\n🔑 Доступные поля:")
		for key := range obj {
			fmt.Printf("  • %s\n", key)
		}
	}
}

func parseHTMLResponse(body []byte, baseURL string) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		log.Println("❌ Ошибка парсинга HTML:", err)
		return
	}

	fmt.Println("\n🌐 HTML страница:")
	fmt.Println(strings.Repeat("=", 60))
	extractAndShowInfo(doc, baseURL)
}

func parseGenericResponse(body []byte, contentType string) {
	fmt.Printf("\n⚠️  Неизвестный тип контента: %s\n", contentType)
	fmt.Println(strings.Repeat("-", 60))

	// Ограничиваем вывод
	content := string(body)
	contentLength := len(content)

	if contentLength > 1000 {
		fmt.Printf("📄 Предпросмотр (первые 1000 из %d символов):\n", contentLength)
		fmt.Println(strings.Repeat("-", 40))
		fmt.Println(content[:1000])
		fmt.Println("\n... [вывод сокращен]")
	} else {
		fmt.Println("📄 Содержимое:")
		fmt.Println(strings.Repeat("-", 40))
		fmt.Println(content)
	}
}

func extractAndShowInfo(doc *goquery.Document, baseURL string) {
	title := strings.TrimSpace(doc.Find("title").First().Text())
	if title == "" {
		title = "(не найден)"
	}
	fmt.Printf("📄 Заголовок: %s\n", title)

	description := ""
	doc.Find("meta[name='description']").Each(func(i int, s *goquery.Selection) {
		if desc, exists := s.Attr("content"); exists && description == "" {
			description = strings.TrimSpace(desc)
		}
	})
	if description != "" {
		fmt.Printf("📝 Описание: %s\n", truncateText(description, 120))
	}

	fmt.Println("\n🔗 Ссылки на странице (первые 10):")
	fmt.Println(strings.Repeat("-", 60))

	linkCount := 0
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		if linkCount >= 10 {
			return
		}

		text := strings.TrimSpace(s.Text())
		href, exists := s.Attr("href")

		if !exists || len(text) > 100 {
			return
		}

		if strings.HasPrefix(href, "#") ||
			strings.HasPrefix(href, "javascript:") ||
			strings.HasPrefix(href, "mailto:") {
			return
		}

		text = cleanLinkText(text)
		if text == "" {
			text = "[без текста]"
		}

		fullURL := makeAbsoluteURL(href, baseURL)

		displayURL := fullURL
		if len(displayURL) > 50 {
			displayURL = displayURL[:47] + "..."
		}

		fmt.Printf("%2d. %s\n", linkCount+1, text)
		fmt.Printf("    %s\n", displayURL)

		linkCount++
	})

	if linkCount == 0 {
		fmt.Println("Ссылки не найдены")
	}

	// Статистика
	fmt.Println("\n📊 Статистика:")
	h1Count := doc.Find("h1").Length()
	h2Count := doc.Find("h2").Length()
	paragraphs := doc.Find("p").Length()
	images := doc.Find("img").Length()
	links := doc.Find("a").Length()

	fmt.Printf("  • Заголовки H1: %d\n", h1Count)
	fmt.Printf("  • Заголовки H2: %d\n", h2Count)
	fmt.Printf("  • Параграфы: %d\n", paragraphs)
	fmt.Printf("  • Изображения: %d\n", images)
	fmt.Printf("  • Всего ссылок: %d\n", links)
}

func truncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	return text[:maxLength] + "..."
}

func cleanLinkText(text string) string {
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\t", " ")

	for strings.Contains(text, "  ") {
		text = strings.ReplaceAll(text, "  ", " ")
	}

	return text
}

func makeAbsoluteURL(href, baseURL string) string {
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}

	if strings.HasPrefix(href, "/") {
		base := baseURL
		// Убираем путь из baseURL
		if strings.HasPrefix(base, "https://") {
			parts := strings.SplitN(base[8:], "/", 2)
			if len(parts) > 1 {
				return "https://" + parts[0] + href
			}
			return "https://" + base[8:] + href
		} else if strings.HasPrefix(base, "http://") {
			parts := strings.SplitN(base[7:], "/", 2)
			if len(parts) > 1 {
				return "http://" + parts[0] + href
			}
			return "http://" + base[7:] + href
		}
	}

	// Относительные URL
	if strings.HasSuffix(baseURL, "/") {
		return baseURL + href
	}

	lastSlash := strings.LastIndex(baseURL, "/")
	if lastSlash >= 8 { // После протокола (https:// или http://)
		return baseURL[:lastSlash+1] + href
	}

	return baseURL + "/" + href
}

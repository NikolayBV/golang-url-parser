package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func main() {
	showWelcomeMessage()

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

		parseURL(validatedURL)

		fmt.Println("\n" + strings.Repeat("-", 50) + "\n")
	}

	fmt.Println("Программа завершена. До свидания!")
}

func showWelcomeMessage() {
	fmt.Println("=== ПАРСЕР ВЕБ-СТРАНИЦ ===")
	fmt.Println("Введите URL для парсинга")
	fmt.Println("Доступные команды:")
	fmt.Println("  exit, quit - выход из программы")
	fmt.Println("  help, ?    - справка")
	fmt.Println(strings.Repeat("=", 30))
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
	fmt.Println("1. Введите любой URL (например: https://example.com)")
	fmt.Println("2. Программа покажет заголовок страницы и ссылки")
	fmt.Println("3. Можно вводить URL без https:// - он добавится автоматически")
	fmt.Println("4. Для выхода введите: exit, quit, q")
	fmt.Println("5. Для повторного показа справки: help, ?")
	fmt.Println("\nПримеры:")
	fmt.Println("  google.com")
	fmt.Println("  https://github.com")
	fmt.Println("  httpbin.org/html")
	fmt.Println(strings.Repeat("-", 30))
}

func validateURL(input string) string {
	if input == "" {
		fmt.Println("Ошибка: URL не может быть пустым")
		return ""
	}

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

func parseURL(url string) {
	fmt.Printf("\nПарсим: %s\n", url)
	fmt.Printf("Время: %s\n", time.Now().Format("15:04:05"))

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println("Ошибка создания запроса:", err)
		return
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; MyParser/1.0)")

	startTime := time.Now()
	resp, err := client.Do(req)
	requestTime := time.Since(startTime)

	if err != nil {
		log.Println("Ошибка HTTP запроса:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Статус: %d %s\n", resp.StatusCode, resp.Status)
	fmt.Printf("Время выполнения запроса: %v\n", requestTime)
	fmt.Printf("Размер страницы: ~%d байт\n", resp.ContentLength)

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Println("Ошибка парсинга HTML:", err)
		return
	}

	extractAndShowInfo(doc, url)
}

func extractAndShowInfo(doc *goquery.Document, baseURL string) {
	title := strings.TrimSpace(doc.Find("title").First().Text())
	if title == "" {
		title = "(не найден)"
	}
	fmt.Printf("\n📄 Заголовок: %s\n\n", title)

	description := ""
	doc.Find("meta[name='description']").Each(func(i int, s *goquery.Selection) {
		if desc, exists := s.Attr("content"); exists && description == "" {
			description = strings.TrimSpace(desc)
		}
	})
	if description != "" {
		fmt.Printf("📝 Описание: %s\n\n", truncateText(description, 100))
	}

	h1Count := doc.Find("h1").Length()
	h2Count := doc.Find("h2").Length()
	fmt.Printf("📊 Структура: H1=%d, H2=%d\n\n", h1Count, h2Count)

	fmt.Println("🔗 Ссылки на странице (первые 15):")
	fmt.Println(strings.Repeat("-", 50))

	linkCount := 0
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		if linkCount >= 15 {
			return
		}

		text := strings.TrimSpace(s.Text())
		href, exists := s.Attr("href")

		if !exists || text == "" || len(text) > 100 {
			return
		}

		if strings.HasPrefix(href, "#") ||
			strings.HasPrefix(href, "javascript:") ||
			strings.HasPrefix(href, "mailto:") {
			return
		}

		text = cleanLinkText(text)
		if text == "" {
			return
		}

		fullURL := makeAbsoluteURL(href, baseURL)

		displayURL := fullURL
		if len(displayURL) > 60 {
			displayURL = displayURL[:57] + "..."
		}

		fmt.Printf("%2d. %s\n", linkCount+1, text)
		fmt.Printf("    %s\n", displayURL)

		linkCount++
	})

	if linkCount == 0 {
		fmt.Println("Ссылки не найдены")
	}

	fmt.Println(strings.Repeat("-", 50))
	fmt.Printf("Всего найдено ссылок: %d\n", doc.Find("a").Length())

	paragraphs := doc.Find("p").Length()
	images := doc.Find("img").Length()
	fmt.Printf("\n📈 Статистика: параграфов=%d, изображений=%d\n", paragraphs, images)
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
		if strings.HasPrefix(baseURL, "https://") {
			domain := strings.TrimPrefix(baseURL, "https://")
			if idx := strings.Index(domain, "/"); idx != -1 {
				domain = domain[:idx]
			}
			return "https://" + domain + href
		} else if strings.HasPrefix(baseURL, "http://") {
			domain := strings.TrimPrefix(baseURL, "http://")
			if idx := strings.Index(domain, "/"); idx != -1 {
				domain = domain[:idx]
			}
			return "http://" + domain + href
		}
	}

	if strings.HasSuffix(baseURL, "/") {
		return baseURL + href
	}

	lastSlash := strings.LastIndex(baseURL, "/")
	if lastSlash > 7 { // После протокола (https://)
		return baseURL[:lastSlash+1] + href
	}

	return baseURL + "/" + href
}

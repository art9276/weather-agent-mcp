package weather

import (
	"maps"
	"strings"
)

// mergeMaps объединяет две карты, сохраняя все ключи из обеих.
func mergeMaps(base, overlay map[string]any) map[string]any {
	// Создаём новую карту для результата
	result := make(map[string]any)

	// Копируем все ключи из базовой карты используя стандартную функцию
	maps.Copy(result, base)

	// Копируем все ключи из overlay (с возможной перезаписью)
	maps.Copy(result, overlay)

	return result
}

// IsWeatherQuery определяет, является ли текстовый запрос запросом о погоде.
func IsWeatherQuery(input string) bool {
	// Приводим к нижнему регистру для регистронезависимого поиска
	lower := strings.ToLower(input)

	// Список ключевых слов, указывающих на запрос о погоде
	// Включает русские и английские слова
	keywords := []string{
		"погода", "прогноз", "weather", "forecast",
		"дождь", "снег", "солнце", "температура",
	}

	// Проверяем наличие любого ключевого слова в запросе
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}

	return false
}

// extractJSON извлекает JSON-объект из строки ответа LLM.
// LLM может возвращать JSON в markdown-разметке (```json ... ```).
// Функция удаляет разметку и возвращает только JSON-объект.
func extractJSON(s string) string {
	// Удаляем markdown-разметку ```json ... ```
	if strings.Contains(s, "```json") {
		start := strings.Index(s, "```json")
		s = s[start+7:]
		end := strings.Index(s, "```")
		if end != -1 {
			s = s[:end]
		}
	} else if strings.Contains(s, "```") {
		// Обрабатываем случай с просто ```
		start := strings.Index(s, "```")
		s = s[start+3:]
		end := strings.Index(s, "```")
		if end != -1 {
			s = s[:end]
		}
	}

	// Находим границы JSON-объекта по первой { и последней }
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start == -1 || end == -1 || end <= start {
		// Если JSON не найден, возвращаем оригинальную строку
		return s
	}

	// Возвращаем подстроку от { до } включительно
	return s[start : end+1]
}

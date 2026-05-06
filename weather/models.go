package weather

// WeatherForecast представляет структуру ответа прогноза погоды.
// Содержит общую информацию о запросе и массив ежедневных прогнозов.
type WeatherForecast struct {
	City     string          `json:"city"`     // City — город, для которого получен прогноз
	Days     int             `json:"days"`     // Days — количество дней прогноза
	Forecast []DailyForecast `json:"forecast"` // Forecast — массив ежедневных прогнозов
}

// DailyForecast представляет прогноз погоды на один день.
// Содержит данные о температуре, условиях, влажности и ветре.
type DailyForecast struct {
	Date        string  `json:"date"`             // Date — дата прогноза в формате YYYY-MM-DD
	Temperature float64 `json:"temperature_c"`    // Temperature — средняя температура в градусах Цельсия
	Condition   string  `json:"condition"`        // Condition — погодные условия: Sunny, Cloudy, Rainy, Snowy и т.д.
	Humidity    int     `json:"humidity_percent"` // Humidity — влажность воздуха в процентах
	WindSpeed   float64 `json:"wind_speed_kmh"`   // WindSpeed — скорость ветра в км/ч
}

// weatherArgs представляет аргументы для запроса прогноза погоды.
// Используется для парсинга JSON-ответа от LLM.
type weatherArgs struct {
	City string `json:"city"` // City — название города
	Days int    `json:"days"` // Days — количество дней для прогноза
}

// ParsedWeatherArgs представляет распарсенные аргументы для внутреннего использования.
// Используется после валидации и установки значений по умолчанию.
type ParsedWeatherArgs struct {
	City string // City — название города
	Days int    // Days — количество дней для прогноза
}

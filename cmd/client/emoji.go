package main

import (
	"sort"
	"strings"
)

// EmojiMap - карта текстовых кодов в эмодзи
var EmojiMap = map[string]string{
	// Улыбки и эмоции
	":smile":          "😊",
	":laugh":          "😄",
	":joy":            "😂",
	":rofl":           "🤣",
	":wink":           "😉",
	":blush":          "😊",
	":heart_eyes":     "😍",
	":kiss":           "😘",
	":cool":           "😎",
	":thinking":       "🤔",
	":neutral":        "😐",
	":expressionless": "😑",
	":unamused":       "😒",
	":roll_eyes":      "🙄",
	":grimacing":      "😬",
	":lying":          "🤥",
	":relieved":       "😌",
	":sleepy":         "😪",
	":yawn":           "🥱",
	":drooling":       "🤤",
	":sleeping":       "😴",
	":mask":           "😷",
	":thermometer":    "🤒",
	":bandage":        "🤕",
	":nauseated":      "🤢",
	":vomiting":       "🤮",
	":sneeze":         "🤧",
	":hot":            "🥵",
	":cold":           "🥶",
	":dizzy":          "😵",
	":exploding":      "🤯",
	":cowboy":         "🤠",
	":partying":       "🥳",
	":disguised":      "🥸",
	":sunglasses":     "😎",
	":nerd":           "🤓",
	":monocle":        "🧐",
	":confused":       "😕",
	":worried":        "😟",
	":frowning":       "🙁",
	":sad":            "☹️",
	":cry":            "😢",
	":sob":            "😭",
	":scream":         "😱",
	":fearful":        "😨",
	":disappointed":   "😞",
	":persevere":      "😣",
	":triumph":        "😤",
	":angry":          "😠",
	":rage":           "😡",
	":cursing":        "🤬",
	":smiling_devil":  "😈",
	":devil":          "👿",
	":skull":          "💀",
	":poop":           "💩",
	":clown":          "🤡",
	":alien":          "👽",
	":robot":          "🤖",
	":ghost":          "👻",

	// Жесты
	":thumbsup":        "👍",
	":thumbsdown":      "👎",
	":ok":              "👌",
	":fist":            "✊",
	":punch":           "👊",
	":wave":            "👋",
	":raised_hands":    "🙌",
	":clap":            "👏",
	":pray":            "🙏",
	":flex":            "💪",
	":fingers_crossed": "🤞",
	":peace":           "✌️",
	":point_up":        "☝️",
	":point_down":      "👇",
	":point_left":      "👈",
	":point_right":     "👉",

	// Сердечки
	":heart":             "❤️",
	":orange_heart":      "🧡",
	":yellow_heart":      "💛",
	":green_heart":       "💚",
	":blue_heart":        "💙",
	":purple_heart":      "💜",
	":black_heart":       "🖤",
	":white_heart":       "🤍",
	":brown_heart":       "🤎",
	":broken_heart":      "💔",
	":sparkling_heart":   "💖",
	":growing_heart":     "💗",
	":beating_heart":     "💓",
	":revolving_heart":   "💞",
	":two_hearts":        "💕",
	":heart_decoration":  "💟",
	":heart_exclamation": "❣️",
	":heart_arrow":       "💘",
	":heart_ribbon":      "💝",

	// Животные
	":cat":       "🐱",
	":dog":       "🐶",
	":rabbit":    "🐰",
	":bear":      "🐻",
	":panda":     "🐼",
	":koala":     "🐨",
	":tiger":     "🐯",
	":lion":      "🦁",
	":cow":       "🐮",
	":pig":       "🐷",
	":frog":      "🐸",
	":monkey":    "🐵",
	":chicken":   "🐔",
	":penguin":   "🐧",
	":bird":      "🐦",
	":unicorn":   "🦄",
	":horse":     "🐴",
	":mouse":     "🐭",
	":hamster":   "🐹",
	":fox":       "🦊",
	":wolf":      "🐺",
	":octopus":   "🐙",
	":fish":      "🐟",
	":dolphin":   "🐬",
	":whale":     "🐳",
	":crab":      "🦀",
	":snail":     "🐌",
	":butterfly": "🦋",
	":bee":       "🐝",
	":ladybug":   "🐞",

	// Еда
	":pizza":      "🍕",
	":burger":     "🍔",
	":fries":      "🍟",
	":hotdog":     "🌭",
	":taco":       "🌮",
	":burrito":    "🌯",
	":sushi":      "🍣",
	":ramen":      "🍜",
	":spaghetti":  "🍝",
	":rice":       "🍚",
	":curry":      "🍛",
	":bento":      "🍱",
	":donut":      "🍩",
	":cake":       "🎂",
	":cookie":     "🍪",
	":chocolate":  "🍫",
	":candy":      "🍬",
	":icecream":   "🍦",
	":popcorn":    "🍿",
	":coffee":     "☕",
	":tea":        "🍵",
	":beer":       "🍺",
	":wine":       "🍷",
	":cocktail":   "🍸",
	":champagne":  "🍾",
	":watermelon": "🍉",
	":apple":      "🍎",
	":banana":     "🍌",
	":cherries":   "🍒",
	":strawberry": "🍓",
	":lemon":      "🍋",
	":avocado":    "🥑",
	":eggplant":   "🍆",
	":carrot":     "🥕",
	":corn":       "🌽",
	":hot_pepper": "🌶️",

	// Погода и природа
	":sun":        "☀️",
	":moon":       "🌙",
	":star":       "⭐",
	":rainbow":    "🌈",
	":cloud":      "☁️",
	":rain":       "🌧️",
	":snow":       "❄️",
	":lightning":  "⚡",
	":tornado":    "🌪️",
	":fire":       "🔥",
	":water":      "💧",
	":earth":      "🌍",
	":flower":     "🌸",
	":rose":       "🌹",
	":tree":       "🌳",
	":cactus":     "🌵",
	":mushroom":   "🍄",
	":leaf":       "🍃",
	":maple_leaf": "🍁",

	// Предметы
	":phone":      "📱",
	":computer":   "💻",
	":camera":     "📷",
	":tv":         "📺",
	":clock":      "🕐",
	":alarm":      "⏰",
	":hourglass":  "⌛",
	":money":      "💵",
	":gem":        "💎",
	":crown":      "👑",
	":ring":       "💍",
	":lipstick":   "💄",
	":book":       "📖",
	":pencil":     "✏️",
	":paperclip":  "📎",
	":scissors":   "✂️",
	":key":        "🔑",
	":lock":       "🔒",
	":hammer":     "🔨",
	":bulb":       "💡",
	":magnet":     "🧲",
	":pill":       "💊",
	":syringe":    "💉",
	":soap":       "🧼",
	":toothbrush": "🪥",

	// Символы
	":check":        "✅",
	":cross":        "❌",
	":question":     "❓",
	":exclamation":  "❗",
	":warning":      "⚠️",
	":recycle":      "♻️",
	":infinity":     "♾️",
	":peace_symbol": "☮️",
	":yin_yang":     "☯️",
	":atom":         "⚛️",
	":menorah":      "🕎",
	":cross_symbol": "✝️",
	":om":           "🕉️",
	":wheel":        "☸️",
	":star_david":   "✡️",

	// Транспорт
	":car":        "🚗",
	":bike":       "🚲",
	":plane":      "✈️",
	":rocket":     "🚀",
	":ship":       "🚢",
	":train":      "🚂",
	":ambulance":  "🚑",
	":police_car": "🚓",
	":fire_truck": "🚒",
	":taxi":       "🚕",
	":bus":        "🚌",
	":helicopter": "🚁",

	// Спорт
	":soccer":     "⚽",
	":basketball": "🏀",
	":football":   "🏈",
	":tennis":     "🎾",
	":golf":       "⛳",
	":trophy":     "🏆",
	":medal":      "🏅",
	":target":     "🎯",
	":dice":       "🎲",
	":chess":      "♟️",
	":joystick":   "🕹️",
	":pool":       "🎱",
}

// ReplaceEmojis заменяет текстовые коды на эмодзи
func ReplaceEmojis(text string) string {
	result := text

	// Заменяем коды на эмодзи
	for code, emoji := range EmojiMap {
		result = strings.ReplaceAll(result, code, emoji)
	}

	return result
}

func FindEmojiSuggestions(prefix string) []string {
	if !strings.HasPrefix(prefix, ":") {
		return nil
	}

	var suggestions []string
	prefix = strings.ToLower(prefix)

	for code := range EmojiMap {
		if strings.HasPrefix(strings.ToLower(code), prefix) {
			suggestions = append(suggestions, code)
		}
	}

	// Сортируем для стабильного порядка
	sort.Strings(suggestions)

	// Ограничим количество подсказок
	if len(suggestions) > 5 {
		suggestions = suggestions[:5]
	}

	return suggestions
}

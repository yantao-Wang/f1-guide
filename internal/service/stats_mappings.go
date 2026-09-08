package service

import "strings"

// 上游英文数据 → 站内中文口径的映射表。
// 数据来源：2026 赛季上游真实数据（2026-09 采集）。
// 未命中的键回退为原文，保证上游新增车队/车手/赛道时不阻塞上线；
// 上游新增车手故事内容时，记得同步维护 driverSlugs。

// constructorColors 车队主色（与 seeds/dev.sql 的口径保持一致）。
var constructorColors = map[string]string{
	"Mercedes":         "#27F4D2",
	"Ferrari":          "#DC0000",
	"McLaren":          "#FF8000",
	"Red Bull":         "#1E41FF",
	"RB F1 Team":       "#6692FF",
	"Alpine F1 Team":   "#FF87BC",
	"Haas F1 Team":     "#B6BABD",
	"Audi":             "#BB0A30",
	"Williams":         "#64C4FF",
	"Aston Martin":     "#006F62",
	"Cadillac F1 Team": "#00A1E0",
}

// fallbackColor 未知车队的兜底色（--text-muted 同值）。
const fallbackColor = "#6B6B6B"

func constructorColor(name string) string {
	if c, ok := constructorColors[name]; ok {
		return c
	}
	return fallbackColor
}

var constructorNamesZh = map[string]string{
	"Mercedes":         "梅赛德斯",
	"Ferrari":          "法拉利",
	"McLaren":          "迈凯伦",
	"Red Bull":         "红牛",
	"RB F1 Team":       "红牛二队",
	"Alpine F1 Team":   "阿尔派",
	"Haas F1 Team":     "哈斯",
	"Audi":             "奥迪",
	"Williams":         "威廉姆斯",
	"Aston Martin":     "阿斯顿·马丁",
	"Cadillac F1 Team": "凯迪拉克",
}

func constructorZh(name string) string {
	if zh, ok := constructorNamesZh[name]; ok {
		return zh
	}
	return name
}

var driverNamesZh = map[string]string{
	"antonelli":      "安东内利",
	"russell":        "拉塞尔",
	"hamilton":       "汉密尔顿",
	"norris":         "诺里斯",
	"leclerc":        "勒克莱尔",
	"max_verstappen": "维斯塔潘",
	"piastri":        "皮亚斯特里",
	"hadjar":         "哈贾尔",
	"lawson":         "劳森",
	"gasly":          "加斯利",
	"arvid_lindblad": "林德布拉德",
	"colapinto":      "科拉平托",
	"bearman":        "贝尔曼",
	"bortoleto":      "博尔托莱托",
	"hulkenberg":     "霍肯伯格",
	"sainz":          "塞恩斯",
	"albon":          "阿尔本",
	"ocon":           "奥康",
	"alonso":         "阿隆索",
	"tsunoda":        "角田裕毅",
	"stroll":         "斯托罗尔",
	"bottas":         "博塔斯",
	"perez":          "佩雷兹",
	"zhou":           "周冠宇",
}

func driverNameZh(id, given, family string) string {
	if zh, ok := driverNamesZh[id]; ok {
		return zh
	}
	return strings.TrimSpace(given + " " + family)
}

// driverSlugs 上游 driverId → 站内车手故事 slug。
// 内容流水线每新增一篇车手故事，在此补一条映射。
var driverSlugs = map[string]string{
	"max_verstappen": "max-verstappen",
	"hamilton":       "lewis-hamilton",
	"leclerc":        "charles-leclerc",
	"norris":         "lando-norris",
	"alonso":         "fernando-alonso",
	"sainz":          "carlos-sainz",
	"russell":        "george-russell",
	"piastri":        "oscar-piastri",
	"zhou":           "zhou-guanyu",
}

// driverSlug 优先映射站内 slug，否则规范化为连字符形式。
func driverSlug(id string) string {
	if slug, ok := driverSlugs[id]; ok {
		return slug
	}
	return strings.ReplaceAll(id, "_", "-")
}

var raceNamesZh = map[string]string{
	"Australian Grand Prix":          "澳大利亚大奖赛",
	"Chinese Grand Prix":             "中国大奖赛",
	"Japanese Grand Prix":            "日本大奖赛",
	"Miami Grand Prix":               "迈阿密大奖赛",
	"Canadian Grand Prix":            "加拿大大奖赛",
	"Monaco Grand Prix":              "摩纳哥大奖赛",
	"Barcelona Grand Prix":           "巴塞罗那大奖赛",
	"Austrian Grand Prix":            "奥地利大奖赛",
	"British Grand Prix":             "英国大奖赛",
	"Belgian Grand Prix":             "比利时大奖赛",
	"Hungarian Grand Prix":           "匈牙利大奖赛",
	"Dutch Grand Prix":               "荷兰大奖赛",
	"Italian Grand Prix":             "意大利大奖赛",
	"Spanish Grand Prix":             "西班牙大奖赛",
	"Azerbaijan Grand Prix":          "阿塞拜疆大奖赛",
	"Bahrain Grand Prix in Malaysia": "巴林大奖赛（马来西亚）",
	"Singapore Grand Prix":           "新加坡大奖赛",
	"United States Grand Prix":       "美国大奖赛",
	"Mexico City Grand Prix":         "墨西哥城大奖赛",
	"Brazilian Grand Prix":           "巴西大奖赛",
	"Las Vegas Grand Prix":           "拉斯维加斯大奖赛",
	"Qatar Grand Prix":               "卡塔尔大奖赛",
	"Abu Dhabi Grand Prix":           "阿布扎比大奖赛",
}

func grandPrixZh(name string) string {
	if zh, ok := raceNamesZh[name]; ok {
		return zh
	}
	return name
}

var circuitNamesZh = map[string]string{
	"Albert Park Grand Prix Circuit": "阿尔伯特公园赛道",
	"Shanghai International Circuit": "上海国际赛车场",
	"Suzuka Circuit":                 "铃鹿赛道",
	"Miami International Autodrome":  "迈阿密国际赛道",
	"Circuit Gilles Villeneuve":      "维伦纽夫赛道",
	"Circuit de Monaco":              "摩纳哥蒙特卡洛赛道",
	"Circuit de Barcelona-Catalunya": "巴塞罗那-加泰罗尼亚赛道",
	"Red Bull Ring":                  "红牛环赛道",
	"Silverstone Circuit":            "银石赛道",
	"Circuit de Spa-Francorchamps":   "斯帕赛道",
	"Hungaroring":                    "匈牙利赛道",
	"Circuit Park Zandvoort":         "赞德沃特赛道",
	"Autodromo Nazionale di Monza":   "蒙扎赛道",
	"Madring":                        "马德里 Madring 赛道",
	"Baku City Circuit":              "巴库城市赛道",
	"Sepang International Circuit":   "雪邦国际赛道",
	"Marina Bay Street Circuit":      "滨海湾街道赛道",
	"Circuit of the Americas":        "美洲赛道",
	"Autódromo Hermanos Rodríguez":   "罗德里格斯兄弟赛道",
	"Autódromo José Carlos Pace":     "英特拉格斯赛道",
	"Las Vegas Strip Street Circuit": "拉斯维加斯大道赛道",
	"Losail International Circuit":   "卢赛尔赛道",
	"Yas Marina Circuit":             "亚斯码头赛道",
}

func circuitZh(name string) string {
	if zh, ok := circuitNamesZh[name]; ok {
		return zh
	}
	return name
}

var countryNamesZh = map[string]string{
	"Australia":   "澳大利亚",
	"China":       "中国",
	"Japan":       "日本",
	"USA":         "美国",
	"Canada":      "加拿大",
	"Monaco":      "摩纳哥",
	"Spain":       "西班牙",
	"Austria":     "奥地利",
	"UK":          "英国",
	"Belgium":     "比利时",
	"Hungary":     "匈牙利",
	"Netherlands": "荷兰",
	"Italy":       "意大利",
	"Azerbaijan":  "阿塞拜疆",
	"Malaysia":    "马来西亚",
	"Singapore":   "新加坡",
	"Mexico":      "墨西哥",
	"Brazil":      "巴西",
	"Qatar":       "卡塔尔",
	"UAE":         "阿联酋",
}

func countryZh(name string) string {
	if zh, ok := countryNamesZh[name]; ok {
		return zh
	}
	return name
}

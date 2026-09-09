// 后台表单：解析、校验、与 domain 互转。
//
// 表单持有原始字符串值——校验失败重渲染时原样保留用户输入；
// 校验通过后由 toXxx 转换为 domain 对象。
package handler

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/yantao-Wang/f1-guide/internal/domain"
)

var (
	slugRe    = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	colorRe   = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
	jolpicaRe = regexp.MustCompile(`^[a-z0-9_]+$`)
	httpURLRe = regexp.MustCompile(`^https?://`)
	currentYr = time.Now().Year()
)

// parseTextLines 把 textarea 内容按行切分（丢弃空行与首尾空白）。
func parseTextLines(s string) []string {
	out := make([]string, 0)
	for _, line := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(line); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// joinLines 把字符串切片还原为 textarea 逐行文本。
func joinLines(items []string) string { return strings.Join(items, "\n") }

// requireField 必填校验。
func requireField(errors map[string]string, field, label, value string) {
	if strings.TrimSpace(value) == "" {
		errors[field] = label + "必填"
	}
}

// parseRangeField 解析整数范围字段（必填）。非法返回 0 并记录错误。
func parseRangeField(errors map[string]string, field, label, raw string, min, max int) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n < min || n > max {
		errors[field] = label + "需为 " + strconv.Itoa(min) + "-" + strconv.Itoa(max) + " 的整数"
		return 0
	}
	return n
}

// parseOptionalYear 解析选填年份（1950-当前年），空串返回 0。
func parseOptionalYear(errors map[string]string, field, label, raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	return parseRangeField(errors, field, label, raw, 1950, currentYr)
}

// validateSlug slug 格式校验。
func validateSlug(errors map[string]string, field, value string) {
	requireField(errors, field, "slug ", value)
	if v := strings.TrimSpace(value); v != "" && (len(v) > 100 || !slugRe.MatchString(v)) {
		errors[field] = "slug 仅允许小写字母、数字与连字符（如 zhou-guanyu）"
	}
}

// validateURL 选填 URL 校验（非空必须以 http/https 开头）。
func validateURL(errors map[string]string, field, label, value string) {
	if v := strings.TrimSpace(value); v != "" && !httpURLRe.MatchString(v) {
		errors[field] = label + "需以 http:// 或 https:// 开头"
	}
}

// --- 车手表单 ---

// driverForm 车手编辑表单（原始字符串值 + 校验错误）。
type driverForm struct {
	Slug, Name, Tagline, TeamName, TeamColor, Number, Championships, Country string
	JolpicaID, Story, PersonalityText, TriviaText, Quote                     string
	Featured                                                                 bool
	FeaturedRaceYear, FeaturedRaceGP                                         string
	RemoveImage                                                              bool
	Errors                                                                   map[string]string
}

// newDriverForm 从请求解析车手表单。
func newDriverForm(r *http.Request) driverForm {
	f := driverForm{Errors: map[string]string{}}
	f.Slug = r.PostForm.Get("slug")
	f.Name = r.PostForm.Get("name")
	f.Tagline = r.PostForm.Get("tagline")
	f.TeamName = r.PostForm.Get("team_name")
	f.TeamColor = r.PostForm.Get("team_color")
	f.Number = r.PostForm.Get("number")
	f.Championships = r.PostForm.Get("championships")
	f.Country = r.PostForm.Get("country")
	f.JolpicaID = strings.TrimSpace(r.PostForm.Get("jolpica_id"))
	f.Story = r.PostForm.Get("story")
	f.PersonalityText = r.PostForm.Get("personality")
	f.TriviaText = r.PostForm.Get("trivia")
	f.Quote = r.PostForm.Get("quote")
	f.Featured = r.PostForm.Get("featured") == "on"
	f.FeaturedRaceYear = r.PostForm.Get("featured_race_year")
	f.FeaturedRaceGP = r.PostForm.Get("featured_race_gp")
	f.RemoveImage = r.PostForm.Get("remove_image") == "on"
	return f
}

// validate 校验车手表单，错误写入 f.Errors。
func (f *driverForm) validate() {
	validateSlug(f.Errors, "slug", f.Slug)
	requireField(f.Errors, "name", "车手姓名", f.Name)
	requireField(f.Errors, "tagline", "一句话人设", f.Tagline)
	requireField(f.Errors, "team_name", "车队名", f.TeamName)
	requireField(f.Errors, "country", "国籍", f.Country)
	requireField(f.Errors, "story", "核心故事", f.Story)
	requireField(f.Errors, "quote", "名言金句", f.Quote)

	if v := strings.TrimSpace(f.TeamColor); v == "" {
		f.Errors["team_color"] = "车队主题色必填"
	} else if !colorRe.MatchString(v) {
		f.Errors["team_color"] = "颜色需为 #RRGGBB 格式"
	}
	parseRangeField(f.Errors, "number", "车号", f.Number, 1, 99)
	parseRangeField(f.Errors, "championships", "世界冠军数", f.Championships, 0, 20)
	parseOptionalYear(f.Errors, "featured_race_year", "推荐比赛年份", f.FeaturedRaceYear)

	if f.JolpicaID != "" && !jolpicaRe.MatchString(f.JolpicaID) {
		f.Errors["jolpica_id"] = "Jolpica ID 仅允许小写字母、数字与下划线（如 max_verstappen）"
	}
}

// toDriver 转换为 domain 对象（调用方需先通过 validate）。
func (f driverForm) toDriver() *domain.Driver {
	number, _ := strconv.Atoi(strings.TrimSpace(f.Number))
	championships, _ := strconv.Atoi(strings.TrimSpace(f.Championships))
	year, _ := strconv.Atoi(strings.TrimSpace(f.FeaturedRaceYear))

	var featuredRace *domain.FeaturedRace
	if year > 0 && strings.TrimSpace(f.FeaturedRaceGP) != "" {
		featuredRace = &domain.FeaturedRace{Year: year, GrandPrix: strings.TrimSpace(f.FeaturedRaceGP)}
	}

	return &domain.Driver{
		Slug:          strings.TrimSpace(f.Slug),
		Name:          strings.TrimSpace(f.Name),
		Tagline:       strings.TrimSpace(f.Tagline),
		Team:          domain.Team{Name: strings.TrimSpace(f.TeamName), Color: strings.TrimSpace(f.TeamColor)},
		Number:        number,
		Championships: championships,
		Country:       strings.TrimSpace(f.Country),
		Story:         strings.TrimSpace(f.Story),
		Personality:   parseTextLines(f.PersonalityText),
		Trivia:        parseTextLines(f.TriviaText),
		Quote:         strings.TrimSpace(f.Quote),
		Featured:      f.Featured,
		FeaturedRace:  featuredRace,
		JolpicaID:     f.JolpicaID,
	}
}

// driverFormFrom 用现有车手回填表单（编辑页）。
func driverFormFrom(d *domain.Driver) driverForm {
	f := driverForm{Errors: map[string]string{}}
	f.Slug = d.Slug
	f.Name = d.Name
	f.Tagline = d.Tagline
	f.TeamName = d.Team.Name
	f.TeamColor = d.Team.Color
	f.Number = strconv.Itoa(d.Number)
	f.Championships = strconv.Itoa(d.Championships)
	f.Country = d.Country
	f.JolpicaID = d.JolpicaID
	f.Story = d.Story
	f.PersonalityText = joinLines(d.Personality)
	f.TriviaText = joinLines(d.Trivia)
	f.Quote = d.Quote
	f.Featured = d.Featured
	if d.FeaturedRace != nil {
		f.FeaturedRaceYear = strconv.Itoa(d.FeaturedRace.Year)
		f.FeaturedRaceGP = d.FeaturedRace.GrandPrix
	}
	return f
}

// --- 赛道表单 ---

// trackForm 赛道编辑表单。
type trackForm struct {
	Slug, Name, Country, Tagline, Type, FirstGrandPrix, LengthKm, Laps string
	HighlightsText, CircuitMapURL                                      string
	RemoveImage                                                        bool
	Errors                                                             map[string]string
}

// newTrackForm 从请求解析赛道表单。
func newTrackForm(r *http.Request) trackForm {
	f := trackForm{Errors: map[string]string{}}
	f.Slug = r.PostForm.Get("slug")
	f.Name = r.PostForm.Get("name")
	f.Country = r.PostForm.Get("country")
	f.Tagline = r.PostForm.Get("tagline")
	f.Type = r.PostForm.Get("type")
	f.FirstGrandPrix = r.PostForm.Get("first_grand_prix")
	f.LengthKm = r.PostForm.Get("length_km")
	f.Laps = r.PostForm.Get("laps")
	f.HighlightsText = r.PostForm.Get("highlights")
	f.CircuitMapURL = strings.TrimSpace(r.PostForm.Get("circuit_map_url"))
	f.RemoveImage = r.PostForm.Get("remove_image") == "on"
	return f
}

// validate 校验赛道表单。
func (f *trackForm) validate() {
	validateSlug(f.Errors, "slug", f.Slug)
	requireField(f.Errors, "name", "赛道名", f.Name)
	requireField(f.Errors, "country", "国家", f.Country)
	requireField(f.Errors, "tagline", "一句话介绍", f.Tagline)
	if f.Type != domain.TrackTypePermanent && f.Type != domain.TrackTypeStreet {
		f.Errors["type"] = "赛道类型非法"
	}
	parseRangeField(f.Errors, "first_grand_prix", "首次办赛年份", f.FirstGrandPrix, 1950, currentYr)
	parseRangeField(f.Errors, "laps", "圈数", f.Laps, 1, 120)

	if v := strings.TrimSpace(f.LengthKm); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err != nil || n <= 0 || n > 30 {
			f.Errors["length_km"] = "赛道长度需为 0-30 之间的数字"
		}
	} else {
		f.Errors["length_km"] = "赛道长度必填"
	}
	validateURL(f.Errors, "circuit_map_url", "赛道图链接", f.CircuitMapURL)
}

// toTrack 转换为 domain 对象。
func (f trackForm) toTrack() *domain.Track {
	firstGP, _ := strconv.Atoi(strings.TrimSpace(f.FirstGrandPrix))
	length, _ := strconv.ParseFloat(strings.TrimSpace(f.LengthKm), 64)
	laps, _ := strconv.Atoi(strings.TrimSpace(f.Laps))

	return &domain.Track{
		TrackSummary: domain.TrackSummary{
			Slug:    strings.TrimSpace(f.Slug),
			Name:    strings.TrimSpace(f.Name),
			Country: strings.TrimSpace(f.Country),
			Tagline: strings.TrimSpace(f.Tagline),
			Type:    f.Type,
		},
		FirstGrandPrix: firstGP,
		LengthKm:       length,
		Laps:           laps,
		Highlights:     parseTextLines(f.HighlightsText),
		CircuitMapURL:  f.CircuitMapURL,
	}
}

// trackFormFrom 用现有赛道回填表单。
func trackFormFrom(t *domain.Track) trackForm {
	f := trackForm{Errors: map[string]string{}}
	f.Slug = t.Slug
	f.Name = t.Name
	f.Country = t.Country
	f.Tagline = t.Tagline
	f.Type = t.Type
	f.FirstGrandPrix = strconv.Itoa(t.FirstGrandPrix)
	f.LengthKm = strconv.FormatFloat(t.LengthKm, 'f', 3, 64)
	f.Laps = strconv.Itoa(t.Laps)
	f.HighlightsText = joinLines(t.Highlights)
	f.CircuitMapURL = t.CircuitMapURL
	return f
}

// --- 名场面表单 ---

// momentForm 名场面编辑表单。
type momentForm struct {
	Slug, Title, Year, GrandPrix, Type, Background, WhyClassic, VideoURL, TrackSlug string
	DriverSlugs                                                                     []string
	Errors                                                                          map[string]string
}

// newMomentForm 从请求解析名场面表单（driver_slugs 多选 checkbox）。
func newMomentForm(r *http.Request) momentForm {
	f := momentForm{Errors: map[string]string{}}
	f.Slug = r.PostForm.Get("slug")
	f.Title = r.PostForm.Get("title")
	f.Year = r.PostForm.Get("year")
	f.GrandPrix = r.PostForm.Get("grand_prix")
	f.Type = r.PostForm.Get("type")
	f.Background = r.PostForm.Get("background")
	f.WhyClassic = r.PostForm.Get("why_classic")
	f.VideoURL = strings.TrimSpace(r.PostForm.Get("video_url"))
	f.TrackSlug = r.PostForm.Get("track_slug")
	if values, ok := r.PostForm["driver_slugs"]; ok {
		f.DriverSlugs = values
	}
	return f
}

// validate 校验名场面表单。
func (f *momentForm) validate() {
	validateSlug(f.Errors, "slug", f.Slug)
	requireField(f.Errors, "title", "标题", f.Title)
	requireField(f.Errors, "grand_prix", "比赛名称", f.GrandPrix)
	requireField(f.Errors, "background", "现场背景", f.Background)
	requireField(f.Errors, "why_classic", "经典理由", f.WhyClassic)
	parseRangeField(f.Errors, "year", "年份", f.Year, 1950, currentYr)
	switch f.Type {
	case domain.MomentTypeChampionship, domain.MomentTypeOvertake, domain.MomentTypeSafety, domain.MomentTypeRain:
	default:
		f.Errors["type"] = "名场面类型非法"
	}
	validateURL(f.Errors, "video_url", "视频链接", f.VideoURL)
}

// toMoment 转换为 domain 对象。
func (f momentForm) toMoment() *domain.Moment {
	year, _ := strconv.Atoi(strings.TrimSpace(f.Year))
	return &domain.Moment{
		MomentSummary: domain.MomentSummary{
			Slug:      strings.TrimSpace(f.Slug),
			Title:     strings.TrimSpace(f.Title),
			Year:      year,
			GrandPrix: strings.TrimSpace(f.GrandPrix),
			Type:      f.Type,
		},
		Background:     strings.TrimSpace(f.Background),
		WhyClassic:     strings.TrimSpace(f.WhyClassic),
		VideoURL:       f.VideoURL,
		RelatedDrivers: f.DriverSlugs,
		RelatedTrack:   f.TrackSlug,
	}
}

// momentFormFrom 用现有名场面回填表单。
func momentFormFrom(m *domain.Moment) momentForm {
	f := momentForm{Errors: map[string]string{}}
	f.Slug = m.Slug
	f.Title = m.Title
	f.Year = strconv.Itoa(m.Year)
	f.GrandPrix = m.GrandPrix
	f.Type = m.Type
	f.Background = m.Background
	f.WhyClassic = m.WhyClassic
	f.VideoURL = m.VideoURL
	f.TrackSlug = m.RelatedTrack
	f.DriverSlugs = m.RelatedDrivers
	return f
}

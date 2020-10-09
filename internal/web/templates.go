package web

import (
	"crypto/md5" // nolint:gosec,gci
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"html/template"
	"io"
	"kaepora/internal/back"
	"kaepora/internal/generator/oot"
	"kaepora/internal/global"
	"kaepora/internal/util"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/russross/blackfriday/v2"
)

func (s *Server) loadTemplates(baseDir string) (map[string]*template.Template, error) {
	layouts, err := filepath.Glob(filepath.Join(baseDir, "templates/layouts/*.html"))
	if err != nil {
		return nil, err
	}

	includes, err := filepath.Glob(filepath.Join(baseDir, "templates/includes/*.html"))
	if err != nil {
		return nil, err
	}

	ret := make(map[string]*template.Template, len(layouts))
	for _, layout := range layouts {
		tpl, err := template.New("").
			Funcs(s.getTemplateFuncMap(baseDir)).
			ParseFiles(append(includes, layout)...)
		if err != nil {
			return nil, err
		}

		key := strings.TrimPrefix(layout, filepath.Join(baseDir, "templates/layouts")+"/")
		ret[key] = tpl
	}

	return ret, nil
}

func (s *Server) getTemplateFuncMap(baseDir string) template.FuncMap {
	return template.FuncMap{
		"t": func(locale string, str string, args ...interface{}) string {
			return s.locales[locale].Get(str, args...)
		},

		"tn": func(locale, singular, plural string, count int, args ...interface{}) string {
			return s.locales[locale].GetN(singular, plural, count, args...)
		},

		"uri": func(locale string, parts ...string) string {
			if len(parts) == 0 {
				return "/" + locale
			}

			return "/" + locale + "/" + filepath.Join(parts...)
		},

		"tmd": func(locale, str string, args ...interface{}) template.HTML {
			return template.HTML(blackfriday.Run( // nolint:gosec
				[]byte(s.locales[locale].Get(str, args...)),
			))
		},

		"ignoreZero": func(i int) string {
			if i == 0 {
				return ""
			}

			return strconv.Itoa(i)
		},

		"matchEntryStatus":      s.tplMatchEntryStatus,
		"matchSeedURL":          s.tplMatchSeedURL,
		"matchSessionStatusTag": s.tplMatchSessionStatusTag,

		"alternate":      s.tplAlternate,
		"assetIntegrity": tplAssetIntegrity(baseDir),
		"gossipText":     tplGossipText,
		"assetURL":       tplAssetURL(baseDir),
		"datetime":       tplLocalDateTime,
		"date":           util.Date,
		"future":         tplFuture,
		"percentage":     tplPercentage,
		"rawPercentage":  tplRawPercentage,
		"ranking":        tplRanking,
		"until":          tplUntil,
		"unsafe": func(v string) template.HTML {
			return template.HTML(v) // nolint:gosec
		},

		"add": func(a, b int) int {
			return a + b
		},
	}
}

func tplLocalDateTime(iface interface{}) template.HTML {
	// nolint:gosec
	return template.HTML(fmt.Sprintf(
		`<span class="js-local-datetime" data-timestamp="%d">%s</span>`,
		util.Timestamp(iface),
		util.Datetime(iface),
	))
}

func tplPercentage(x int, parts ...int) string {
	pct := tplRawPercentage(x, parts...)
	if pct < 0 {
		return "- %"
	}

	return fmt.Sprintf("%d %%", pct)
}

func tplRawPercentage(x int, parts ...int) int {
	var total int
	for _, v := range parts {
		total += v
	}

	if total == 0 {
		return -1
	}

	return int(math.Round(float64(x) / float64(total) * 100.0))
}

func (s *Server) tplMatchSessionStatusTag(locale string, status back.MatchSessionStatus) template.HTML {
	var str, class string
	switch status {
	case back.MatchSessionStatusWaiting:
		str = s.locales[locale].Get("Planned")
		class = "is-success is-light"
	case back.MatchSessionStatusJoinable:
		str = s.locales[locale].Get("Joinable")
		class = "is-success is-light"
	case back.MatchSessionStatusPreparing:
		str = s.locales[locale].Get("Preparing")
		class = "is-warning is-light"
	case back.MatchSessionStatusInProgress:
		str = s.locales[locale].Get("Race in progress")
		class = "is-success"
	case back.MatchSessionStatusClosed:
		str = s.locales[locale].Get("Closed")
	default:
		return ""
	}

	return template.HTML(fmt.Sprintf(`<span class="tag is-medium is-rounded %s">%s</span>`, class, str)) // nolint:gosec
}

// nolint:gosec
func tplRanking(v interface{}) template.HTML {
	var rating, deviation float64

	switch iface := v.(type) { // HACK: Damn this is ugly. A Ranked interface maybe?
	case back.LeaderboardEntry:
		rating, deviation = iface.Rating, iface.Deviation
	case back.PlayerRating:
		rating, deviation = iface.Rating, iface.Deviation
	}

	return template.HTML(fmt.Sprintf(
		`<div class="Ranking"><span title="%d±%d">%d</span></div>`,
		int(math.Round(rating)),
		int(math.Round(deviation*2)),
		int(math.Round(rating-deviation*2)),
	))
}

func (s *Server) tplMatchSeedURL(m back.Match) string {
	if len(m.GeneratorState) == 0 {
		return "#"
	}

	gen, err := s.back.GetGenerator(m.Generator)
	if err != nil {
		log.Printf("warning: %s", err)
		return "#"
	}

	url := gen.GetDownloadURL(m.GeneratorState)
	if url == "" {
		return "#"
	}

	return url
}

func (s *Server) tplMatchEntryStatus(locale string, e back.MatchEntry) string {
	switch e.Status {
	case back.MatchEntryStatusWaiting:
		return s.locales[locale].Get("not started")
	case back.MatchEntryStatusInProgress:
		return s.locales[locale].Get("in progress")
	case back.MatchEntryStatusForfeit:
		var duration string
		if e.StartedAt.Time.Time().IsZero() {
			duration = s.locales[locale].Get("before start")
		} else {
			duration = e.EndedAt.Time.Time().Sub(e.StartedAt.Time.Time()).Round(time.Second).String()
		}

		return fmt.Sprintf(s.locales[locale].Get("forfeit (%s)"), duration)
	case back.MatchEntryStatusFinished:
		return e.EndedAt.Time.Time().Sub(e.StartedAt.Time.Time()).Round(time.Second).String()
	default:
		return "n/a"
	}
}

func tplUntil(iface interface{}, trunc string) string {
	var t time.Time
	switch iface := iface.(type) {
	case time.Time:
		t = iface
	case util.TimeAsDateTimeTZ:
		t = iface.Time()
	case util.NullTimeAsTimestamp:
		t = iface.Time.Time()
	default:
		panic(fmt.Errorf("unexpected type %T", iface))
	}

	delta := time.Until(t)

	switch trunc {
	case "m":
		delta = delta.Truncate(time.Minute)
	case "s":
		fallthrough
	default:
		delta = delta.Truncate(time.Second)
	}

	return util.FormatDuration(delta)
}

func tplFuture(iface interface{}) bool {
	var t time.Time
	switch iface := iface.(type) {
	case time.Time:
		t = iface
	case util.TimeAsDateTimeTZ:
		t = iface.Time()
	case util.NullTimeAsTimestamp:
		t = iface.Time.Time()
	default:
		panic(fmt.Errorf("unexpected type %T", iface))
	}

	return t.After(time.Now())
}

// nolint:gosec
func tplGossipText(gossip oot.SpoilerLogGossip) template.HTML {
	str := strings.ReplaceAll(gossip.Text, "^", "\n")

	r := strings.NewReplacer("^", "\n", "&", " ", "@", "Link")
	str = r.Replace(str)

	var i int
	for ; strings.Contains(str, "#"); i++ {
		if len(gossip.Colors) == 0 {
			str = strings.Replace(str, "#", `<span style="font-weight: bold">`, 1)
		} else {
			str = strings.Replace(
				str, "#",
				fmt.Sprintf(`<span style="color: %s">`, gossipColorToCSSColor(gossip.Colors[i])),
				1,
			)
		}

		str = strings.Replace(str, "#", `</span>`, 1)
	}

	return template.HTML(str)
}

func gossipColorToCSSColor(color string) string {
	switch color { // DO NOT USE HEX COLORS, the # char is a canary in tplGossipText
	case "Green":
		return "green"
	case "Red":
		return "red"
	case "Light Blue":
		return "blue"
	case "Pink":
		return "rgb(255, 0, 255)"
	default:
		return "grey"
	}
}

// tplAssetURL returns the path to an asset with a cache buster appended.
func tplAssetURL(baseDir string) func(string) string {
	hashCache := map[string]string{}

	return func(name string) string {
		base := "/_/" + name + "?"

		if h, ok := hashCache[name]; ok {
			return base + h
		}

		f, err := os.Open(filepath.Join(baseDir, "static", name))
		if err != nil {
			log.Printf("warning: unable to open %s for checksumming", name)
			return base + global.Version
		}
		defer f.Close()

		h := md5.New() // nolint:gosec
		if _, err := io.Copy(h, f); err != nil {
			log.Printf("warning: unable to checksum %s", name)
			return base + global.Version
		}

		hashCache[name] = hex.EncodeToString(h.Sum(nil))
		return base + hashCache[name]
	}
}

func tplAssetIntegrity(baseDir string) func(name string) (string, error) {
	hashCache := map[string]string{}

	return func(name string) (string, error) {
		if hash, ok := hashCache[name]; ok {
			return hash, nil
		}

		f, err := os.Open(filepath.Join(baseDir, "static", name))
		if err != nil {
			return "", err
		}
		defer f.Close()

		h := sha512.New()
		if _, err := io.Copy(h, f); err != nil {
			return "", err
		}

		hashCache[name] = "sha512-" + base64.StdEncoding.EncodeToString(h.Sum(nil))
		return hashCache[name], nil
	}
}

func (s *Server) tplAlternate(path, lang string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return ""
	}

	parts[0] = lang
	alt := strings.Join(parts, "/")

	return fmt.Sprintf("%s://%s/%s", s.config.Scheme(), s.config.Domain, alt)
}

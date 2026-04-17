package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html/template"
	"image"
	"image/jpeg"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	_ "golang.org/x/image/webp"

	"github.com/gomarkdown/markdown"
	"github.com/nfnt/resize"
	"github.com/oliamb/cutter"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	htmlMini "github.com/tdewolff/minify/v2/html"
	"golang.org/x/net/html"

	"gopkg.in/yaml.v2"
)

const imgThumbWidth = 420
const imgJpgQuality = 70

var imageExts = []string{".jpg", ".jpeg", ".png", ".webp"}

type Site struct {
	Info struct {
		Email    string `yaml:"email"`
		About    string `yaml:"about"`
		Github   string `yaml:"github"`
		LinkedIn string `yaml:"linked_in"`
	} `yaml:"info"`
	Events []struct {
		Year        int    `yaml:"year"`
		Type        string `yaml:"type"`
		Name        string `yaml:"name"`
		Description string `yaml:"desc"`
		ImagesPath  string `yaml:"images_path"`
		Images      []string
	} `yaml:"events"`
	CopyrightYear int
	EventYears    []int
}

func getPathImages(path string) []string {
	files, _ := os.ReadDir(path)
	var images []string
	for _, f := range files {
		ext := filepath.Ext(f.Name())
		if slices.Contains(imageExts, ext) {
			images = append(images, filepath.Join(path, f.Name()))
		}
	}
	return images
}

func main() {

	// parse site.yaml
	siteRaw, err := os.ReadFile("data/site.yaml")
	if err != nil {
		panic(err)
	}
	site := Site{
		CopyrightYear: time.Now().Year(),
		EventYears:    make([]int, 0),
	}
	if err := yaml.Unmarshal(siteRaw, &site); err != nil {
		panic(err)
	}

	// compile additional information
	sort.Slice(site.Events, func(i, j int) bool {
		return site.Events[i].Year > site.Events[j].Year
	})
	for i, e := range site.Events {
		if e.Year > 0 && !slices.Contains(site.EventYears, e.Year) {
			site.EventYears = append(site.EventYears, e.Year)
		}
		if e.ImagesPath != "" {
			site.Events[i].Images = getPathImages(e.ImagesPath)
		}
	}

	// load template + generate html
	t := template.New("site.tmpl").Funcs(template.FuncMap{
		"Markdown": func(v string) template.HTML {
			return template.HTML(markdown.ToHTML([]byte(v), nil, nil))
		},
	})
	t = template.Must(t.ParseFiles("data/site.tmpl"))
	buf := bytes.Buffer{}
	if err := t.Execute(&buf, site); err != nil {
		panic(err)
	}

	// crawl html look for images, resize as needed
	outputHtml := buf.String()
	doc, err := html.Parse(bytes.NewReader(buf.Bytes()))
	if err != nil {
		panic(err)
	}
	var crawler func(*html.Node)
	crawler = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "img" {
			src := ""
			for _, a := range node.Attr {
				if a.Key == "src" {
					src = a.Val
				}
			}
			imgFile, err := os.Open(src)
			if err != nil {
				panic(err)
			}
			img, _, err := image.Decode(imgFile)
			if err != nil {
				panic(err)
			}
			imgBuf := bytes.Buffer{}
			imgType := "jpeg"
			if img.Bounds().Size().X > imgThumbWidth {
				img = resize.Resize(imgThumbWidth, 0, img, resize.Lanczos3)
				img, err = cutter.Crop(img, cutter.Config{
					Width:   4,
					Height:  3,
					Mode:    cutter.Centered,
					Options: cutter.Ratio,
				})
				if err := jpeg.Encode(&imgBuf, img, &jpeg.Options{Quality: imgJpgQuality}); err != nil {
					panic(err)
				}
			} else {
				imgType = "png"
				pngEnc := png.Encoder{CompressionLevel: png.BestCompression}
				if err := pngEnc.Encode(&imgBuf, img); err != nil {
					panic(err)
				}
			}
			b64Img := base64.StdEncoding.EncodeToString(imgBuf.Bytes())
			outputHtml = strings.ReplaceAll(
				outputHtml,
				fmt.Sprintf("img src=\"%s\"", src),
				fmt.Sprintf("img src=\"data:image/%s;base64,%s\"", imgType, b64Img),
			)
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			crawler(child)
		}
	}
	crawler(doc)

	m := minify.New()
	m.AddFunc("text/html", htmlMini.Minify)
	m.AddFunc("text/css", css.Minify)
	m.Minify("text/html", os.Stdout, bytes.NewReader([]byte(outputHtml)))

}

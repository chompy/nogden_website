package main

import (
	"html/template"
	"io/ioutil"
	"os"
	"time"

	"github.com/gomarkdown/markdown"

	"gopkg.in/yaml.v2"
)

type Site struct {
	Info struct {
		Email string `yaml:"email"`
		About string `yaml:"about"`
	} `yaml:"info"`
	Categories []struct {
		Key  string `yaml:"key"`
		Name string `yaml:"name"`
		Desc string `yaml:"desc"`
	} `yaml:"categories"`
	Projects []struct {
		Name      string   `yaml:"name"`
		Category  string   `yaml:"category"`
		BuiltWith []string `yaml:"built_with"`
		Address   string   `yaml:"address"`
		Source    string   `yaml:"source"`
		Desc      string   `yaml:"desc"`
		Images    []string `yaml:"images"`
	} `yaml:"projects"`
	CopyrightYear int
}

func main() {

	// parse site.yaml
	siteRaw, err := ioutil.ReadFile("data/site.yaml")
	if err != nil {
		panic(err)
	}
	site := Site{
		CopyrightYear: time.Now().Year(),
	}
	if err := yaml.Unmarshal(siteRaw, &site); err != nil {
		panic(err)
	}

	// load template
	t := template.New("site.tmpl").Funcs(template.FuncMap{
		"Markdown": func(v string) template.HTML {
			return template.HTML(markdown.ToHTML([]byte(v), nil, nil))
		},
	})
	t = template.Must(t.ParseFiles("data/site.tmpl"))

	if err := t.Execute(os.Stdout, site); err != nil {
		panic(err)
	}

}

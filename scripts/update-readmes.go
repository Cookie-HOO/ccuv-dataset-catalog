// update-readmes rewrites the generated supported-dataset tables in both READMEs.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
)

const start = "<!-- datasets:start -->"
const end = "<!-- datasets:end -->"

type localized struct {
	EN string `json:"en"`
	ZH string `json:"zh-CN"`
}

type artifact struct {
	Platform struct {
		OS   string `json:"os"`
		Arch string `json:"arch"`
	} `json:"platform"`
}

type entry struct {
	ID         string     `json:"id"`
	Title      localized  `json:"title"`
	Summary    localized  `json:"summary"`
	Repository string     `json:"repository"`
	Artifacts  []artifact `json:"artifacts"`
}

type envelope struct {
	Signed struct {
		Entries []entry `json:"entries"`
	} `json:"signed"`
}

func main() {
	check := flag.Bool("check", false, "fail when README dataset tables are out of date")
	flag.Parse()
	data, err := os.ReadFile("catalog/catalog.json")
	if err != nil {
		panic(err)
	}
	var catalog envelope
	if err := json.Unmarshal(data, &catalog); err != nil {
		panic(err)
	}
	english := []string{"| Dataset | Summary | Repository | Platforms |", "| --- | --- | --- | --- |"}
	chinese := []string{"| 数据集 | 简介 | 仓库 | 支持平台 |", "| --- | --- | --- | --- |"}
	for _, item := range catalog.Signed.Entries {
		platforms := make([]string, 0, len(item.Artifacts))
		for _, value := range item.Artifacts {
			platforms = append(platforms, value.Platform.OS+"/"+value.Platform.Arch)
		}
		sort.Strings(platforms)
		link := "https://github.com/" + item.Repository
		english = append(english, fmt.Sprintf("| [%s](%s) | %s | %s | %s |", item.Title.EN, link, item.Summary.EN, item.Repository, strings.Join(platforms, ", ")))
		chinese = append(chinese, fmt.Sprintf("| [%s (%s)](%s) | %s | %s | %s |", item.Title.ZH, item.Title.EN, link, item.Summary.ZH, item.Repository, strings.Join(platforms, ", ")))
	}
	update("README.md", strings.Join(english, "\n"), *check)
	update("README.zh-CN.md", strings.Join(chinese, "\n"), *check)
}

func update(path, table string, check bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	content := string(data)
	startAt := strings.Index(content, start)
	endAt := strings.Index(content, end)
	if startAt < 0 || endAt < 0 || endAt <= startAt {
		panic("README dataset markers are invalid: " + path)
	}
	updated := content[:startAt+len(start)] + "\n" + table + "\n" + content[endAt:]
	if updated == content {
		return
	}
	if check {
		panic("README dataset table is out of date: " + path)
	}
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		panic(err)
	}
}

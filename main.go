package main

import (
	"bufio"
	"fmt"
	"github.com/gocolly/colly/v2"
	_ "github.com/gocolly/colly/v2/debug"
	"github.com/tealeg/xlsx/v3"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var baseQueryUrl = "http://www.hscode.net/IntegrateQueries"

type hscodeYS struct {
	Hscode     string
	Name       string
	Element    string
	Unit1      string
	Unit2      string
	Mnf        string
	Tariff     string
	Provision  string
	Excise     string
	Export     string
	Rebate     string
	Vat        string
	Customs    string
	Inspection string
}

func setHsYsFieldValue(hys *hscodeYS, fieldIndex int, value string) {
	s := reflect.ValueOf(hys).Elem()
	f := s.Field(fieldIndex)
	if value == "无" {
		value = ""
	}
	f.SetString(value)
}

func main() {
	c := colly.NewCollector(
		colly.AllowedDomains("www.hscode.net"),
		//colly.Async(true),
		//colly.Debugger(&debug.LogDebugger{}),
	)

	d := c.Clone()
	confPath, _ := os.UserConfigDir()
	hsysConfPath := path.Join(confPath, "hsys")
	noteCachePath := path.Join(hsysConfPath, "latest_cache")
	var latestDate string
	d.OnHTML("body .mlcFont", func(el *colly.HTMLElement) {
		if el.Index == 0 {
			latestDateRe := regexp.MustCompile(`\w+\/\w+\/\w+`)
			latestDate = latestDateRe.FindString(el.Text)
			f, err := os.Open(noteCachePath)
			if err == nil {
				defer f.Close()
				scanner := bufio.NewScanner(f)
				for scanner.Scan() {
					if scanner.Text() == latestDate {
						latestDate = ""
					}
				}
				if err = scanner.Err(); err != nil {
					fmt.Println(err.Error())
				}
			} else {
				fmt.Println(err.Error())
			}
			// fmt.Printf("%q\n", latest_date_re.FindAll([]byte(el.Text), -1))
		}
	})
	err := d.Visit(baseQueryUrl + "/QueryYS")
	if err != nil {
		fmt.Println("err %v", err)
	}
	if latestDate == "" {
		fmt.Println("no need crawler")
		return
	}
	fmt.Println("latest note date:", latestDate)
	firstPageReq := c.Clone()
	totalPage := 0

	hsList := make([]hscodeYS, 0)
	parseYsScListEl := func(htmlEl *colly.HTMLElement) {
		htmlEl.ForEach(".scx_item", func(i int, scxEl *colly.HTMLElement) {
			hsYs := hscodeYS{}
			scxEl.ForEach(".even", func(j int, scxEvenEl *colly.HTMLElement) {
				setHsYsFieldValue(&hsYs, j, strings.TrimSpace(scxEvenEl.Text))
				/*
					switch j {
					case 0:
						hsYs.hscode = scxEvenEl.Text
					case 1:
						hsYs.name = scxEvenEl.Text
					case 2:
						hsYs.element = scxEvenEl.Text
					}
				*/
			})
			scxEl.ForEach(".even1", func(j int, scxEven1El *colly.HTMLElement) {
				setHsYsFieldValue(&hsYs, j+2, strings.TrimSpace(scxEven1El.Text))
				/*
					switch j {
					case 0:
						hsYs.unit1 = scxEven1El.Text
					case 1:
						hsYs.unit2 = scxEven1El.Text
					case 2:
						hsYs.mnf = scxEven1El.Text
					case 3:
						hsYs.tariff = scxEven1El.Text
					case 4:
						hsYs.provision = scxEven1El.Text
					case 5:
						hsYs.excise = scxEven1El.Text
					case 6:
						hsYs.export = scxEven1El.Text
					case 7:
						hsYs.rebate = scxEven1El.Text
					case 8:
						hsYs.vat = scxEven1El.Text
					case 9:
						hsYs.customs = scxEven1El.Text
					case 10:
						hsYs.inspection = scxEven1El.Text
					}
				*/
			})
			hsList = append(hsList, hsYs)
		})
	}
	firstPageReq.OnHTML(".scx_listitem_0", parseYsScListEl)
	firstPageReq.OnHTML(".total_info", func(tinEl *colly.HTMLElement) {
		page_count_re := regexp.MustCompile(`\w+`)
		totalPage, _ = strconv.Atoi(page_count_re.FindString(tinEl.Text))
	})
	reqData := map[string]string{
		"pageIndex": "1",
	}
	err = firstPageReq.Post(baseQueryUrl+"/YsInfoPager", reqData)
	if err != nil {
		fmt.Println("err %v", err)
	}
	c.OnHTML(".scx_listitem_0", parseYsScListEl)
	c.Limit(&colly.LimitRule{
		Parallelism: 2,
		RandomDelay: 5 * time.Second,
	})

	for page := 1; page < totalPage; page++ {
		reqData["pageIndex"] = strconv.Itoa(page + 1)
		err = c.Post(baseQueryUrl+"/YsInfoPager", reqData)
		if err != nil {
			fmt.Println("err %v", err)
		}
	}
	xlsHeader := []string{"商品编码", "商品名称", "申报要素", "法一单位", "法二单位",
		"最惠国进口税率", "普通进口税率", "暂定进口税率", "消费税率", "出口关税率",
		"出口退税率", "增值税率", "海关监管条件", "检验检疫类别"}
	var f *xlsx.File = xlsx.NewFile()
	sheet, err := f.AddSheet("海关HSCode表")
	if err != nil {
		fmt.Printf(err.Error())
	}
	var xlsxStyle *xlsx.Style = &xlsx.Style{
		Alignment: *xlsx.DefaultAlignment(),
		Border:    *xlsx.NewBorder("thin", "thin", "thin", "thin"),
		Fill:      *xlsx.DefaultFill(),
		Font:      *xlsx.DefaultFont(),
	}
	var row *xlsx.Row = sheet.AddRow()
	var cell *xlsx.Cell
	for _, header := range xlsHeader {
		cell = row.AddCell()
		cell.SetStyle(xlsxStyle)
		cell.Value = header
	}
	for _, hsys := range hsList {
		row = sheet.AddRow()
		s := reflect.ValueOf(&hsys).Elem()
		for i := 0; i < s.NumField(); i++ {
			cell = row.AddCell()
			cell.SetStyle(xlsxStyle)
			cell.Value = s.Field(i).String()
		}
	}
	now := time.Now()
	err = f.Save(fmt.Sprintf("商品编码表-%4d%02d.xlsx", now.Year(), now.Month()))
	if err != nil {
		fmt.Printf(err.Error())
	}
	err = os.MkdirAll(hsysConfPath, 0755)
	if err != nil {
		fmt.Printf(err.Error())
	} else {
		err := ioutil.WriteFile(noteCachePath, []byte(latestDate), 0755)
		if err != nil {
			fmt.Printf(err.Error())
		}
	}
}

package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
)

type tests struct {
	Total int `json:"total"`
}

type deaths struct {
	New   string `json:"new"`
	Total int    `json:"total"`
}

type cases struct {
	New       string `json:"new"`
	Active    int    `json:"active"`
	Critical  int    `json:"critical"`
	Recovered int    `json:"recovered"`
	Total     int    `json:"total"`
}

type response struct {
	Country string `json:"country"`
	Cases   cases  `json:"cases"`
	Deaths  deaths `json:"deaths"`
	Tests   tests  `json:"tests"`
	Day     string `json:"day"`
	Time    string `json:"time"`
}

type params struct {
	Country string `json:"country"`
}

type covidReport struct {
	Get        string     `json:"get"`
	Parameters params     `json:"parameters"`
	Errors     []int      `json:"errors"`
	Results    int        `json:"results"`
	Response   []response `json:"response"`
}

func covid(country string) (string, error) {

	var report covidReport

	url := "https://covid-193.p.rapidapi.com/statistics?country=" + country
	client := resty.New()

	resp, err := client.R().
		SetHeader("x-rapidapi-host", "covid-193.p.rapidapi.com").
		SetHeader("x-rapidapi-key", *rToken).
		Get(url)

	if err != nil {
		return "", err
	}

	err = json.Unmarshal(resp.Body(), &report)

	if report.Results < 1 {
		return fmt.Sprintf("No results for %s. %s", country, nfStrings[rnd.Intn(len(nfStrings))]), nil
	}

	if country == "all" {
		return fmt.Sprintf("Covid-19 World: %d active cases, %d critical cases, %d recoverd, %d total cases, %d deaths.\n",
			report.Response[0].Cases.Active, report.Response[0].Cases.Critical, report.Response[0].Cases.Recovered, report.Response[0].Cases.Total, report.Response[0].Deaths.Total), nil
	}
	return fmt.Sprintf("Covid-19 %s: %d tested, %d active cases, %d critical cases, %d recoverd, %d total cases, %d deaths.\n",
		report.Response[0].Country, report.Response[0].Tests.Total, report.Response[0].Cases.Active, report.Response[0].Cases.Critical, report.Response[0].Cases.Recovered, report.Response[0].Cases.Total, report.Response[0].Deaths.Total), nil
}

func reaper() (string, error) {

	var report covidReport

	url := "https://covid-193.p.rapidapi.com/statistics?country=usa"
	client := resty.New()

	resp, err := client.R().
		SetHeader("x-rapidapi-host", "covid-193.p.rapidapi.com").
		SetHeader("x-rapidapi-key", *rToken).
		Get(url)

	if err != nil {
		return "", err
	}

	err = json.Unmarshal(resp.Body(), &report)

	if report.Results < 1 {
		return "No death count available.", nil
	}

	t, _ := time.Parse(time.RFC3339, report.Response[0].Time)
	location, err := time.LoadLocation("America/New_York")
	var tStr string
	// var tLoc time.Time

	if err != nil {
		tStr = report.Response[0].Time
	} else {
		tLoc := t.In(location)
		zone, _ := tLoc.Zone()
		tStr = tLoc.Format("2006-01-02 @ 15:04 ") + zone
	}

	return fmt.Sprintf("USA (%s): %d covid-19 deaths.\n", tStr, report.Response[0].Deaths.Total), nil
}

func cronReport() {
	if len(covChans) > 0 {
		report, err := reaper()
		if err == nil {
			for c := range covChans {
				discord.ChannelMessageSend(c, report)
			}
		}
	}
}

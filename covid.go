package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
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
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("x-rapidapi-host", "covid-193.p.rapidapi.com")
	req.Header.Add("x-rapidapi-key", *rToken)
	res, err := http.DefaultClient.Do(req)

	if err != nil {
		return "", err
	}

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	err = json.Unmarshal(body, &report)

	if report.Results < 1 {
		return fmt.Sprintf("No results for %s. %s", country, nfStrings[rnd.Intn(len(nfStrings))]), nil
	}

	if country == "all" {
		return fmt.Sprintf("Covid-19 World: %d active cases, %d critical cases, %d recoverd, %d total cases, %d deaths (%s).\n",
			report.Response[0].Cases.Active, report.Response[0].Cases.Critical, report.Response[0].Cases.Recovered,
			report.Response[0].Cases.Total, report.Response[0].Deaths.Total, report.Response[0].Deaths.New), nil
	}
	return fmt.Sprintf("Covid-19 %s: %d tested, %d active cases, %d critical cases, %d recoverd, %d total cases, %d deaths (%s).\n",
		report.Response[0].Country, report.Response[0].Tests.Total, report.Response[0].Cases.Active, report.Response[0].Cases.Critical,
		report.Response[0].Cases.Recovered, report.Response[0].Cases.Total, report.Response[0].Deaths.Total, report.Response[0].Deaths.New), nil
}

func reaper() (string, error) {

	var report covidReport

	url := "https://covid-193.p.rapidapi.com/statistics?country=usa"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("x-rapidapi-host", "covid-193.p.rapidapi.com")
	req.Header.Add("x-rapidapi-key", *rToken)
	res, err := http.DefaultClient.Do(req)

	if err != nil {
		return "", err
	}

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	err = json.Unmarshal(body, &report)

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

	return fmt.Sprintf("USA (%s): %d covid-19 deaths. (%s)\n", tStr, report.Response[0].Deaths.Total, report.Response[0].Deaths.New), nil
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

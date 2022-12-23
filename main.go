package main

import (
	"fmt"
	"log"
	"net/http"
	"math/rand"
	"time"
	"github.com/PuerkitoBio/goquery"
	"strings"
)


type SeoData struct{
	Url				string
	Title 			string
	H1				string
	MetaDescription string
	StatusCode		int
}

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/61.0.3163.100 Safari/537.36",
	"Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/61.0.3163.100 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/61.0.3163.100 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/604.1.38 (KHTML, like Gecko) Version/11.0 Safari/604.1.38",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:56.0) Gecko/20100101 Firefox/56.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13) AppleWebKit/604.1.38 (KHTML, like Gecko) Version/11.0 Safari/604.1.38",
}

func randomUserAgent()string{
	rand.Seed(time.Now().Unix())
	randNum := rand.Int() % len(userAgents)
	return userAgents[randNum]
}


func makeRequest(link string)(*http.Response, error){
	client := http.Client{}
	req, _:= http.NewRequest("GET", link, nil)
	req.Header.Set("User-Agent", randomUserAgent())
	response, err := client.Do(req)

	return response, err
}


func extractUrls(response *http.Response) ([]string, error){
	doc, err := goquery.NewDocumentFromReader(response.Body)

	results := []string{}
	sel := doc.Find("loc")
	for i := range sel.Nodes{
		loc := sel.Eq(i)
		result := loc.Text()
		results = append(results, result)
	}

	return results, err
}


func isSitemap(linkList []string)([]string, []string){
	siteMapUrls := []string{}
	pageUrls := []string{}
	for _, link := range linkList{
		foundLink := strings.Contains(link, "xml")
		if foundLink {
			fmt.Println("Found Sitemap: ", link)
			siteMapUrls = append(siteMapUrls, link)
		} else{
			pageUrls = append(pageUrls, link)
		}
	}
	return siteMapUrls, pageUrls
}


func extractSitemapUrls(url string)([]string){
	workList := make(chan []string)	
	toCrawl := []string{}
	var n int
	n++
	go func(url string){workList <- []string{url}}(url)

	for ; n > 0; n--{
		list := <- workList
		for _, link := range list{
			n++
			go func(url string){
				res, _ := makeRequest(url)
				urlList, err := extractUrls(res)
				if err != nil{
					log.Println("Error while extracting the url")
				}
				siteMap, pages := isSitemap(urlList)
				if siteMap != nil {
					workList <- siteMap
				}
				for _, page := range pages{
					toCrawl = append(toCrawl, page)
				}

			}(link)
		}
	}
	return toCrawl
}



func crawl(tocrawl string)(*http.Response, error){
	res, err := makeRequest(tocrawl)
	if err != nil {
		fmt.Println("error while making request: ", err)
	}
	return res, nil
}


func scrapePage(scrapeUrls string)(SeoData, error){
	res, err := crawl(scrapeUrls)
	if err != nil {
		return SeoData{}, err
	}

	data, err := seoData(res)
	if err != nil {
		return SeoData{}, err
	}
	return data, nil
}


func seoData(res *http.Response)(SeoData, error){
	doc, _ := goquery.NewDocumentFromReader(res.Body)

	result := SeoData{}
	result.Title = doc.Find("title").First().Text()
	result.H1 = doc.Find("h1").First().Text()
	result.MetaDescription, _ = doc.Find("meta[name^=description]").Attr("content")
	result.StatusCode = res.StatusCode
	result.Url = res.Request.URL.String()

	return result, nil
}


func scrapeSitemap(urls []string)[]SeoData{
	workList := make(chan []string)
	var n int
	n++
	go func() { workList <- urls }()
	seoData := []SeoData{}
	for ; n > 0; n--{
		list := <- workList

		for _, link := range list{
			if link != ""{
				n++
				go func(link string){
					log.Printf("Requesting Urls: %s", link)
					results, err := scrapePage(link)
					if err != nil {
						fmt.Println("error while scraping the page: ", err)
					}
					seoData = append(seoData, results) 
					workList <- []string{}	
				}(link)
			}
		}
	}
	return seoData

}



func scrapeSitemapUrls(url string)[]SeoData{
	tocrawl := extractSitemapUrls(url)
	res := scrapeSitemap(tocrawl)

	return res
}


func main(){
	results := scrapeSitemapUrls("https://www.rocktherankings.com/sitemap.xml")
	for _, res := range results{
		fmt.Println(res)
	}
}
package sitemap

import (
    "log"
    "strconv"
    "strings"
    "io/ioutil"
    "compress/gzip"
    "os"
    "time"
    "net/http"
)

type SitemapGroup struct {
	name    string
    urls    []URL
    url_channel    chan URL
    group_count    int
    folder  string
}

type HttpResponse struct {
    url      string
    response *http.Response
    err      error
}

func (s *SitemapGroup) Add (url URL ) {
    s.url_channel <- url
}

func (s *SitemapGroup) CloseGroup ( ) {
    s.Create(s.getURLSet())
    close(s.url_channel)
    s.Clear()
}

func (s *SitemapGroup) Clear ( ) {
    s.urls = []URL{}
}

func (s *SitemapGroup) getURLSet() URLSet{
    return URLSet{URLs: s.urls}
}

func (s *SitemapGroup) Create (url_set URLSet) {

        xml := createXML(url_set)

        var path string = s.folder + s.name + "_" + strconv.Itoa(s.group_count) + ".xml.gz"

        err := saveXml(xml, path)

        if err != nil {
            log.Fatal("File not saved:", err)
        }
        log.Printf("Sitemap created on %s", path)
        s.group_count++

}

func NewSitemapGroup(folder string,name string) *SitemapGroup {
	s := new(SitemapGroup)
    s.name = strings.Replace(name, ".xml.gz", "", 1)
    s.group_count = 1
    s.url_channel = make(chan URL)
    _, err := ioutil.ReadDir(folder)
	if err != nil {
		log.Fatal("Dir not allowed - ", err)
	}
    s.folder = folder

    go func() {
        for entry := range s.url_channel {

            s.urls = append(s.urls, entry)

            if len(s.urls) == MAXURLSETSIZE {

                go func(urls URLSet) {
                    s.Create(urls)
                }(s.getURLSet())

                s.Clear()
            }
        }
    }()

	return s
}


func createXML(group URLSet) (sitemapXml []byte) {
    sitemapXml, err := createSitemapXml(group)
    if err != nil {
        log.Fatal("work failed:", err)
    }
    return
}

func saveXml(xmlFile []byte, path string) (err error){

        fo, err := os.Create(path)
        if err != nil {
            return err
        }
        defer fo.Close()

        zip := gzip.NewWriter(fo)
        defer zip.Close()
        _, err = zip.Write(xmlFile)
        if err != nil {
            return err
        }

        return err

}

func CreateSitemapIndex(indexFile string, folder string, public_dir string) (err error) {


    fs, err := ioutil.ReadDir(folder)
    if err != nil {
        return err
    }

    var index = Index{Sitemaps:[]Sitemap{}}
    //search sitemaps
    for _, f := range fs {
        if strings.HasSuffix(f.Name(), ".xml.gz") && !strings.HasSuffix(indexFile, f.Name()) {
            index.Sitemaps = append(index.Sitemaps, Sitemap{Loc: public_dir + f.Name(),LastMod: time.Now()})
        }
    }

    //create xml
    indexXml, err := createSitemapIndexXml(index)
    if err != nil {
        return err
    }
    //touch path
    fo, err := os.Create(indexFile)
    if err != nil {
        return err
    }
    defer fo.Close()
    //Save gzip
    zip := gzip.NewWriter(fo)
    defer zip.Close()
    _, err = zip.Write(indexXml)
    if err != nil {
        return err
    }

    log.Printf("Sitemap Index created on %s", indexFile)
    return err
}

func PingSearchEngines(indexFile string) {
    var urls = []string{
    	"http://www.google.com/webmasters/tools/ping?sitemap="+indexFile,
    	"http://submissions.ask.com/ping?sitemap="+indexFile,
    	"http://www.bing.com/webmaster/ping.aspx?siteMap="+indexFile,
        "http://search.yahooapis.com/SiteExplorerService/V1/ping?sitemap="+indexFile
        //"http://search.yahooapis.com/SiteExplorerService/V1/updateNotification?appid=USERID&url="
    }

    results := asyncHttpGets(urls)
    for _ = range urls {
		result := <-results
		log.Printf("%s status: %s\n", result.url, result.response.Status)
	}

}

func asyncHttpGets(urls []string) <-chan *HttpResponse {
	ch := make(chan *HttpResponse, len(urls)) // buffered
	for _, url := range urls {
		go func(url string) {
			log.Printf("Fetching %s \n", url)
			resp, err := http.Get(url)
			resp.Body.Close()
			ch <- &HttpResponse{url, resp, err}
		}(url)
	}
	return ch
}

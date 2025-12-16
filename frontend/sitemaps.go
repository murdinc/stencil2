package frontend

import (
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/murdinc/stencil2/configs"
	"github.com/murdinc/stencil2/database"
)

type SitemapIndex struct {
	XMLName     xml.Name     `xml:"sitemapindex"`
	Xmlns       string       `xml:"xmlns,attr"`
	SitemapURLs []SitemapURL `xml:"sitemap"`
}

type SitemapURL struct {
	XMLName xml.Name `xml:"url"`
	Loc     string   `xml:"loc"`
	LastMod string   `xml:"lastmod"`
}

type URL struct {
	XMLName    xml.Name `xml:"url"`
	Loc        string   `xml:"loc"`
	LastMod    string   `xml:"lastmod"`
	ChangeFreq string   `xml:"changefreq"`
	Priority   string   `xml:"priority"`
}

type URLSet struct {
	XMLName xml.Name `xml:"urlset"`
	Xmlns   string   `xml:"xmlns,attr"`
	URLs    []URL    `xml:"url"`
}

func InitSitemaps(envConfig configs.EnvironmentConfig, websiteConfig configs.WebsiteConfig) {
	dbConn := &database.DBConnection{}

	// Open a connection to the MySQL database
	err := dbConn.Connect(envConfig.Database.User, envConfig.Database.Password, envConfig.Database.Host, envConfig.Database.Port, websiteConfig.Database.Name, 1000)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}

	if dbConn.Connected {

		defer dbConn.Database.Close()

		err = dbConn.InitSitemaps()
		if err != nil {
			log.Fatalf("Failed to initialize sitemaps: %v", err)
		}

		sitemapsDir := filepath.Join(websiteConfig.Directory, "sitemaps")
		DeleteXMLFiles(sitemapsDir)

		log.Printf("Successfully initialized sitemaps for site: [%s]", websiteConfig.SiteName)

		return
	}

	log.Printf("No Database connection, skipping [%s]...", websiteConfig.SiteName)

}

func DeleteXMLFiles(sitemapsDir string) error {
	// Get a list of .xml files in the directory
	xmlFiles, err := filepath.Glob(filepath.Join(sitemapsDir, "*.xml"))
	if err != nil {
		return err
	}

	// Delete each .xml file
	for _, xmlFile := range xmlFiles {
		err := os.Remove(xmlFile)
		if err != nil {
			return err
		}
		log.Printf("Deleted: %s\n", xmlFile)
	}

	return nil
}

func BuildSitemaps(envConfig configs.EnvironmentConfig, websiteConfig configs.WebsiteConfig) {
	dbConn := &database.DBConnection{}

	// Open a connection to the MySQL database
	err := dbConn.Connect(envConfig.Database.User, envConfig.Database.Password, envConfig.Database.Host, envConfig.Database.Port, websiteConfig.Database.Name, 1000)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}

	if dbConn.Connected {

		defer dbConn.Database.Close()

		incompleteDates, err := dbConn.GetIncompleteSitemaps()
		if err != nil {
			log.Fatalf("Failed to get incomplete sitemaps list: %v", err)
		}

		if len(incompleteDates) < 1 {
			log.Printf("No incomplete sitemaps found for site: [%s]", websiteConfig.SiteName)
			return
		}

		urlSet := URLSet{
			Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		}

		// Create the 'sitemaps' directory if it doesn't exist
		sitemapsDir := filepath.Join(websiteConfig.Directory, "sitemaps")
		if err := os.MkdirAll(sitemapsDir, os.ModePerm); err != nil {
			log.Fatalf("Error creating 'sitemaps' directory:", err)
			return
		}

		for _, month := range incompleteDates {
			posts, err := dbConn.GetPublishedPostsByMonth(month)
			if err != nil {
				log.Fatalf("Failed to get incomplete sitemap posts: %v", err)
			}

			var urls []URL

			for _, post := range posts {
				url := URL{
					Loc:        "https://" + websiteConfig.SiteName + post.URL,
					LastMod:    post.Updated.Format("2006-01-02T15:04:05-07:00"),
					ChangeFreq: "daily",
					Priority:   "0.5",
				}

				urls = append(urls, url)
			}

			urlSet.URLs = urls

			output, err := xml.MarshalIndent(urlSet, "", "    ")
			if err != nil {
				log.Fatalf("Failed to marshal XML sitemap file: %v", err)
				return
			}

			output = []byte(xml.Header + string(output))

			filename := filepath.Join(sitemapsDir, fmt.Sprintf("sitemap-%s.xml", month.Format("2006-01")))

			file, err := os.Create(filename)
			if err != nil {
				log.Fatalf("Failed to create sitemap file: %v", err)
				return
			}
			defer file.Close()

			_, err = file.Write(output)
			if err != nil {
				log.Fatalf("Failed to write sitemap file: %v", err)
			}
			log.Printf("Sitemap [%s] generated successfully.", filename)
		}

		existingSitemaps, err := filepath.Glob(filepath.Join(sitemapsDir, "sitemap-*.xml"))
		if err != nil {
			log.Fatalf("Failed to find existing sitemap files: %v", err)
			return
		}

		var sitemapURLs []SitemapURL

		// Add existing sitemaps URLs to the sitemap index
		for _, existingSitemap := range existingSitemaps {
			fi, err := os.Stat(existingSitemap)
			if err != nil {
				log.Fatalf("Failed to stat sitemap file: %v", err)
				return
			}

			base := filepath.Base(existingSitemap)

			if base != "sitemaps-index.xml" {
				sitemapURL := SitemapURL{
					Loc:     "https://" + websiteConfig.SiteName + "/sitemaps/" + base,
					LastMod: fi.ModTime().Format("2006-01-02T15:04:05-07:00"),
				}
				sitemapURLs = append(sitemapURLs, sitemapURL)
			}
		}

		// Create sitemaps index
		sitemapIndex := SitemapIndex{
			Xmlns:       "http://www.sitemaps.org/schemas/sitemap/0.9",
			SitemapURLs: sitemapURLs,
		}

		indexOutput, err := xml.MarshalIndent(sitemapIndex, "", "    ")
		if err != nil {
			log.Fatalf("Failed to marshal XML index file: %v", err)
			return
		}

		indexOutput = []byte(xml.Header + string(indexOutput))

		indexFilename := filepath.Join(sitemapsDir, "sitemaps-index.xml")

		indexFile, err := os.Create(indexFilename)
		if err != nil {
			log.Fatalf("Failed to create index file: %v", err)
			return
		}
		defer indexFile.Close()

		_, err = indexFile.Write(indexOutput)
		if err != nil {
			log.Fatalf("Failed to write index file: %v", err)
		}

		log.Printf("Sitemap [%s] generated successfully.", indexFilename)

		err = dbConn.MarkSitemapsAsComplete()
		if err != nil {
			log.Fatalf("Failed to mark sitemaps as complete: %v", err)
		}

		log.Printf("Successfully built incomplete sitemaps for site: [%s]", websiteConfig.SiteName)

		return
	}

	log.Printf("No Database connection, skipping [%s]...", websiteConfig.SiteName)

}

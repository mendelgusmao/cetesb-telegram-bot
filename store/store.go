package store

import (
	"fmt"
	"hash/crc32"
	"log"
	"time"

	"github.com/mendelgusmao/cetesb-telegram-bot/scraper"
	"github.com/mendelgusmao/scoredb/lib/database"
)

func New(database *database.Database, scraper *scraper.Scraper) *Store {
	return &Store{
		database: database,
		scraper:  scraper,
	}
}

func (s *Store) ScrapeAndStore() error {
	cities, beaches, err := s.ScrapeAndTransform()

	if err != nil {
		return fmt.Errorf("Store.ScrapeAndStore: %v", err)
	}

	if err := s.Store("cities", cities); err != nil {
		return err
	}

	return s.Store("beaches", beaches)
}

func (s *Store) ScrapeAndTransform() (cities []database.Document, beaches []database.Document, err error) {
	scrapedCityBeaches, err := s.scraper.Scrape()

	if err != nil {
		return nil, nil, fmt.Errorf("Store.ScrapeAndTransform: %v", err)
	}

	cities = make([]database.Document, 0)
	beaches = make([]database.Document, 0)

	checksum := crc32.ChecksumIEEE([]byte(fmt.Sprintf("%v|%v", cities, beaches)))

	if checksum == s.lastChecksum {
		return
	}

	s.lastChecksum = checksum
	log.Println("[store.Scrape] Change detected! Going to update the database.")

	for cityName, cityBeaches := range scrapedCityBeaches {
		for _, beach := range cityBeaches {
			beachDocument := database.Document{
				Keys: []string{
					beach.Name,
					fmt.Sprintf("%s %s", beach.Name, beach.City.Name),
					fmt.Sprintf("%s %s", beach.City.Name, beach.Name),
				},
				Content: beach,
			}

			beaches = append(beaches, beachDocument)
		}

		cities = append(cities, database.Document{
			Keys:    []string{cityName},
			Content: cityBeaches,
		})
	}

	log.Printf("[store.Scrape] Found %d cities\n", len(scrapedCityBeaches))
	log.Printf("[store.Scrape] Found %d beaches\n", len(beaches))

	return
}

func (s *Store) Store(collection string, documents []database.Document) error {
	var err error

	if !s.database.CollectionExists(collection) {
		err = s.database.CreateCollection(collection, databaseConfiguration, documents)
	} else {
		err = s.database.UpdateCollection(collection, documents)
	}

	if err != nil {
		return fmt.Errorf("[store.Store] %v", err)
	}

	return nil
}

func (s *Store) Query(key string) (QueryResult, error) {
	cityMatches, _ := s.database.Query("cities", key)
	citiesQueryResult := newQueryResult("cities", cityMatches)

	if citiesQueryResult.HasPerfectMatches {
		return citiesQueryResult, nil
	}

	beachMatches, _ := s.database.Query("beaches", key)

	return newQueryResult("beaches", beachMatches), nil
}

func (s *Store) Work() {
	if err := s.ScrapeAndStore(); err != nil {
		log.Printf("[store.Work] %v", err)
	}

	ticker := time.NewTicker(1 * time.Hour)

	go func() {
		for range ticker.C {
			if err := s.ScrapeAndStore(); err != nil {
				log.Printf("[store.Work (ticker)] %v", err)
			}
		}
	}()
}

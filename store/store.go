package store

import (
	"fmt"
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
	cities, beaches := s.Scrape()

	if err := s.Store("cities", cities); err != nil {
		return err
	}

	return s.Store("beaches", beaches)
}

func (s *Store) Scrape() (cities []database.Document, beaches []database.Document) {
	scrapedCities := s.scraper.ScrapeCities()
	cities = make([]database.Document, len(scrapedCities))
	beaches = make([]database.Document, 0)

	for cityIndex, city := range scrapedCities {
		scrapedBeaches := s.scraper.ScrapeBeaches(city)

		for _, beach := range scrapedBeaches {
			beachDocument := database.Document{
				Keys: []string{
					beach.Name,
					fmt.Sprintf("%s %s", beach.City.Name, beach.Name),
				},
				ExactKeys: []string{
					beach.Name,
				},
				Content: beach,
			}

			beaches = append(beaches, beachDocument)

			break
		}

		cities[cityIndex] = database.Document{
			Keys:      []string{city.Name},
			ExactKeys: []string{city.Name},
			Content:   scrapedBeaches,
		}

		break
	}

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

func (s *Store) StartUpdater() {
	ticker := time.NewTicker(1 * time.Hour)

	go func() {
		for {
			select {
			case <-ticker.C:
				err := s.ScrapeAndStore()

				if err != nil {
					fmt.Printf("[store.StartUpdater] %v", err)
				}
			}
		}
	}()
}

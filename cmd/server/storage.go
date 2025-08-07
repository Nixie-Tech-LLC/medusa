package main

import (
	"log"

	"github.com/Nixie-Tech-LLC/medusa/internal/storage"
)

// InitStorage selects and returns the configured storage backend
func InitStorage(env Environment) storage.Storage {
	if env.UseSpaces {
		spacesStorage, err := storage.NewSpacesStorage(
			env.SpacesEndpoint,
			env.SpacesRegion,
			env.SpacesBucket,
			env.SpacesCDNURL,
			env.SpacesAccessKey,
			env.SpacesSecretKey,
		)
		if err != nil {
			log.Fatalf("failed to initialize Spaces storage: %v", err)
		}
		log.Printf("Using DigitalOcean Spaces storage with CDN: %s", env.SpacesCDNURL)
		return spacesStorage
	}

	local := storage.NewLocalStorage("./uploads")
	log.Printf("Using local file storage in ./uploads")
	return local
}

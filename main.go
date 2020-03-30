package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Root      string `default:"/"`
	Bind      string `default:":8000"`
	SecretKey string `required:"true" split_words:"true"`
}

type ServiceUpdate struct {
	SecretKey *string
	ID        *string
	Image     *string
}

func addConfig(c Config, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = context.WithValue(ctx, "config", c)
		r = r.WithContext(ctx)
		http.HandlerFunc(h).ServeHTTP(w, r)
	}
}

func handlePush(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "404 not found.", http.StatusNotFound)
		return
	}

	if r.Method == "GET" || r.Method == "POST" {
		cfg, ok := r.Context().Value("config").(Config)
		if !ok {
			http.Error(w, "Could not load config", http.StatusInternalServerError)
			return
		}

		cli, err := client.NewClientWithOpts(client.FromEnv)
		if err != nil {
			log.Printf("%v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		services, err := cli.ServiceList(context.Background(), types.ServiceListOptions{})
		if err != nil {
			log.Printf("%v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if r.Method == "POST" {
			var update ServiceUpdate

			err = json.NewDecoder(r.Body).Decode(&update)
			if err != nil {
				log.Printf("%v", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if update.SecretKey == nil {
				http.Error(w, "SecretKey is a required field", http.StatusBadRequest)
				return
			}

			if update.ID == nil {
				http.Error(w, "ID is a required field", http.StatusBadRequest)
				return
			}

			if update.Image == nil {
				http.Error(w, "Image is a required field", http.StatusBadRequest)
				return
			}

			if *update.SecretKey != cfg.SecretKey {
				log.Printf("Incorrect SECRET_KEY used")
				http.Error(w, "SecretKey does not match", http.StatusForbidden)
				return
			}

			services_updated := make(map[string]string)
			for _, service := range services {
				keeper_id, found := service.Spec.Labels["keeper.id"]

				if found && keeper_id == *update.ID {
					service.Spec.TaskTemplate.ContainerSpec.Image = *update.Image
					resp, err := cli.ServiceUpdate(
						context.Background(),
						service.ID,
						service.Version,
						service.Spec,
						types.ServiceUpdateOptions{QueryRegistry: true},
					)

					if err != nil {
						log.Printf("%v", err)
						http.Error(w, err.Error(), http.StatusServiceUnavailable)
						return
					} else {
						services_updated[service.Spec.Name] = strings.Join(resp.Warnings, ", ")
					}
				}
			}

			if len(services_updated) == 0 {
				log.Printf("Could not find service: %s", *update.ID)
				http.Error(w, fmt.Sprintf("Could not find service: %s", *update.ID), http.StatusNotFound)
				return
			} else {
				log.Printf("Updated services. ID: %s, Image: %s ", *update.ID, *update.Image)
				for service, warnings := range services_updated {
					log.Printf("Service %s updated with following response %s", service, warnings)
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(services_updated)
			}
		} else {
			supported_services := make(map[string]string)
			for _, service := range services {
				keeper_id, found := service.Spec.Labels["keeper.id"]

				if found {
					supported_services[keeper_id] = service.Spec.Name
				}
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(supported_services)
		}
	} else {
		http.Error(w, "Only GET and POST is supported", http.StatusMethodNotAllowed)
	}
}

func main() {
	var cfg Config
	err := envconfig.Process("KEEPER", &cfg)
	if err != nil {
		log.Fatal(err.Error())
	}

	http.HandleFunc(cfg.Root, addConfig(cfg, handlePush))
	log.Printf("Starting docker-keeper on %s...\n", cfg.Bind)
	if err := http.ListenAndServe(cfg.Bind, nil); err != nil {
		log.Fatal(err)
	}
}

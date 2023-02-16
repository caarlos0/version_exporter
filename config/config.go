package config

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	yaml "gopkg.in/yaml.v2"
)

// Config struct representing the config file.
type Config struct {
	Repositories map[string]string `yaml:"repositories"`
}

func doLoad(file string, config *Config) error {
	bts, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	var newConfig Config
	if err := yaml.Unmarshal(bts, &newConfig); err != nil {
		return err
	}
	*config = newConfig
	return nil
}

// Load loads a config file and reloads it if a SIGHUP is received.
func Load(file string, config *Config, onReload func()) {
	if err := doLoad(file, config); err != nil {
		log.Fatalln("failed to load config: ", err)
	}
	configCh := make(chan os.Signal, 1)
	signal.Notify(configCh, syscall.SIGHUP)
	go func() {
		for range configCh {
			if err := doLoad(file, config); err != nil {
				log.Fatalln("failed to reload config: ", err)
			}
			onReload()
			log.Println("config reloaded...")
		}
	}()
}

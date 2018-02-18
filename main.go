package main

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/go-ini/ini"
	"github.com/hashicorp/consul/api"
	flags "github.com/jessevdk/go-flags"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"net/url"
	"os"
)

// Options is command line flags.
type Options struct {
	Profile string `short:"p" long:"profile" default:"DEFAULT" description:"DC section name to print."`
	All     bool   `short:"a" long:"all" description:"Print all nodes on all DC."`
}

func getConfigs() (map[string]api.Config, error) {
	configs := map[string]api.Config{}
	f := fmt.Sprintf("%s/.cnodes", os.Getenv("HOME"))
	data, err := ioutil.ReadFile(f)
	if err != nil {
		return configs, errors.Wrapf(err, "failed to read %q", f)
	}
	ini, err := ini.Load(data, f)
	if err != nil {
		return configs, errors.Wrapf(err, "failed to load %q", f)
	}
	for _, sec := range ini.Sections() {
		apiConfig := api.DefaultConfig()
		consulURL := sec.Key("url").String()

		u, err := url.Parse(consulURL)
		if err != nil {
			return configs, errors.Wrapf(err, "failed to parse consulURL %q", consulURL)
		}
		apiConfig.Address = u.Host
		apiConfig.Scheme = u.Scheme
		configs[sec.Name()] = *apiConfig
	}
	return configs, nil
}

func printNodes(config api.Config) error {
	client, err := api.NewClient(&config)
	if err != nil {
		return errors.Wrap(err, "failed to create consul client")
	}
	catalog := client.Catalog()
	queryOptions := api.QueryOptions{}
	nodes, _, err := catalog.Nodes(&queryOptions)
	if err != nil {
		return errors.Wrapf(err, "failed to fetch consul nodes")
	}

	health := client.Health()
	healthChecks, _, err := health.State("any", &queryOptions)
	if err != nil {
		return errors.Wrapf(err, "failed to fetch consul health states")
	}

	for _, node := range nodes {
		status := "passing"
		for _, healthCheck := range healthChecks {
			if healthCheck.Node == node.Node && healthCheck.Status != "passing" {
				status = healthCheck.Status
			}
		}

		l := fmt.Sprintf("%-9s %-16s %s %s\n", status, node.Address, node.Node, node.Datacenter)
		if status == "passing" {
			color.Green(l)
		} else if status == "critical" {
			color.Red(l)

		}
	}
	return nil
}

func main() {
	var opts Options
	_, err := flags.Parse(&opts)

	if err != nil {
		log.Fatalln(err)
	}

	configs, err := getConfigs()
	if err != nil {
		log.Fatalln(err)
	}
	if opts.All {
		for _, cfg := range configs {
			printNodes(cfg)
		}
	} else {
		err = printNodes(configs[opts.Profile])
		if err != nil {
			log.Fatalln(err)
		}
	}
}

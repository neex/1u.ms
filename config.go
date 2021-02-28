package main

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/miekg/dns"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Domain            string
	PredefinedRecords map[query][]string
}

func NewConfig(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var c struct {
		Domain            string                         `yaml:"domain"`
		PredefinedRecords map[string]map[string][]string `yaml:"predefined_records"`
	}

	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, err
	}

	conf := &Config{
		Domain:            c.Domain,
		PredefinedRecords: make(map[query][]string),
	}

	for domain, replies := range c.PredefinedRecords {
		if !strings.HasSuffix(domain, ".") {
			domain = domain + "."
		}
		for queryType, rrs := range replies {
			t, ok := dns.StringToType[queryType]
			if !ok {
				return nil, fmt.Errorf("unknown query type: %v", queryType)
			}

			for _, rr := range rrs {
				if _, err := dns.NewRR(rr); err != nil {
					return nil, fmt.Errorf("failed to parse %#v for %v %v: %v",
						rr, queryType, domain, err)
				}
			}

			q := query{
				t:    t,
				name: domain,
			}

			conf.PredefinedRecords[q] = rrs
		}
	}

	return conf, nil
}

package main

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/miekg/dns"
	"gopkg.in/yaml.v2"
)

type PredefinedRecords map[query][]string

func (p *PredefinedRecords) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if *p == nil {
		*p = make(PredefinedRecords)
	}
	conf := map[string]map[string][]string{}
	if err := unmarshal(&conf); err != nil {
		return err
	}

	for domain, replies := range conf {
		if !strings.HasSuffix(domain, ".") {
			domain = domain + "."
		}
		for queryType, rrs := range replies {
			t, ok := dns.StringToType[queryType]
			if !ok {
				return fmt.Errorf("unknown query type: %v", queryType)
			}

			for _, rr := range rrs {
				if _, err := dns.NewRR(rr); err != nil {
					return fmt.Errorf("failed to parse %#v for %v %v: %v",
						rr, queryType, domain, err)
				}
			}

			q := query{
				t:    t,
				name: domain,
			}

			(*p)[q] = rrs
		}
	}
	return nil
}

type Config struct {
	Domain            string            `yaml:"domain"`
	PredefinedRecords PredefinedRecords `yaml:"predefined_records"`
	HTTP              struct {
		ListenOn []string `yaml:"listen_on"`
	} `yaml:"http"`
}

func NewConfig(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	conf := &Config{}
	if err := yaml.Unmarshal(data, conf); err != nil {
		return nil, err
	}

	return conf, nil
}

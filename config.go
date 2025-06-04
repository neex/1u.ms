package main

import (
	"fmt"
	"os"
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
				t:            t,
				name:         domain,
				nameForReply: domain,
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

func NewConfig(filenames []string) (*Config, error) {
	mergedConfig := make(map[interface{}]interface{})

	for _, filename := range filenames {
		data, err := os.ReadFile(filename)
		if err != nil {
			return nil, fmt.Errorf("error reading config file %s: %w", filename, err)
		}

		var currentConfig map[interface{}]interface{}
		if err := yaml.Unmarshal(data, &currentConfig); err != nil {
			return nil, fmt.Errorf("error parsing config file %s: %w", filename, err)
		}

		recursiveMerge(mergedConfig, currentConfig)
	}

	// Convert merged map back to YAML and then to Config struct
	mergedYAML, err := yaml.Marshal(mergedConfig)
	if err != nil {
		return nil, fmt.Errorf("error marshaling merged config: %w", err)
	}

	conf := &Config{}
	if err := yaml.Unmarshal(mergedYAML, conf); err != nil {
		return nil, fmt.Errorf("error unmarshaling merged config: %w", err)
	}

	return conf, nil
}

func recursiveMerge(dst, src map[interface{}]interface{}) {
	for k, v := range src {
		if existing, exists := dst[k]; exists {
			if existingMap, ok := existing.(map[interface{}]interface{}); ok {
				if srcMap, ok := v.(map[interface{}]interface{}); ok {
					recursiveMerge(existingMap, srcMap)
					continue
				}
			}
		}
		dst[k] = v
	}
}

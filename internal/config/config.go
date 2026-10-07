package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/goombaio/namegenerator"
	"gopkg.in/yaml.v3"
)

const defaultAlgorithm = "round_robin"

var namePattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

type Config struct {
	Listeners   []Listener     `yaml:"listeners"`
	Balancer    BalancerConfig `yaml:"balancer"`
	HealthCheck HealthCheck    `yaml:"health_check"`
	Backends    []Backend      `yaml:"backends"`
}

type Listener struct {
	Addr string `yaml:"addr"`
}

type BalancerConfig struct {
	Algorithm string `yaml:"algorithm"`
}

type HealthCheck struct {
	Interval time.Duration `yaml:"interval"`
	Timeout  time.Duration `yaml:"timeout"`
	Path     string        `yaml:"path"`
}

type Backend struct {
	Name        string      `yaml:"name"`
	Host        string      `yaml:"host"`
	Port        int         `yaml:"port"`
	HealthCheck HealthCheck `yaml:"health_check"`
}

func (b Backend) Addr() string {
	return net.JoinHostPort(b.Host, strconv.Itoa(b.Port))
}

func (b Backend) EffectiveHealthCheck(global HealthCheck) HealthCheck {
	hc := global
	if b.HealthCheck.Interval > 0 {
		hc.Interval = b.HealthCheck.Interval
	}
	if b.HealthCheck.Timeout > 0 {
		hc.Timeout = b.HealthCheck.Timeout
	}
	if b.HealthCheck.Path != "" {
		hc.Path = b.HealthCheck.Path
	}
	return hc
}

func Default() *Config {
	return &Config{
		Listeners:   []Listener{{Addr: ":8080"}},
		Balancer:    BalancerConfig{Algorithm: defaultAlgorithm},
		HealthCheck: HealthCheck{Interval: 3 * time.Second, Timeout: 2 * time.Second, Path: "/"},
	}
}

func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cfg := Default()
	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)
	if err := dec.Decode(cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	cfg.generateNames()
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config %s: %w", path, err)
	}
	return cfg, nil
}

func (c *Config) generateNames() {
	gen := namegenerator.NewNameGenerator(time.Now().UnixNano())
	seen := make(map[string]bool, len(c.Backends))
	for i := range c.Backends {
		b := &c.Backends[i]
		if b.Name == "" {
			for {
				name := gen.Generate()
				if !seen[name] {
					b.Name = name
					break
				}
			}
		}
		seen[b.Name] = true
	}
}

func (c *Config) Validate() error {
	var errs []error

	if len(c.Listeners) == 0 {
		errs = append(errs, errors.New("listeners: at least one listener is required"))
	}
	for i, l := range c.Listeners {
		if _, _, err := net.SplitHostPort(l.Addr); err != nil {
			errs = append(errs, fmt.Errorf("listeners[%d].addr %q: %w", i, l.Addr, err))
		}
	}
	if c.Balancer.Algorithm != defaultAlgorithm {
		errs = append(errs, fmt.Errorf("balancer.algorithm %q: only %q is supported", c.Balancer.Algorithm, defaultAlgorithm))
	}
	errs = append(errs, validateHealthCheck("health_check", c.HealthCheck, false)...)

	if len(c.Backends) == 0 {
		errs = append(errs, errors.New("backends: at least one backend is required"))
	}
	seen := make(map[string]bool, len(c.Backends))
	for i, b := range c.Backends {
		field := fmt.Sprintf("backends[%d]", i)
		if b.Host == "" {
			errs = append(errs, fmt.Errorf("%s.host: required", field))
		}
		if b.Port < 1 || b.Port > 65535 {
			errs = append(errs, fmt.Errorf("%s.port: must be between 1 and 65535, got %d", field, b.Port))
		}
		if !namePattern.MatchString(b.Name) {
			errs = append(errs, fmt.Errorf("%s.name %q: must be kebab-case", field, b.Name))
		}
		if seen[b.Name] {
			errs = append(errs, fmt.Errorf("%s.name %q: duplicate", field, b.Name))
		}
		seen[b.Name] = true
		errs = append(errs, validateHealthCheck(field+".health_check", b.HealthCheck, true)...)
	}
	return errors.Join(errs...)
}

func validateHealthCheck(field string, hc HealthCheck, inherit bool) []error {
	var errs []error
	if hc.Interval < 0 || (!inherit && hc.Interval == 0) {
		errs = append(errs, fmt.Errorf("%s.interval: must be positive", field))
	}
	if hc.Timeout < 0 || (!inherit && hc.Timeout == 0) {
		errs = append(errs, fmt.Errorf("%s.timeout: must be positive", field))
	}
	if hc.Path != "" && !strings.HasPrefix(hc.Path, "/") {
		errs = append(errs, fmt.Errorf("%s.path %q: must start with /", field, hc.Path))
	}
	if !inherit && hc.Path == "" {
		errs = append(errs, fmt.Errorf("%s.path: required", field))
	}
	return errs
}

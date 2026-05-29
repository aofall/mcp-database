package db

import "fmt"

type InstanceManager struct {
	instances map[string]DBDriver
}

func NewInstanceManager() *InstanceManager {
	return &InstanceManager{
		instances: make(map[string]DBDriver),
	}
}

func (m *InstanceManager) Register(alias string, cfg Config) error {
	driver, err := NewDriver(cfg)
	if err != nil {
		return err
	}
	m.instances[alias] = driver
	return nil
}

func (m *InstanceManager) Get(alias string) (DBDriver, error) {
	driver, exists := m.instances[alias]
	if !exists {
		return nil, fmt.Errorf("database instance '%s' not found. Available: %v", alias, m.ListAliases())
	}
	return driver, nil
}

func (m *InstanceManager) ListAliases() []string {
	keys := make([]string, 0, len(m.instances))
	for k := range m.instances {
		keys = append(keys, k)
	}
	return keys
}

func (m *InstanceManager) Close() error {
	var closeErr error
	for alias, driver := range m.instances {
		if err := driver.Close(); err != nil {
			if closeErr == nil {
				closeErr = fmt.Errorf("close database instance '%s': %w", alias, err)
			} else {
				closeErr = fmt.Errorf("%v; close database instance '%s': %w", closeErr, alias, err)
			}
		}
	}
	return closeErr
}

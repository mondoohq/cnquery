package execruntime

import "os"

type envProvider interface {
	Getenv(key string) string
	Setenv(key, value string) error
	Unsetenv(key string) error
}

type osEnvProvider struct{}

func (e *osEnvProvider) Getenv(key string) string {
	return os.Getenv(key)
}

func (e *osEnvProvider) Setenv(key, value string) error {
	return os.Setenv(key, value)
}

func (e *osEnvProvider) Unsetenv(key string) error {
	return os.Unsetenv(key)
}

func newMockEnvProvider() envProvider {
	mp := &mockEnvProvider{}
	mp.variables = make(map[string]string)
	return mp
}

type mockEnvProvider struct {
	variables map[string]string
}

func (e *mockEnvProvider) Getenv(key string) string {
	return e.variables[key]
}

func (e *mockEnvProvider) Setenv(key, value string) error {
	e.variables[key] = value
	return nil
}

func (e *mockEnvProvider) Unsetenv(key string) error {
	delete(e.variables, key)
	return nil
}

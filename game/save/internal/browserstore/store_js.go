//go:build js

package browserstore

import (
	"fmt"
	"strings"
	"syscall/js"
)

type Store struct {
	value js.Value
}

type Entry struct {
	Key   string
	Value string
}

func Open() (store Store, err error) {
	defer recoverJSError("access localStorage", &err)
	storage := js.Global().Get("localStorage")
	if storage.IsNull() || storage.IsUndefined() {
		return Store{}, fmt.Errorf("browser localStorage is unavailable")
	}
	return Store{value: storage}, nil
}

func (s Store) Set(key, value string) error {
	_, err := s.call("setItem", key, value)
	return err
}

func (s Store) Get(key string) (value string, found bool, err error) {
	result, err := s.call("getItem", key)
	if err != nil {
		return "", false, err
	}
	if result.IsNull() || result.IsUndefined() {
		return "", false, nil
	}
	return result.String(), true, nil
}

func (s Store) Remove(key string) error {
	_, err := s.call("removeItem", key)
	return err
}

func (s Store) Entries(prefix string) ([]Entry, error) {
	length, err := s.length()
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, length)
	for i := range length {
		keyValue, err := s.call("key", i)
		if err != nil {
			return nil, err
		}
		if keyValue.IsNull() || keyValue.IsUndefined() {
			continue
		}
		key := keyValue.String()
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		value, found, err := s.Get(key)
		if err != nil {
			return nil, err
		}
		if found {
			entries = append(entries, Entry{Key: key, Value: value})
		}
	}
	return entries, nil
}

func (s Store) length() (length int, err error) {
	defer recoverJSError("read localStorage length", &err)
	return s.value.Get("length").Int(), nil
}

func (s Store) call(method string, args ...any) (value js.Value, err error) {
	defer recoverJSError(method+" localStorage", &err)
	return s.value.Call(method, args...), nil
}

func recoverJSError(operation string, err *error) {
	if recovered := recover(); recovered != nil {
		*err = fmt.Errorf("%s: %v", operation, recovered)
	}
}

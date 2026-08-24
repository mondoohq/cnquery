// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"sort"
	"strings"

	"sync"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/llx"
	"go.mondoo.com/mql/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/providers-sdk/v1/resources"
)

// translationSource answers the compiler's downgrade lookups from the loaded
// providers (ADR 040 part 6).
//
// Reading a catalog means running the provider, because the translation is Go
// the provider author wrote and only the binary knows it. That is the trade this
// design makes for authoring flexibility, and it has a consequence worth
// stating: a host that compiles content for older readers needs the provider
// binaries installed, not just their schemas.
//
// Lookups are cached, including the misses: a provider that cannot be reached is
// asked once, not once per field.
type translationSource struct {
	coordinator ProvidersCoordinator

	mu          sync.Mutex
	cache       map[string][]*llx.TranslationStep
	unavailable map[string]struct{}
}

// NewTranslationSource returns a llx.TranslationSource backed by the loaded
// providers. It is what makes the downgrade mechanism reachable from any compile
// that has a coordinator, which is every compile that has a runtime.
func NewTranslationSource(coordinator ProvidersCoordinator) llx.TranslationSource {
	return &translationSource{
		coordinator: coordinator,
		cache:       map[string][]*llx.TranslationStep{},
		unavailable: map[string]struct{}{},
	}
}

func (s *translationSource) TranslationsFor(provider string) []*llx.TranslationStep {
	if s == nil || s.coordinator == nil || provider == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := resources.ProviderKey(provider)
	if steps, ok := s.cache[key]; ok {
		return steps
	}

	steps, err := s.fetch(provider)
	if err != nil {
		// Not being able to read a catalog is not a compile error. The content
		// still runs; readers on older versions of this provider just see the
		// affected fields as unavailable instead of translated.
		log.Debug().Err(err).Str("provider", provider).
			Msg("cannot read downgrade translations, compiling without them")
		s.unavailable[key] = struct{}{}
		s.cache[key] = nil
		return nil
	}

	s.cache[key] = steps
	return steps
}

func (s *translationSource) fetch(provider string) ([]*llx.TranslationStep, error) {
	// A translation catalog is a static property of the provider build, so this
	// needs the provider running but not connected to anything.
	running, err := s.coordinator.GetRunningProvider(provider, UpdateProvidersConfig{})
	if err != nil {
		return nil, err
	}
	res, err := running.Plugin.Translations(&plugin.TranslationsReq{})
	if err != nil {
		return nil, err
	}

	steps := make([]*llx.TranslationStep, 0, len(res.GetTranslations()))
	for _, t := range res.GetTranslations() {
		if t == nil || t.Block == nil {
			continue
		}
		steps = append(steps, &llx.TranslationStep{
			Resource:  t.Resource,
			Field:     t.Field,
			ChangedIn: t.ChangedIn,
			Block:     t.Block,
		})
	}
	return steps, nil
}

// Unavailable names the providers a catalog could not be read from, so a caller
// can report them once after compiling rather than once per field.
func (s *translationSource) Unavailable() UnavailableProviders {
	s.mu.Lock()
	defer s.mu.Unlock()

	res := make(UnavailableProviders, 0, len(s.unavailable))
	for name := range s.unavailable {
		res = append(res, name)
	}
	sort.Strings(res)
	return res
}

// UnavailableProviders names providers a catalog could not be read from.
type UnavailableProviders []string

// Warning renders the one aggregate message a caller should log when some
// providers could not supply translations. Deliberately a single line naming the
// providers: one message per affected field would bury the point in a scan that
// touches thousands.
func (u UnavailableProviders) Warning() string {
	if len(u) == 0 {
		return ""
	}
	return "compiled without downgrade fallbacks for " + strings.Join(u, ", ") +
		" (provider binary not available); content will still run, but readers on older" +
		" versions of these providers will see affected fields as unavailable"
}

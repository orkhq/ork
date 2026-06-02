package orchestration

import (
	"fmt"

	"ork/internal/adapters"
	manifestcore "ork/pkg/manifest/core"
)

func validateComponentSource(component *manifestcore.Component, adapter adapters.Adapter) error {
	if _, err := component.Source.Validate(); err != nil {
		return fmt.Errorf("component %q has an invalid source configuration: %w", component.Name, err)
	}

	if !adapter.SupportedSources().SatisfiedBy(component.Source) {
		return fmt.Errorf("component %q source type %q is not supported by adapter %q. Supported source types are: %s",
			component.Name, component.Source.Type(), component.Type, adapter.SupportedSources().String())
	}

	return nil
}

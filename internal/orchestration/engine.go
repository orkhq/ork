package orchestration

import (
	"fmt"

	"orch.io/internal/adapters"
	manifestcore "orch.io/pkg/manifest/core"
)

func RunUp(m *manifestcore.Manifest) error {
	fmt.Printf("Starting sandbox: %s\n", m.Metadata.ID)

	for _, c := range m.Components {
		fmt.Printf("→ Applying component: %s (%s)\n", c.Name, c.Type)
		adapter, err := adapters.Get(c.Type)
		if err != nil {
			return err
		}
		if err := adapter.Apply(c); err != nil {
			return fmt.Errorf("component %s failed: %w", c.Name, err)
		}
	}

	fmt.Printf("✅ Sandbox created successfully\n")
	return nil
}

func RunDown(m *manifestcore.Manifest) error {
	fmt.Printf("Tearing down sandbox: %s\n", m.Metadata.ID)

	for _, c := range m.Components {
		fmt.Printf("→ Destroying component: %s\n", c.Name)
		adapter, err := adapters.Get(c.Type)
		if err != nil {
			return err
		}
		if err := adapter.Destroy(c); err != nil {
			return fmt.Errorf("component %s destroy failed: %w", c.Name, err)
		}
	}

	fmt.Printf("🧹 Sandbox torn down\n")
	return nil
}

package adapters

import (
	"fmt"
	"sort"
	"strings"

	"orch.io/pkg/events"
	manifestcore "orch.io/pkg/manifest/core"
)

var explicitProviderEnvKeys = map[string]struct{}{
	"AWS_ACCESS_KEY_ID":     {},
	"AWS_SECRET_ACCESS_KEY": {},
	"AWS_SESSION_TOKEN":     {},
}

var explicitTerraformCredentialVarKeys = map[string]struct{}{
	"access_key":            {},
	"aws_access_key_id":     {},
	"aws_secret_access_key": {},
	"aws_session_token":     {},
	"secret_key":            {},
}

func detectExplicitProviderCredentialEnv(env map[string]string) []string {
	var refs []string
	for key := range env {
		normalized := strings.ToUpper(key)
		if _, ok := explicitProviderEnvKeys[normalized]; ok {
			refs = append(refs, "env."+key)
		}
	}
	sort.Strings(refs)
	return refs
}

func detectExplicitTerraformCredentialVars(vars map[string]string) []string {
	var refs []string
	for key := range vars {
		normalized := strings.ToLower(key)
		if _, ok := explicitTerraformCredentialVarKeys[normalized]; ok {
			refs = append(refs, "vars."+key)
		}
	}
	sort.Strings(refs)
	return refs
}

func providerCredentialWarning(c *manifestcore.Component, refs []string) events.Event {
	sort.Strings(refs)
	return events.Event{
		Type: events.EventWarning,
		Message: fmt.Sprintf(
			"Component passes provider credentials through manifest values (%s). This component cannot be reliably torn down by Orch without access to those values.",
			strings.Join(refs, ", "),
		),
		Hint:      "Prefer ambient provider auth on the runner, such as instance profiles, workload identity, or a runner-local cloud profile.",
		Adapter:   c.Type,
		Runner:    c.Runner,
		Component: c.Name,
	}
}

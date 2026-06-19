package runners

import "github.com/go-viper/mapstructure/v2"

// ProviderConfig is the interface for cloud provider credential configurations
// that runners inject as environment variables into execution contexts.
type ProviderConfig interface {
	IsConfigured() bool
	BuildEnvVars() map[string]string
}

// AWSProviderConfig holds AWS credentials and region for injection into a
// runner's execution environment.
type AWSProviderConfig struct {
	Region          string `mapstructure:"region"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
}

func (a *AWSProviderConfig) IsConfigured() bool {
	return a.Region != "" && a.AccessKeyID != "" && a.SecretAccessKey != ""
}

func (a *AWSProviderConfig) BuildEnvVars() map[string]string {
	envs := make(map[string]string)
	if a.Region != "" {
		envs["AWS_REGION"] = a.Region
	}
	if a.AccessKeyID != "" {
		envs["AWS_ACCESS_KEY_ID"] = a.AccessKeyID
	}
	if a.SecretAccessKey != "" {
		envs["AWS_SECRET_ACCESS_KEY"] = a.SecretAccessKey
	}
	return envs
}

// AggregatedProviderConfig collects all provider configurations for a runner
// and produces a merged set of environment variables.
type AggregatedProviderConfig struct {
	AWS *AWSProviderConfig `mapstructure:"aws,omitempty"`
}

func (a *AggregatedProviderConfig) GetAllEnvVars() map[string]string {
	envs := make(map[string]string)
	if a.AWS != nil && a.AWS.IsConfigured() {
		for k, v := range a.AWS.BuildEnvVars() {
			envs[k] = v
		}
	}

	return envs
}

// RetrieveProviderConfigForRunner decodes the raw provider map from a runner
// manifest and returns the aggregated environment variables and the parsed
// provider config.
func RetrieveProviderConfigForRunner(config map[string]interface{}) (map[string]string, *AggregatedProviderConfig, error) {
	var pcfg AggregatedProviderConfig
	if err := mapstructure.Decode(config, &pcfg); err != nil {
		return nil, nil, err
	}

	return pcfg.GetAllEnvVars(), &pcfg, nil
}

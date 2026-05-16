package state

import "context"

type Backend interface {
	Load(ctx context.Context, envID string) (*OrchState, error)
	Save(ctx context.Context, envID string, state *OrchState) error
	Exists(ctx context.Context, envID string) (bool, error)
	Delete(ctx context.Context, envID string) error
	SaveArtifact(ctx context.Context, envID string, componentName string, artifact Artifact, sourcePath string) error
	RestoreArtifact(ctx context.Context, envID string, componentName string, artifact Artifact, destinationPath string) error
	ArtifactExists(ctx context.Context, envID string, componentName string, artifact Artifact) (bool, error)
}

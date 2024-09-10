package reg

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	oras "oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

type GroupState struct {
	Name     string   `json:"name"             yaml:"name"`
	Registry string   `json:"registry"         yaml:"registry"`
	Images   []Images `json:"images,omitempty" yaml:"images,omitempty"`
}

type SatelliteState struct {
	Name     string   `json:"name"             yaml:"name"`
	Groups   []string `json:"groups"           yaml:"groups"`
	Registry string   `json:"registry"         yaml:"registry"`
	Images   []Images `json:"images,omitempty" yaml:"images,omitempty"`
}

// TO-DO: Auth for specific images experimental
// type Auth struct {
// 	Username string `json:"username" yaml:"username"`
// 	Password string `json:"password" yaml:"password"`
// }

type Images struct {
	Registry   string `json:"registry"         yaml:"registry"`
	Repository string `json:"repository"       yaml:"repository"`
	Tag        string `json:"tag"              yaml:"tag"`
	Digest     string `json:"digest,omitempty" yaml:"digest,omitempty"`

	// Auth       Auth `json:"auth,omitempty" yaml:"auth,omitempty"`
}

type State interface {
	Marshal() ([]byte, error)
	GetName() string
	GetRegistry() string
	isGroupState() bool
}

func (g GroupState) Marshal() ([]byte, error) {
	return json.MarshalIndent(g, "", "  ")
}

func (g GroupState) GetName() string {
	return g.Name
}

func (g GroupState) GetRegistry() string {
	return g.Registry
}

func (g GroupState) isGroupState() bool {
	return true
}

func (s SatelliteState) Marshal() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}

func (s SatelliteState) GetName() string {
	return s.Name
}

func (s SatelliteState) GetRegistry() string {
	return s.Registry
}

func (s SatelliteState) isGroupState() bool {
	return false
}

// Creates State artifact & push to registry.
func PushStateArtifact(ctx context.Context, State State) error {
	// generate the json content from state
	jsonData, err := State.Marshal()
	if err != nil {
		return fmt.Errorf("error in marshaling into json: %v", err)
	}

	fmt.Println(" --- JSON (Start) ---")
	fmt.Println(string(jsonData))
	fmt.Println(" --- JSON (END) ---")

	// Create temp file in /tmp/
	tmpFile, err := os.CreateTemp("/tmp/", "state-*.json")
	if err != nil {
		return fmt.Errorf("error creating tempDir: %v", err)
	}
	defer os.Remove(tmpFile.Name()) // Clean up the file after execution

	if _, err := tmpFile.Write(jsonData); err != nil {
		return fmt.Errorf("error writing data to file: %v", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("error in closing file: %v", err)
	}

	// Create a file store
	fs, err := file.New("/tmp/")
	if err != nil {
		return fmt.Errorf("error creating file store: %v", err)
	}
	defer fs.Close()

	// Add files to the file store
	mediaType := "application/vnd.test.file.v1+json"
	fileNames := []string{fmt.Sprintf("%s", tmpFile.Name())}
	fileDescriptors := make([]v1.Descriptor, 0, len(fileNames))
	for _, name := range fileNames {
		fileDescriptor, err := fs.Add(ctx, name, mediaType, "")
		if err != nil {
			return fmt.Errorf("error in fileDescriptor: %v", err)
		}
		fileDescriptors = append(fileDescriptors, fileDescriptor)
		fmt.Printf("file descriptor for %s: %v\n", name, fileDescriptor)
	}

	// Pack the files and tag the packed manifest
	artifactType := "application/vnd.test.artifact.v1+json"
	opts := oras.PackManifestOptions{
		Layers: fileDescriptors,
	}
	manifestDescriptor, err := oras.PackManifest(
		ctx,
		fs,
		oras.PackManifestVersion1_1,
		artifactType,
		opts,
	)
	if err != nil {
		return fmt.Errorf("error in manifestDescriptor: %v", err)
	}
	fmt.Println("manifest descriptor:", manifestDescriptor)

	tag := "latest"
	if err = fs.Tag(ctx, manifestDescriptor, tag); err != nil {
		return fmt.Errorf("error in tag descriptor: %v", err)
	}

	var repo *remote.Repository

	// Connect to a remote repository
	if State.isGroupState() {
		repo, err = remote.NewRepository(
			fmt.Sprintf("%s/satellite/groups/%s", State.GetRegistry(), State.GetName()),
		)
		if err != nil {
			return fmt.Errorf("error: unable to establish a client: %v", err)
		}
	} else {
		repo, err = remote.NewRepository(
			fmt.Sprintf("%s/satellite/satellites/%s", State.GetRegistry(), State.GetName()),
		)
		if err != nil {
			return fmt.Errorf("error: unable to establish a client: %v", err)
		}
	}

	repo.Client = &auth.Client{
		Client: retry.DefaultClient,
		Cache:  auth.NewCache(),
		Credential: auth.StaticCredential(State.GetRegistry(), auth.Credential{
			// Username: "admin",
			// Password: "Harbor12345",
			Username: os.Getenv("HARBOR_USERNAME"),
			Password: os.Getenv("HARBOR_PASSWORD"),
		}),
	}

	_, err = repo.Exists(ctx, manifestDescriptor)
	if err != nil {
		fmt.Println("error checking with existing repo")
		return fmt.Errorf(
			"please create project named 'satellite' for storing satellite state artifact: %v",
			err,
		)
	}

	// Copy State to remote repository
	_, err = oras.Copy(ctx, fs, tag, repo, tag, oras.DefaultCopyOptions)
	if err != nil {
		return fmt.Errorf("error in pushing state artifact: %v", err)
	}

	return nil
}
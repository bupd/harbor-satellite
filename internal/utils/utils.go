package utils

import (
	"context"
	"fmt"
	"net/url"
	"os/signal"
	"strings"
	"syscall"

	"github.com/container-registry/harbor-satellite/pkg/config"
	"github.com/rs/zerolog"
)

// / HandleOwnRegistry handles the own registry address and port and sets the Zot URL
func HandleOwnRegistry(cm *config.ConfigManager) error {
	remoteRegistryURL := string(cm.GetLocalRegistryURL())
	_, err := url.Parse(remoteRegistryURL)
	if err != nil {
		return fmt.Errorf("error parsing URL: %w", err)
	}
	cm.With(config.SetLocalRegistryURL(FormatRegistryURL(remoteRegistryURL)))
	return nil
}

// Helper function to determine if input is a valid URL
func IsValidURL(input string) bool {
	parsedURL, err := url.Parse(input)
	return err == nil && parsedURL.Scheme != ""
}

func GetRepositoryAndImageNameFromArtifact(repository string) (string, string, error) {
	parts := strings.Split(repository, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid repository format: %s. Expected format: repo/image", repository)
	}

	repo := parts[0]
	image := strings.Join(parts[1:], "/")
	return repo, image, nil
}

func SetupContext(context context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := signal.NotifyContext(context, syscall.SIGTERM, syscall.SIGINT)
	return ctx, cancel
}

// FormatRegistryURL formats the registry URL by trimming the "https://" or "http://" prefix if present
func FormatRegistryURL(url string) string {
	// Trim the "https://" or "http://" prefix if present
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	return url
}

func HandleNewConfigWarnings(log *zerolog.Logger, warnings []string) {
	log.Info().Msg("The newly fetched remote config has the following warnings")
	HandleWarnings(log, warnings)
}

func HandleWarnings(log *zerolog.Logger, warnings []string) {
	for i := range warnings {
		log.Warn().Msg(string(warnings[i]))
	}
}
